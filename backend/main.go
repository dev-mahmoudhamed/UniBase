package main

import (
	"db-server/db/cache"
	"db-server/db/registry"
	"db-server/db/session"
	"db-server/handlers"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultAddr   = "127.0.0.1:5000"
	sessionMaxAge = 24 * time.Hour
)

func main() {
	reg := registry.GetRegistry()
	if err := reg.LoadFromFile("./config/providers.json"); err != nil {
		log.Fatalf("Failed to load provider registry: %v", err)
	}
	log.Printf("Loaded %d database providers", len(reg.GetSupportedProviderIDs()))

	cache.InitQueryCache()

	// Drop stored credentials for sessions older than a day, now and every hour.
	go func() {
		for {
			session.CleanupExpiredSessions(sessionMaxAge)
			time.Sleep(time.Hour)
		}
	}()

	// No CORS headers: the UI is served from this same origin (and through the
	// dev proxy in `ng serve`), so other websites cannot call this API.
	r := gin.Default()

	distPath := "../frontend/dist/frontend/browser"
	indexFile := filepath.Join(distPath, "index.html")

	r.Static("/assets", filepath.Join(distPath, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))

	api := r.Group("/api")
	{
		providers := api.Group("/providers")
		{
			providers.GET("", handlers.GetProviders)
		}

		connection := api.Group("/connection")
		{
			connection.POST("/test", handlers.TestConnection)
			connection.POST("/initialize", handlers.InitializeConnection)
		}

		explorer := api.Group("/explorer")
		{
			explorer.POST("/children", handlers.GetExplorerChildren)
			explorer.POST("/collection-data", handlers.GetCollectionData)
			explorer.POST("/collection-metadata", handlers.GetCollectionMetadata)
		}

		query := api.Group("/query")
		{
			query.POST("/execute", handlers.ExecuteQuery)
			query.POST("/cancel", handlers.CancelQuery)
		}

		redis := api.Group("/redis")
		{
			redis.POST("/key/get", handlers.GetRedisKeyValue)
			redis.POST("/key/update", handlers.UpdateRedisKeyValue)
			redis.POST("/key/delete", handlers.DeleteRedisKey)
		}

		mongo := api.Group("/mongo")
		{
			mongo.POST("/document/update", handlers.UpdateMongoDocument)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join(distPath, c.Request.URL.Path)
		if info, err := os.Stat(path); err == nil && !info.IsDir() && filepath.Base(path) != "index.html" {
			c.File(path)
			return
		}
		// index.html names the hashed bundles, so browsers must re-check it after
		// every rebuild; otherwise they keep running the previous version.
		c.Header("Cache-Control", "no-cache")
		c.File(indexFile)
	})

	// Localhost only by default: UniBase holds database credentials and has no login.
	// Set UNIBASE_ADDR (e.g. 0.0.0.0:5000) to expose it deliberately.
	addr := os.Getenv("UNIBASE_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	log.Printf("Server starting on http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
