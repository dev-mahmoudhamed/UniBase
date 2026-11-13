// Package session keeps open connection settings in memory, keyed by a random
// session ID. Nothing is written to disk, so credentials are gone when the
// server stops; the UI reconnects automatically.
package session

import (
	"crypto/rand"
	"db-server/models"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

type entry struct {
	config   models.ConnectionConfig
	lastUsed time.Time
}

var (
	mu       sync.Mutex
	sessions = map[string]*entry{}
)

// StoreSession saves the settings and returns a new unique session ID.
func StoreSession(config models.ConnectionConfig) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: failed to generate ID: %w", err)
	}
	id := hex.EncodeToString(b)

	mu.Lock()
	sessions[id] = &entry{config: config, lastUsed: time.Now()}
	mu.Unlock()
	return id, nil
}

// GetSession returns a copy of the settings for sessionID.
func GetSession(sessionID string) (*models.ConnectionConfig, error) {
	mu.Lock()
	defer mu.Unlock()
	e, ok := sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session: not found")
	}
	e.lastUsed = time.Now()
	cfg := e.config
	return &cfg, nil
}

// CleanupExpiredSessions removes sessions not used for longer than idle.
func CleanupExpiredSessions(idle time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	removed := 0
	for id, e := range sessions {
		if time.Since(e.lastUsed) > idle {
			delete(sessions, id)
			removed++
		}
	}
	if removed > 0 {
		log.Printf("session: removed %d idle session(s)", removed)
	}
}
