package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success sends HTTP 200 with a success envelope.
func Success(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// BadRequest sends HTTP 400 with an error envelope.
func BadRequest(c *gin.Context, err string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Status: "error",
		Error:  err,
	})
}

// Unauthorized sends HTTP 401 with an error envelope.
func Unauthorized(c *gin.Context, err string) {
	c.JSON(http.StatusUnauthorized, APIResponse{
		Status: "error",
		Error:  err,
	})
}

// Accepted sends HTTP 202 for async / long-running operations.
func Accepted(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusAccepted, APIResponse{
		Status:  "accepted",
		Message: message,
		Data:    data,
	})
}

// InternalError logs the raw error and sends HTTP 500 with a safe user message.
// Pass an empty userMsg to use the generic fallback.
func InternalError(c *gin.Context, err error, userMsg string, extra ...interface{}) {
	if err != nil {
		log.Printf("internal error: %v", err)
	}
	if userMsg == "" {
		userMsg = "An internal server error occurred"
	}

	resp := APIResponse{
		Status: "error",
		Error:  userMsg,
	}
	if len(extra) > 0 {
		resp.Data = extra[0]
	}

	c.JSON(http.StatusInternalServerError, resp)
}

// ValidateRequiredFields checks that the given keys exist and are non-empty
// in the provided map. Returns false and writes a 400 response on failure.
func ValidateRequiredFields(c *gin.Context, data map[string]interface{}, fields []string) bool {
	for _, field := range fields {
		if val, ok := data[field]; !ok || val == "" {
			BadRequest(c, field+" is required")
			return false
		}
	}
	return true
}
