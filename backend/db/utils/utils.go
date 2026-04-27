package utils

import (
	"context"
	"database/sql"
	"db-server/db/session"
	"db-server/models"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// goBatchSplitter matches a standalone GO statement on its own line (case-insensitive)
var goBatchSplitter = regexp.MustCompile(`(?im)^\s*GO\s*$`)

// SplitSQLBatches splits a SQL script on GO batch separators, trimming empty batches.
func SplitSQLBatches(script string) []string {
	parts := goBatchSplitter.Split(script, -1)
	batches := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			batches = append(batches, t)
		}
	}
	return batches
}

// executeBatch runs a single SQL batch and captures the last result set that has columns.
func executeBatch(conn *sql.Conn, ctx context.Context, batch string, finalColumns *[]string, finalResults *[]map[string]interface{}) error {
	rows, err := conn.QueryContext(ctx, batch)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	for {
		columns, colErr := rows.Columns()
		if colErr == nil && len(columns) > 0 {
			*finalColumns = columns
			results := []map[string]interface{}{}

			values := make([]interface{}, len(columns))
			pointers := make([]interface{}, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}

			for rows.Next() {
				if scanErr := rows.Scan(pointers...); scanErr != nil {
					return fmt.Errorf("failed to scan row: %w", scanErr)
				}

				result := make(map[string]interface{})
				for i, colName := range columns {
					val := values[i]
					if b, ok := val.([]byte); ok {
						result[colName] = string(b)
					} else {
						result[colName] = val
					}
				}
				results = append(results, result)
			}

			if rowErr := rows.Err(); rowErr != nil {
				return fmt.Errorf("rows iteration error: %w", rowErr)
			}

			*finalResults = results
		} else {
			// Drain rows for batches that return no columns (USE, CREATE, etc.)
			for rows.Next() {
			}
		}

		if !rows.NextResultSet() {
			break
		}
	}

	return nil
}

// ExecuteDynamicQuery splits the query on GO batch separators and executes each
// batch independently, which is required by SQL Server for DDL statements like
// CREATE TRIGGER, CREATE PROCEDURE, etc.
func ExecuteDynamicQuery(db *sql.DB, query string) (models.QueryResult, error) {
	startTime := time.Now()

	batches := SplitSQLBatches(query)
	// If there are no GO separators, treat the whole query as one batch.
	if len(batches) == 0 {
		batches = []string{query}
	}

	var finalColumns []string
	var finalResults []map[string]interface{}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to acquire connection for query execution: %w", err)
	}
	defer conn.Close()

	for _, batch := range batches {
		if err := executeBatch(conn, ctx, batch, &finalColumns, &finalResults); err != nil {
			return models.QueryResult{}, err
		}
	}

	executionTime := time.Since(startTime).Milliseconds()

	if finalColumns == nil {
		finalColumns = []string{}
	}
	if finalResults == nil {
		finalResults = []map[string]interface{}{}
	}

	return models.QueryResult{
		Columns:       finalColumns,
		Rows:          finalResults,
		RowCount:      len(finalResults),
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
