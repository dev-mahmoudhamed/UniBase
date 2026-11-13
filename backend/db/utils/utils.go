package utils

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"db-server/db/session"
	"db-server/models"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	mssql "github.com/denisenkom/go-mssqldb"
)

// MaxRowsPerResultSet caps how many rows of each result set are read into memory.
const MaxRowsPerResultSet = 1000

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

// executeBatch runs one SQL batch and appends every result set that has columns.
func executeBatch(ctx context.Context, conn *sql.Conn, batch string, sets *[]models.ResultSet) error {
	rows, err := conn.QueryContext(ctx, batch)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	for {
		columns, _ := rows.Columns()
		if len(columns) > 0 {
			set, err := readResultSet(rows, columns)
			if err != nil {
				return err
			}
			*sets = append(*sets, set)
		}
		if !rows.NextResultSet() {
			break
		}
	}
	return rows.Err()
}

func readResultSet(rows *sql.Rows, columns []string) (models.ResultSet, error) {
	dbTypes := make([]string, len(columns))
	if types, err := rows.ColumnTypes(); err == nil {
		for i, t := range types {
			dbTypes[i] = t.DatabaseTypeName()
		}
	}

	set := models.ResultSet{Columns: columns, Rows: [][]interface{}{}}
	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	for rows.Next() {
		if len(set.Rows) == MaxRowsPerResultSet {
			set.Truncated = true
			break
		}
		if err := rows.Scan(pointers...); err != nil {
			return set, fmt.Errorf("failed to scan row: %w", err)
		}
		row := make([]interface{}, len(columns))
		for i, v := range values {
			row[i] = cellValue(v, dbTypes[i])
		}
		set.Rows = append(set.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return set, fmt.Errorf("rows iteration error: %w", err)
	}
	set.RowCount = len(set.Rows)
	return set, nil
}

// cellValue makes driver byte slices readable: SQL Server GUIDs as text,
// UTF-8 bytes as strings, anything else as 0x-prefixed hex.
func cellValue(v interface{}, dbType string) interface{} {
	b, ok := v.([]byte)
	if !ok {
		return v
	}
	if dbType == "UNIQUEIDENTIFIER" && len(b) == 16 {
		var id mssql.UniqueIdentifier
		if id.Scan(b) == nil {
			return id.String()
		}
	}
	if utf8.Valid(b) {
		return string(b)
	}
	return "0x" + hex.EncodeToString(b)
}

// ExecuteDynamicQuery splits the script on GO separators (required by SQL Server
// for CREATE PROCEDURE and similar) and runs the batches in order on one
// connection, returning every result set. ctx carries the timeout and cancel.
func ExecuteDynamicQuery(ctx context.Context, db *sql.DB, query string) (models.QueryResult, error) {
	startTime := time.Now()

	batches := SplitSQLBatches(query)
	if len(batches) == 0 {
		batches = []string{query}
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to acquire connection for query execution: %w", err)
	}
	// User scripts can change session state (USE, SET, open transactions), so this
	// connection is discarded instead of going back to the shared pool.
	defer conn.Raw(func(any) error { return driver.ErrBadConn })

	sets := []models.ResultSet{}
	for _, batch := range batches {
		if err := executeBatch(ctx, conn, batch, &sets); err != nil {
			return models.QueryResult{}, err
		}
	}

	return models.QueryResult{
		ResultSets:    sets,
		ExecutionTime: time.Since(startTime).Milliseconds(),
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
