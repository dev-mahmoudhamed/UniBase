package handlers

import (
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"

	"github.com/gin-gonic/gin"
)


// UpdateMongoDocument handles updating specific fields on a MongoDB document
func UpdateMongoDocument(c *gin.Context) {
	var req models.MongoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	config, err := utils.GetSession(req.SessionID)
	if err != nil {
		Unauthorized(c, "Session expired or invalid")
		return
	}

	provider, err := providers.GetProvider(config.Provider)
	if err != nil {
		BadRequest(c, "Provider error")
		return
	}

	mongoProvider, ok := provider.(*providers.MongoDBProvider)
	if !ok {
		BadRequest(c, "This endpoint is only available for MongoDB connections")
		return
	}

	if err := mongoProvider.UpdateDocument(*config, req.Database, req.Collection, req.ObjectID, req.Properties); err != nil {
		InternalError(c, err, "Failed to update document")
		return
	}

	Success(c, "Document updated successfully", nil)
}
