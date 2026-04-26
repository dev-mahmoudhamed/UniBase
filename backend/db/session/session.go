package session

import (
	"database/sql"
	"db-server/models"
	"fmt"
	"hash/fnv"
	"log"
	"strconv"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const (
	sessionDBPath  = "./sessions.db"
	createTableSQL = `CREATE TABLE IF NOT EXISTS sessions (
		"id" INTEGER PRIMARY KEY,
		"provider" TEXT,
		"host" TEXT,
		"port" INTEGER,
		"user" TEXT,
		"database" TEXT,
		"ssl_mode" TEXT
	);`
)

var (
	sessionDB     *sql.DB
	sessionPWDMap = make(map[string]string)
	mapMutex      sync.RWMutex
)

func InitSessionStore() {
	var err error
	sessionDB, err = sql.Open("sqlite3", sessionDBPath)
	if err != nil {
		log.Fatal("Failed to open sessions.db: ", err)
	}

	statement, err := sessionDB.Prepare(createTableSQL)
	if err != nil {
		log.Fatal("Failed to prepare create table statement: ", err)
	}
	defer statement.Close()

	_, err = statement.Exec()
	if err != nil {
		log.Fatal("Failed to create sessions table: ", err)
	}
}

func generateSessionID(config models.ConnectionConfig) string {
	h := fnv.New64a()
	h.Write([]byte(config.Provider))
	h.Write([]byte(config.Host))
	h.Write([]byte(fmt.Sprintf("%d", config.Port)))
	h.Write([]byte(config.User))
	h.Write([]byte(config.Database))
	h.Write([]byte(config.SSLMode))

	id := int64(h.Sum64())
	if id < 0 {
		id = -id
	}
	return strconv.FormatInt(id, 10)
}

func StoreSession(config models.ConnectionConfig) (string, error) {
	sessionID := generateSessionID(config)

	stmt, err := sessionDB.Prepare("INSERT OR REPLACE INTO sessions(id, provider, host, port, user, database, ssl_mode) values(?,?,?,?,?,?,?)")
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	_, err = stmt.Exec(sessionID, config.Provider, config.Host, config.Port, config.User, config.Database, config.SSLMode)
	if err != nil {
		return "", err
	}

	// Store Password in Map (Thread-safe)
	mapMutex.Lock()
	sessionPWDMap[sessionID] = config.Password
	mapMutex.Unlock()

	return sessionID, nil
}

// GetSession retrieves session from SQLite and Map
func GetSession(sessionIDStr string) (*models.ConnectionConfig, error) {
	// Get from Map
	mapMutex.RLock()
	password, ok := sessionPWDMap[sessionIDStr]
	mapMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found in memory")
	}

	// Get from SQLite
	id, err := strconv.ParseInt(sessionIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID format: %w", err)
	}

	var config models.ConnectionConfig
	err = sessionDB.QueryRow("SELECT provider, host, port, user, database, ssl_mode FROM sessions WHERE id = ?", id).
		Scan(&config.Provider, &config.Host, &config.Port, &config.User, &config.Database, &config.SSLMode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found in database")
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	config.Password = password
	return &config, nil
}
