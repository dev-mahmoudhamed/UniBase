package handlers

import (
	"context"
	"db-server/db/cache"
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	gocache "github.com/patrickmn/go-cache"
)

// QueryRequest carries one execution. QueryID must be unique per run (the client
// generates a fresh ID for every Run click); it is only reused while polling.
type QueryRequest struct {
	QueryID   string `json:"queryId" binding:"required"`
	SessionID string `json:"sessionId" binding:"required"`
	Query     string `json:"query" binding:"required"`
}

// queryOutcome is what a finished run leaves in the cache for the poller.
type queryOutcome struct {
	Result models.QueryResult
	Err    string
}

const (
	queryWaitTimeout = 30 * time.Second
	queryMaxDuration = 10 * time.Minute
)

// runningQuery is a run in progress: done closes when it finishes, cancel stops it.
type runningQuery struct {
	done   chan struct{}
	cancel context.CancelFunc
}

var (
	// runningQueries is guarded by runningMu together with the cache check, so a
	// run can't finish between "not in cache" and "not running" and leave a waiter hanging.
	runningQueries = make(map[string]*runningQuery)
	runningMu      sync.Mutex
)

// ExecuteQuery starts a run (or attaches to one in progress), waits up to 30s,
// and otherwise answers 202 so the client can poll with the same queryId.
func ExecuteQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		Unauthorized(c, "Session expired or invalid")
		return
	}

	key := req.SessionID + ":" + req.QueryID
	queryCache := cache.GetCache()

	runningMu.Lock()
	if val, found := queryCache.Get(key); found {
		runningMu.Unlock()
		respondOutcome(c, req.QueryID, key, val)
		return
	}
	run, running := runningQueries[key]
	if !running {
		ctx, cancel := context.WithTimeout(context.Background(), queryMaxDuration)
		run = &runningQuery{done: make(chan struct{}), cancel: cancel}
		runningQueries[key] = run
		go runQuery(ctx, key, *config, req.Query, run)
	}
	runningMu.Unlock()

	select {
	case <-run.done:
		val, _ := queryCache.Get(key)
		respondOutcome(c, req.QueryID, key, val)
	case <-time.After(queryWaitTimeout):
		Accepted(c, "query timeout", gin.H{"queryId": req.QueryID})
	}
}

func runQuery(ctx context.Context, key string, config models.ConnectionConfig, query string, run *runningQuery) {
	defer run.cancel()
	var outcome queryOutcome
	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		outcome.Err = err.Error()
	} else if result, err := provider.ExecuteQuery(ctx, config, query); err != nil {
		switch ctx.Err() {
		case context.Canceled:
			outcome.Err = "Query cancelled"
		case context.DeadlineExceeded:
			outcome.Err = "Query stopped after " + queryMaxDuration.String()
		default:
			outcome.Err = err.Error()
		}
	} else {
		outcome.Result = result
	}

	runningMu.Lock()
	cache.GetCache().Set(key, outcome, gocache.DefaultExpiration)
	delete(runningQueries, key)
	runningMu.Unlock()
	close(run.done)
}

type cancelRequest struct {
	QueryID   string `json:"queryId" binding:"required"`
	SessionID string `json:"sessionId" binding:"required"`
}

// CancelQuery stops a running query. The waiting request then answers "Query cancelled".
func CancelQuery(c *gin.Context) {
	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}
	runningMu.Lock()
	run, ok := runningQueries[req.SessionID+":"+req.QueryID]
	runningMu.Unlock()
	if ok {
		run.cancel()
	}
	Success(c, "Cancel requested", nil)
}

// respondOutcome writes the finished run and drops it from the cache: each run ID
// is delivered once, so results never outlive the request that asked for them.
func respondOutcome(c *gin.Context, queryID, key string, val interface{}) {
	outcome, ok := val.(queryOutcome)
	if !ok {
		InternalError(c, nil, "Query finished but no result found")
		return
	}
	cache.GetCache().Delete(key)
	if outcome.Err != "" {
		InternalError(c, nil, outcome.Err, gin.H{"queryId": queryID})
		return
	}
	Success(c, "Query executed successfully", gin.H{
		"queryId": queryID,
		"results": outcome.Result,
	})
}
