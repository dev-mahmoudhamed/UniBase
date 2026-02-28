package handlers

import (
	"db-server/db/providers"
	"db-server/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TestConnection(c *gin.Context) {
	var config models.ConnectionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := provider.TestConnection(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Connection established successfully"})
}

func InitializeConnection(c *gin.Context) {
	var config models.ConnectionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID, metaResult, err := provider.InitializeConnection(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"provider":   config.Provider,
		"host":       config.Host,
		"metadata":   metaResult,
	})
}
