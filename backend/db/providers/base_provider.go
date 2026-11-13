package providers

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"db-server/models"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	connectTimeout = 10 * time.Second
	poolIdleTTL    = 15 * time.Minute
)

type BaseProvider struct{}

func (b *BaseProvider) GetCollectionMetadata(config models.ConnectionConfig, collectionName string, context map[string]string) (models.CollectionMetadata, error) {
	return models.CollectionMetadata{}, fmt.Errorf("metadata retrieval not implemented for this provider")
}

// Connection pools are shared across requests, keyed by connection string, and
// closed after poolIdleTTL without use. Opening a pool per request cost a full
// TCP/TLS/login round trip on every tree click.
type pooled struct {
	db       *sql.DB
	client   *mongo.Client
	rdb      *redis.Client
	lastUsed time.Time
}

var (
	poolMu sync.Mutex
	pools  = map[string]*pooled{}
)

func init() {
	go func() {
		for range time.Tick(time.Minute) {
			closeIdlePools()
		}
	}()
}

func closeIdlePools() {
	poolMu.Lock()
	defer poolMu.Unlock()
	for key, p := range pools {
		if time.Since(p.lastUsed) < poolIdleTTL {
			continue
		}
		if p.db != nil {
			_ = p.db.Close()
		}
		if p.client != nil {
			_ = p.client.Disconnect(context.Background())
		}
		if p.rdb != nil {
			_ = p.rdb.Close()
		}
		delete(pools, key)
	}
}

// sqlPool returns the shared *sql.DB for this driver and DSN. Callers must not close it.
func sqlPool(driver, dsn string) (*sql.DB, error) {
	key := driver + "|" + dsn
	poolMu.Lock()
	defer poolMu.Unlock()
	if p, ok := pools[key]; ok {
		p.lastUsed = time.Now()
		return p.db, nil
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open failed: %w", err)
	}
	db.SetConnMaxIdleTime(5 * time.Minute)
	pools[key] = &pooled{db: db, lastUsed: time.Now()}
	return db, nil
}

// pingSQL checks that the server is reachable and the credentials work.
func pingSQL(driver, dsn string) error {
	db, err := sqlPool(driver, dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

// mongoClient returns the shared client for this URI. Callers must not disconnect it.
func mongoClient(uri string) (*mongo.Client, error) {
	key := "mongodb|" + uri
	poolMu.Lock()
	defer poolMu.Unlock()
	if p, ok := pools[key]; ok {
		p.lastUsed = time.Now()
		return p.client, nil
	}
	opts := options.Client().ApplyURI(uri).SetConnectTimeout(connectTimeout).SetServerSelectionTimeout(connectTimeout)
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	pools[key] = &pooled{client: client, lastUsed: time.Now()}
	return client, nil
}

// redisClient returns the shared client for these options. Callers must not close it.
func redisClient(opts *redis.Options) *redis.Client {
	key := fmt.Sprintf("redis|%s|%s|%s|%d", opts.Addr, opts.Username, opts.Password, opts.DB)
	poolMu.Lock()
	defer poolMu.Unlock()
	if p, ok := pools[key]; ok {
		p.lastUsed = time.Now()
		return p.rdb
	}
	opts.DialTimeout = connectTimeout
	rdb := redis.NewClient(opts)
	pools[key] = &pooled{rdb: rdb, lastUsed: time.Now()}
	return rdb
}
