package handlers

import (
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"
	"strings"

	"github.com/gin-gonic/gin"
)


// GetRedisKeyValue handles requests to get a Redis key's value
func GetRedisKeyValue(c *gin.Context) {
	var req models.RedisKeyValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		Unauthorized(c, "Session expired or invalid")
		return
	}

	if config.Provider != "redis" {
		BadRequest(c, "This endpoint is only available for Redis connections")
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
		BadRequest(c, "Provider error")
		return
	}

	redisProvider, ok := provider.(*providers.RedisProvider)
	if !ok {
		BadRequest(c, "Invalid Redis provider")
		return
	}

	result, err := redisProvider.GetKeyValue(*config, req.Key)
	if err != nil {
		InternalError(c, err, "Failed to retrieve key value")
		return
	}

	Success(c, "Key value retrieved", result)
}

// UpdateRedisKeyValue handles requests to update a Redis key's value
func UpdateRedisKeyValue(c *gin.Context) {
	var req models.RedisKeyValueUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		Unauthorized(c, "Session expired or invalid")
		return
	}

	if config.Provider != "redis" {
		BadRequest(c, "This endpoint is only available for Redis connections")
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
		BadRequest(c, "Provider error")
		return
	}

	redisProvider, ok := provider.(*providers.RedisProvider)
	if !ok {
		BadRequest(c, "Invalid Redis provider")
		return
	}

	if err := redisProvider.SetKeyValue(*config, req.OldKey, req.NewKey, req.Value, req.TTL); err != nil {
		InternalError(c, err, "Failed to update key")
		return
	}

	Success(c, "Key updated successfully", nil)
}

// DeleteRedisKey handles requests to delete a Redis key
func DeleteRedisKey(c *gin.Context) {
	var req models.RedisKeyDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		Unauthorized(c, "Session expired or invalid")
		return
	}

	if config.Provider != "redis" {
		BadRequest(c, "This endpoint is only available for Redis connections")
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
		BadRequest(c, "Provider error")
		return
	}

	redisProvider, ok := provider.(*providers.RedisProvider)
	if !ok {
		BadRequest(c, "Invalid Redis provider")
		return
	}

	if err := redisProvider.DeleteKey(*config, req.Key); err != nil {
		InternalError(c, err, "Failed to delete key")
		return
	}

	Success(c, "Key deleted successfully", nil)
}

