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

	// Serve static files
	r.Static("/assets", filepath.Join(distPath, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))

	api := r.Group("/api")
	{
		api.GET("/providers", handlers.GetProviders)
		api.GET("/providers/:id", handlers.GetProviderByID)
		api.POST("/providers/validate", handlers.ValidateProviderConfig)
	}

	r.POST("/connection/test", handlers.TestConnection)
	r.POST("/connection/initialize", handlers.InitializeConnection)
	r.POST("/query/execute", handlers.ExecuteQuery)
	r.POST("/explorer/children", handlers.GetExplorerChildren)
	r.POST("/explorer/collection-data", handlers.GetCollectionData)

	// Redis key CRUD
	r.POST("/redis/key/get", handlers.GetRedisKeyValue)
	r.POST("/redis/key/update", handlers.UpdateRedisKeyValue)
	r.POST("/redis/key/delete", handlers.DeleteRedisKey)

	// SPA fallback (Angular routing)
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join(distPath, c.Request.URL.Path)

		if _, err := os.Stat(path); err == nil {
			c.File(path)
			return
		}

		c.File(indexFile)
	})

	// Start server
	log.Println("Server starting on :5000")
	r.Run(":5000")
}
