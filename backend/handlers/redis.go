package handlers

import (
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRedisKeyValue handles requests to get a Redis key's value
func GetRedisKeyValue(c *gin.Context) {
	var req models.RedisKeyValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if config.Provider != "redis" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This endpoint is only available for Redis connections"})
		return
	}

	// Set the database index from the request
	if req.Database != "" {
		if strings.HasPrefix(req.Database, "db") {
			config.Database = strings.TrimPrefix(req.Database, "db")
		} else {
			config.Database = req.Database
		}
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	redisProvider, ok := provider.(*providers.RedisProvider)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Redis provider"})
		return
	}

	result, err := redisProvider.GetKeyValue(*config, req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateRedisKeyValue handles requests to update a Redis key's value
func UpdateRedisKeyValue(c *gin.Context) {
	var req models.RedisKeyValueUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if config.Provider != "redis" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This endpoint is only available for Redis connections"})
		return
	}

	if req.Database != "" {
		if strings.HasPrefix(req.Database, "db") {
			config.Database = strings.TrimPrefix(req.Database, "db")
		} else {
			config.Database = req.Database
		}
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	redisProvider, ok := provider.(*providers.RedisProvider)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Redis provider"})
		return
	}

	if err := redisProvider.SetKeyValue(*config, req.OldKey, req.NewKey, req.Value, req.TTL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Key updated successfully"})
}

// DeleteRedisKey handles requests to delete a Redis key
func DeleteRedisKey(c *gin.Context) {
	var req models.RedisKeyDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if config.Provider != "redis" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This endpoint is only available for Redis connections"})
		return
	}

	if req.Database != "" {
		if strings.HasPrefix(req.Database, "db") {
			config.Database = strings.TrimPrefix(req.Database, "db")
		} else {
			config.Database = req.Database
		}
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	redisProvider, ok := provider.(*providers.RedisProvider)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Redis provider"})
		return
	}

	if err := redisProvider.DeleteKey(*config, req.Key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Key deleted successfully"})
}
