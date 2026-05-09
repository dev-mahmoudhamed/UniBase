package session

import (
	"context"
	"database/sql"
	"db-server/models"
	"fmt"
	"log"
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const (
	sessionDBPath  = "./sessions.db"
	sessionTimeout = 5 * time.Second
	createTableSQL = `CREATE TABLE IF NOT EXISTS sessions (
		"id" TEXT PRIMARY KEY,
		"provider" TEXT NOT NULL,
		"host" TEXT NOT NULL,
		"port" INTEGER NOT NULL,
		"user" TEXT,
		"password" TEXT,
		"database" TEXT,
		"ssl_mode" TEXT,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
)

// SessionStore wraps the SQLite database used to persist connection sessions.
type SessionStore struct {
	db *sql.DB
	mu sync.RWMutex
}

var (
	defaultStore *SessionStore
	once         sync.Once
)

// InitSessionStore initialises the singleton session store.
// It is safe to call multiple times; subsequent calls are no-ops.
func InitSessionStore() {
	once.Do(func() {
		db, err := sql.Open("sqlite3", sessionDBPath)
		if err != nil {
			log.Fatalf("session: failed to open %s: %v", sessionDBPath, err)
		}

		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(time.Hour)

		ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("session: failed to ping %s: %v", sessionDBPath, err)
		}

		if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
			log.Fatalf("session: failed to create sessions table: %v", err)
		}

		defaultStore = &SessionStore{db: db}
		log.Println("session: store initialised successfully")
	})
}

// CloseSessionStore closes the underlying database connection.
func CloseSessionStore() {
	if defaultStore != nil && defaultStore.db != nil {
		if err := defaultStore.db.Close(); err != nil {
			log.Printf("session: error closing store: %v", err)
		}
	}
}

func generateSecureID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: failed to generate secure ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// StoreSession persists a new session and returns its unique ID.
func StoreSession(config models.ConnectionConfig) (string, error) {
	if defaultStore == nil {
		return "", fmt.Errorf("session: store not initialised")
	}

	sessionID, err := generateSecureID()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()

	const query = `INSERT INTO sessions(id, provider, host, port, user, password, database, ssl_mode)
	               VALUES(?,?,?,?,?,?,?,?)`

	defaultStore.mu.Lock()
	defer defaultStore.mu.Unlock()

	if _, err := defaultStore.db.ExecContext(ctx, query,
		sessionID, config.Provider, config.Host, config.Port,
		config.User, config.Password, config.Database, config.SSLMode,
	); err != nil {
		return "", fmt.Errorf("session: failed to store session: %w", err)
	}

	log.Printf("session: created %s for provider=%s host=%s", sessionID, config.Provider, config.Host)
	return sessionID, nil
}

// GetSession retrieves the connection config associated with sessionID.
func GetSession(sessionID string) (*models.ConnectionConfig, error) {
	if defaultStore == nil {
		return nil, fmt.Errorf("session: store not initialised")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session: empty session ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()

	const query = `SELECT provider, host, port, user, password, database, ssl_mode
	               FROM sessions WHERE id = ?`

	defaultStore.mu.RLock()
	defer defaultStore.mu.RUnlock()

	var cfg models.ConnectionConfig
	err := defaultStore.db.QueryRowContext(ctx, query, sessionID).Scan(
		&cfg.Provider, &cfg.Host, &cfg.Port,
		&cfg.User, &cfg.Password, &cfg.Database, &cfg.SSLMode,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session: not found")
		}
		return nil, fmt.Errorf("session: failed to retrieve session: %w", err)
	}

	return &cfg, nil
}

// CleanupExpiredSessions removes sessions older than the given duration.
func CleanupExpiredSessions(olderThan time.Duration) {
	if defaultStore == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-olderThan).Format("2006-01-02 15:04:05")

	defaultStore.mu.Lock()
	defer defaultStore.mu.Unlock()

	res, err := defaultStore.db.ExecContext(ctx, "DELETE FROM sessions WHERE created_at < ?", cutoff)
	if err != nil {
		log.Printf("session: cleanup error: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("session: cleaned up %d expired session(s)", n)
	}
}
