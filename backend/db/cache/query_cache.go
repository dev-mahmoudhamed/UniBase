package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	DefaultExpiration = 10 * time.Minute
	CleanupInterval   = 5 * time.Minute
)

var QueryCache *cache.Cache

// InitQueryCache initializes the query cache with default settings
func InitQueryCache() {
	QueryCache = cache.New(DefaultExpiration, CleanupInterval)
}

// GetCache returns the query cache instance
func GetCache() *cache.Cache {
	return QueryCache
}
