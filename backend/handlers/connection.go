package handlers

import (
	"context"
	"db-server/db/providers"
	"db-server/models"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const connectionTimeout = 15 * time.Second

// connectionRequest is the validated input for both test and initialize.
type connectionRequest struct {
	models.ConnectionConfig
}

// bindAndResolveProvider centralises JSON binding, basic field validation,
// and provider resolution. Returns false (and writes the HTTP error) on failure.
func bindAndResolveProvider(c *gin.Context) (providers.DatabaseProvider, models.ConnectionConfig, bool) {
	var cfg models.ConnectionConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		BadRequest(c, "Invalid request format: "+sanitizeError(err))
		return nil, cfg, false
	}

	// Required-field validation
	if strings.TrimSpace(cfg.Provider) == "" {
		BadRequest(c, "Database provider is required")
		return nil, cfg, false
	}
	if strings.TrimSpace(cfg.Host) == "" {
		BadRequest(c, "Host address is required")
		return nil, cfg, false
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		BadRequest(c, "Port must be between 1 and 65535")
		return nil, cfg, false
	}

	provider, err := providers.GetProvider(cfg.Provider)
	if err != nil {
		BadRequest(c, "Unsupported provider: "+cfg.Provider)
		return nil, cfg, false
	}

	return provider, cfg, true
}

// sanitizeError strips internal stack/path details from error strings that
// should never be surfaced to clients.
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Trim anything after a newline (stack traces)
	if idx := strings.Index(msg, "\n"); idx > 0 {
		msg = msg[:idx]
	}
	return msg
}

// mapConnectionError converts low-level driver errors to user-friendly messages
// and picks the appropriate HTTP status code.
func mapConnectionError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "access denied"),
		strings.Contains(msg, "authentication failed"),
		strings.Contains(msg, "invalid credentials"),
		strings.Contains(msg, "password"):
		c.JSON(http.StatusUnauthorized, APIResponse{
			Status: "error",
			Error:  "Authentication failed — check your username and password",
		})

	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "network"):
		c.JSON(http.StatusBadGateway, APIResponse{
			Status: "error",
			Error:  "Server unreachable — check host, port, and network connectivity",
		})

	case strings.Contains(msg, "unknown database"),
		strings.Contains(msg, "database") && strings.Contains(msg, "not exist"):
		BadRequest(c, "Database does not exist — check the database name")

	default:
		// Do NOT leak the raw driver error; log it server-side only.
		log.Printf("connection: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Status: "error",
			Error:  "Could not establish connection — check your connection settings",
		})
	}
}

// TestConnection validates credentials without persisting a session.
func TestConnection(c *gin.Context) {
	provider, cfg, ok := bindAndResolveProvider(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	_ = ctx // providers use their own internal context; passed for documentation

	log.Printf("connection: test requested for provider=%s host=%s", cfg.Provider, cfg.Host)

	if err := provider.TestConnection(cfg); err != nil {
		log.Printf("connection: test failed for provider=%s host=%s: %v", cfg.Provider, cfg.Host, err)
		mapConnectionError(c, err)
		return
	}

	log.Printf("connection: test succeeded for provider=%s host=%s", cfg.Provider, cfg.Host)
	Success(c, "Connection established successfully", nil)
}

// InitializeConnection opens, verifies, and persists a new database session.
func InitializeConnection(c *gin.Context) {
	provider, cfg, ok := bindAndResolveProvider(c)
	if !ok {
		return
	}

	log.Printf("connection: initialize requested for provider=%s host=%s", cfg.Provider, cfg.Host)

	sessionID, metaResult, err := provider.InitializeConnection(cfg)
	if err != nil {
		log.Printf("connection: initialize failed for provider=%s host=%s: %v", cfg.Provider, cfg.Host, err)
		mapConnectionError(c, err)
		return
	}

	log.Printf("connection: initialized session %s for provider=%s host=%s", sessionID, cfg.Provider, cfg.Host)

	Success(c, "Connection initialized successfully", gin.H{
		"session_id": sessionID,
		"provider":   cfg.Provider,
		"host":       cfg.Host,
		"metadata":   metaResult,
	})
}
