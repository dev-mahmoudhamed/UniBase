package main

import (
	"db-server/db/cache"
	"db-server/db/registry"
	"db-server/db/session"
	"db-server/handlers"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize provider registry
	reg := registry.GetRegistry()
	if err := reg.LoadFromFile("./config/providers.json"); err != nil {
		log.Fatalf("Failed to load provider registry: %v", err)
	}
	log.Printf("Loaded %d database providers", len(reg.GetSupportedProviderIDs()))

	// Initialize database components
	session.InitSessionStore()
	cache.InitQueryCache()

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	distPath := "../frontend/dist/frontend/browser"
	indexFile := filepath.Join(distPath, "index.html")

	// Static assets
	r.Static("/assets", filepath.Join(distPath, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))

	// ── API routes ────────────────────────────────────────────────────────────
	api := r.Group("/api")
	{
		// Providers — database provider metadata & validation
		providers := api.Group("/providers")
		{
			providers.GET("", handlers.GetProviders)
			providers.GET("/:id", handlers.GetProviderByID)
			providers.POST("/validate", handlers.ValidateProviderConfig)
		}

		// Connection — test & initialize database connections
		connection := api.Group("/connection")
		{
			connection.POST("/test", handlers.TestConnection)
			connection.POST("/initialize", handlers.InitializeConnection)
		}

		// Query — execute SQL / NoSQL queries
		query := api.Group("/query")
		{
			query.POST("/execute", handlers.ExecuteQuery)
		}

		// Explorer — lazy-load the object explorer tree
		explorer := api.Group("/explorer")
		{
			explorer.POST("/children", handlers.GetExplorerChildren)
			explorer.POST("/collection-data", handlers.GetCollectionData)
		}

		// Redis — key-level CRUD operations
		redis := api.Group("/redis")
		{
			redis.POST("/key/get", handlers.GetRedisKeyValue)
			redis.POST("/key/update", handlers.UpdateRedisKeyValue)
			redis.POST("/key/delete", handlers.DeleteRedisKey)
		}

		// MongoDB — document-level operations
		mongo := api.Group("/mongo")
		{
			mongo.POST("/document/update", handlers.UpdateMongoDocument)
		}
	}

	// SPA fallback — must come last so Angular routing works
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join(distPath, c.Request.URL.Path)
		if _, err := os.Stat(path); err == nil {
			c.File(path)
			return
		}
		c.File(indexFile)
	})

	log.Println("Server starting on :5000")
	r.Run(":5000")
}
