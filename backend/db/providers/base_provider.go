package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"db-server/models"
)

type BaseProvider struct{}

func (b *BaseProvider) GetCollectionMetadata(config models.ConnectionConfig, collectionName string, context map[string]string) (models.CollectionMetadata, error) {
	return models.CollectionMetadata{}, fmt.Errorf("metadata retrieval not implemented for this provider")
}


// TestConnectionWithTimeout tests database connection with a timeout
func (b *BaseProvider) TestConnectionWithTimeout(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (s *SQLServerProvider) BuildExplorerTree(db *sql.DB, scripts map[string]string) (string, error) {

	dbJson, err := s.ExecuteScript(db, scripts["databases"])
	if err != nil {
		return "", err
	}

	var dbList []struct {
		Name     string `json:"name"`
		IsSystem bool   `json:"is_system"`
	}

	if err := json.Unmarshal([]byte(dbJson), &dbList); err != nil {
		return "", err
	}

	var result models.ExplorerTree

	for _, dbMeta := range dbList {

		_, err := db.Exec("USE [" + dbMeta.Name + "]")
		if err != nil {
			return "", err
		}

		// Execute all scripts
		tablesJson, _ := s.ExecuteScript(db, scripts["tables"])
		columnsJson, _ := s.ExecuteScript(db, scripts["columns"])
		keysJson, _ := s.ExecuteScript(db, scripts["keys"])
		indexesJson, _ := s.ExecuteScript(db, scripts["indexes"])
		viewsJson, _ := s.ExecuteScript(db, scripts["views"])
		procsJson, _ := s.ExecuteScript(db, scripts["procedures"])
		triggersJson, _ := s.ExecuteScript(db, scripts["triggers"])

		// Unmarshal
		var tables []models.Table
		var columns []models.Column
		var keys []models.Key
		var indexes []models.Index
		var views []models.SimpleNode
		var procs []models.SimpleNode
		var triggers []models.SimpleNode

		json.Unmarshal([]byte(tablesJson), &tables)
		json.Unmarshal([]byte(columnsJson), &columns)
		json.Unmarshal([]byte(keysJson), &keys)
		json.Unmarshal([]byte(indexesJson), &indexes)
		json.Unmarshal([]byte(viewsJson), &views)
		json.Unmarshal([]byte(procsJson), &procs)
		json.Unmarshal([]byte(triggersJson), &triggers)

		// 🔥 Build fast lookup map
		tableMap := make(map[int]*models.Table)

		for i := range tables {
			tableMap[tables[i].ObjectID] = &tables[i]
		}

		// Attach columns
		for _, col := range columns {
			if table, ok := tableMap[col.ObjectID]; ok {
				table.Columns = append(table.Columns, col)
			}
		}

		// Attach keys
		for _, key := range keys {
			if table, ok := tableMap[key.ObjectID]; ok {
				table.Keys = append(table.Keys, key)
			}
		}

		// Attach indexes
		for _, idx := range indexes {
			if table, ok := tableMap[idx.ObjectID]; ok {
				table.Indexes = append(table.Indexes, idx)
			}
		}

		// Build database node
		database := models.Database{
			Name:             dbMeta.Name,
			Tables:           tables,
			Views:            views,
			StoredProcedures: procs,
			DatabaseTriggers: triggers,
			IsSystem:         dbMeta.IsSystem,
		}

		if dbMeta.IsSystem {
			result.SystemDatabases = append(result.SystemDatabases, database)
		} else {
			result.UserDatabases = append(result.UserDatabases, database)
		}
	}

	finalJson, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(finalJson), nil
}

func (b *BaseProvider) ExecuteScript(db *sql.DB, script string) (string, error) {
	var result string
	err := db.QueryRow(script).Scan(&result)
	if err != nil {
		return "", fmt.Errorf("failed to execute script: %w", err)
	}
	return result, nil
}

func (b *BaseProvider) OpenAndValidate(driver, connStr string) (*sql.DB, error) {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("open failed: %w", err)
	}
	return db, nil
}
