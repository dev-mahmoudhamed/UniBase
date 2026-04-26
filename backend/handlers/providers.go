package handlers

import (
	"db-server/db/registry"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetProviders returns all available database provider metadata
func GetProviders(c *gin.Context) {
	reg := registry.GetRegistry()
	response := reg.GetProvidersResponse()

	c.JSON(http.StatusOK, response)
}

// func GetProviderByID(c *gin.Context) {
// 	providerID := c.Param("id")
// 	reg := registry.GetRegistry()
// 	provider, err := reg.GetProvider(providerID)
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, provider)
// }

// func ValidateProviderConfig(c *gin.Context) {
// 	var request struct {
// 		ProviderID string                 `json:"providerId" binding:"required"`
// 		Config     map[string]interface{} `json:"config" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
// 		return
// 	}
// 	reg := registry.GetRegistry()
// 	errors := reg.ValidateConnection(request.ProviderID, request.Config)
// 	if len(errors) > 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"valid":  false,
// 			"errors": errors,
// 		})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"valid": true,
// 	})
// }
