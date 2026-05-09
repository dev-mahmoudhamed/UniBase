package handlers

import (
	"db-server/db/registry"

	"github.com/gin-gonic/gin"
)

func GetProviders(c *gin.Context) {
	reg := registry.GetRegistry()
	response := reg.GetProvidersResponse()

	Success(c, "Providers retrieved", response)
}

