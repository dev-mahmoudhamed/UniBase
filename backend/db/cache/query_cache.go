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

func InitQueryCache() {
	QueryCache = cache.New(DefaultExpiration, CleanupInterval)
}

func GetCache() *cache.Cache {
	return QueryCache
}
