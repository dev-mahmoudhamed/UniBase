package handlers

import (
	"db-server/db/registry"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetProviders(c *gin.Context) {
	reg := registry.GetRegistry()
	response := reg.GetProvidersResponse()

	c.JSON(http.StatusOK, response)
}
