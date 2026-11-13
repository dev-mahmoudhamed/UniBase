package handlers

import (
	"db-server/db/providers"
	"db-server/models"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// mapConnectionError picks a status and a short hint, and always includes the
// first line of the driver's message so the user can see what actually failed.
func mapConnectionError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	detail := sanitizeError(err)
	msg := strings.ToLower(detail)

	status, hint := http.StatusInternalServerError, "Could not connect"
	switch {
	case strings.Contains(msg, "password authentication failed"),
		strings.Contains(msg, "login failed"),
		strings.Contains(msg, "access denied"),
		strings.Contains(msg, "authentication failed"),
		strings.Contains(msg, "wrongpass"), // Redis
		strings.Contains(msg, "noauth"):    // Redis
		status, hint = http.StatusUnauthorized, "Authentication failed, check the username and password"
	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "actively refused"), // Windows wording
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "server selection"):
		status, hint = http.StatusBadGateway, "Server unreachable, check host, port and network"
	case strings.Contains(msg, "unknown database"),
		strings.Contains(msg, "does not exist"),
		strings.Contains(msg, "cannot open database"):
		status, hint = http.StatusBadRequest, "Database not found, check the database name"
	}

	log.Printf("connection: %v", err)
	c.JSON(status, APIResponse{Status: "error", Error: hint + ": " + detail})
}

// TestConnection validates credentials without persisting a session.
func TestConnection(c *gin.Context) {
	provider, cfg, ok := bindAndResolveProvider(c)
	if !ok {
		return
	}

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
