package utils

import (
	"database/sql"
	"db-server/db/session"
	"db-server/models"
	"fmt"
	"time"
)

func ExecuteDynamicQuery(db *sql.DB, query string) (models.QueryResult, error) {
	startTime := time.Now()
	rows, err := db.Query(query)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to get columns: %w", err)
	}

	results := []map[string]interface{}{}

	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return models.QueryResult{}, fmt.Errorf("failed to scan row: %w", err)
		}

		result := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]
			// Handle []uint8 (byte slice) which is common for strings in some drivers
			if b, ok := val.([]byte); ok {
				result[colName] = string(b)
			} else {
				result[colName] = val
			}
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return models.QueryResult{}, fmt.Errorf("rows iteration error: %w", err)
	}

	executionTime := time.Since(startTime).Milliseconds()

	return models.QueryResult{
		Columns:       columns,
		Rows:          results,
		RowCount:      len(results),
		ExecutionTime: executionTime,
	}, nil
}

// StoreSession is a convenience wrapper for session.StoreSession
func StoreSession(config models.ConnectionConfig) (string, error) {
	return session.StoreSession(config)
}

// GetSession is a convenience wrapper for session.GetSession
func GetSession(sessionID string) (*models.ConnectionConfig, error) {
	return session.GetSession(sessionID)
}
