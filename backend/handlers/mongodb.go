package handlers

import (
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UpdateMongoDocument handles updating specific fields on a MongoDB document
func UpdateMongoDocument(c *gin.Context) {
	var req models.MongoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mongoProvider, ok := provider.(*providers.MongoDBProvider)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This endpoint is only available for MongoDB connections"})
		return
	}

	if err := mongoProvider.UpdateDocument(*config, req.Database, req.Collection, req.ObjectID, req.Properties); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document updated successfully"})
}
