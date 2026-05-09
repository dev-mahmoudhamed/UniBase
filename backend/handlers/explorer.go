package handlers

import (
	"db-server/db/providers"
	"db-server/db/utils"
	"db-server/models"
	"sort"

	"github.com/gin-gonic/gin"
)


// GetExplorerChildren handles requests to get child nodes for the object explorer tree
func GetExplorerChildren(c *gin.Context) {
	var req models.ExplorerNodeRequest
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

	nodes, err := provider.GetExplorerChildren(*config, req.NodeType, req.Context)
	if err != nil {
		InternalError(c, err, "Failed to load explorer nodes")
		return
	}

	Success(c, "Nodes retrieved", models.ExplorerNodeResponse{Nodes: nodes})
}

func GetCollectionData(c *gin.Context) {
	var req models.CollectionDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	// Get session config
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

	// Check if the provider supports collection data
	collProvider, ok := provider.(providers.CollectionDataProvider)
	if !ok {
		BadRequest(c, "This provider does not support collection data retrieval")
		return
	}

	data, totalCount, err := collProvider.GetCollectionData(*config, req.CollectionName, req.Context, req.Filter, req.Projection, req.Sort, req.Skip, req.Limit)
	if err != nil {
		InternalError(c, err, "Failed to retrieve collection data")
		return
	}
 
	// Calculate columns sorted: _id first, then alphabetically
	columnsMap := make(map[string]bool)
	for _, doc := range data {
		for k := range doc {
			columnsMap[k] = true
		}
	}
 
	var columns []string
	if columnsMap["_id"] {
		columns = append(columns, "_id")
		delete(columnsMap, "_id")
	}
 
	var otherColumns []string
	for k := range columnsMap {
		otherColumns = append(otherColumns, k)
	}
	sort.Strings(otherColumns)
	columns = append(columns, otherColumns...)

	Success(c, "Data retrieved", gin.H{
		"collection": req.CollectionName,
		"documents":  data,
		"columns":    columns,
		"count":      totalCount,
	})
}

func GetCollectionMetadata(c *gin.Context) {
	var req models.CollectionDataRequest
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

	metadata, err := provider.GetCollectionMetadata(*config, req.CollectionName, req.Context)
	if err != nil {
		InternalError(c, err, "Failed to retrieve collection metadata")
		return
	}

	Success(c, "Metadata retrieved", models.CollectionMetadataResponse{
		Collection: req.CollectionName,
		Metadata:   metadata,
	})
}
