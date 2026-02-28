package handlers

import (
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetExplorerChildren handles requests to get child nodes for the object explorer tree
func GetExplorerChildren(c *gin.Context) {
	var req models.ExplorerNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get session config to find provider and connection details
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

	nodes, err := provider.GetExplorerChildren(*config, req.NodeType, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.ExplorerNodeResponse{Nodes: nodes})
}

// GetCollectionData handles requests to fetch document data from a collection (MongoDB)
func GetCollectionData(c *gin.Context) {
	var req models.CollectionDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get session config
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

	// Check if the provider supports collection data
	collProvider, ok := provider.(providers.CollectionDataProvider)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This provider does not support collection data retrieval"})
		return
	}

	data, err := collProvider.GetCollectionData(*config, req.CollectionName, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collection": req.CollectionName,
		"documents":  data,
		"count":      len(data),
	})
}
