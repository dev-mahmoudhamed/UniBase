package handlers

import (
	"db-server/db/cache"
	"db-server/db/providers"
	"db-server/db/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	gocache "github.com/patrickmn/go-cache"
)

type QueryRequest struct {
	QueryID   string `json:"queryId" binding:"required"`
	SessionID string `json:"sessionId" binding:"required"`
	Query     string `json:"query" binding:"required"`
}

var (
	ongoingQueries = make(map[string]chan struct{})
	ongoingMutex   sync.Mutex
)

func getDoneChan(queryID string) chan struct{} {
	ongoingMutex.Lock()
	defer ongoingMutex.Unlock()
	if ch, ok := ongoingQueries[queryID]; ok {
		return ch
	}
	ch := make(chan struct{})
	ongoingQueries[queryID] = ch
	return ch
}

func removeDoneChan(queryID string) {
	ongoingMutex.Lock()
	defer ongoingMutex.Unlock()
	delete(ongoingQueries, queryID)
}

// ExecuteQuery handles query execution with caching and polling support
func ExecuteQuery(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get session config early to validate session and build scope
	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Use SessionID in the cache key to ensure queries are scoped per session
	cacheKey := req.SessionID + ":" + req.QueryID
	queryCache := cache.GetCache()

	// Check if already in cache and finished
	if val, found := queryCache.Get(cacheKey); found && val != nil {
		if errStr, ok := val.(string); ok && len(errStr) > 5 && errStr[:5] == "ERR: " {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"queryId": req.QueryID,
				"error":   errStr[5:],
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"queryId": req.QueryID,
			"results": val,
		})
		return
	}

	doneChan := getDoneChan(cacheKey)

	// If not in cache at all, start execution
	if _, found := queryCache.Get(cacheKey); !found {
		queryCache.Set(cacheKey, nil, gocache.DefaultExpiration)

		go func() {
			defer removeDoneChan(cacheKey)
			defer close(doneChan)

			provider, err := providers.GetProvider(config.Provider)
			if err != nil {
				queryCache.Set(cacheKey, "ERR: "+err.Error(), gocache.DefaultExpiration)
				return
			}

			results, err := provider.ExecuteQuery(*config, req.Query)
			if err != nil {
				queryCache.Set(cacheKey, "ERR: "+err.Error(), gocache.DefaultExpiration)
				return
			}

			// Store result in cache
			queryCache.Set(cacheKey, results, gocache.DefaultExpiration)
		}()
	}

	// Wait for query to finish or timeout
	select {
	case <-doneChan:
		// Re-fetch from cache to see if it was success or error
		if val, found := queryCache.Get(cacheKey); found && val != nil {
			if errStr, ok := val.(string); ok && len(errStr) > 5 && errStr[:5] == "ERR: " {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  "error",
					"queryId": req.QueryID,
					"error":   errStr[5:],
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"status":  "success",
					"queryId": req.QueryID,
					"results": val,
				})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Query finished but no result found"})
		}
	case <-time.After(30 * time.Second):
		c.JSON(http.StatusAccepted, gin.H{
			"message": "query timeout",
			"queryId": req.QueryID,
		})
	}
}
