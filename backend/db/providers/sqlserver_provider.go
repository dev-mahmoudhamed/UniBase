package providers

import (
	"database/sql"
	"db-server/db/utils"
	"db-server/models"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

const (
	sqlServerDriver     = "sqlserver"
	sqlServerScriptPath = "./scripts/mssql/metadata.sql"
)

type SQLServerProvider struct {
	BaseProvider
	db *sql.DB
}

func (s *SQLServerProvider) buildConnectionURL(config models.ConnectionConfig) string {
	query := url.Values{}
	query.Add("database", config.Database)

	u := &url.URL{
		Scheme:   sqlServerDriver,
		User:     url.UserPassword(config.User, config.Password),
		Host:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		RawQuery: query.Encode(),
	}

	return u.String()
}

func (s *SQLServerProvider) TestConnection(config models.ConnectionConfig) error {
	connURL := s.buildConnectionURL(config)

	db, err := s.OpenAndValidate(sqlServerDriver, connURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("error pinging database: %v", err)
	}

	return nil
}

func (s *SQLServerProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	connURL := s.buildConnectionURL(config)

	db, err := sql.Open(sqlServerDriver, connURL)
	if err != nil {
		return "", nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	s.db = db
	if err := db.Ping(); err != nil {
		return "", nil, err
	}

	sessionID, err := utils.StoreSession(config)
	if err != nil {
		return "", nil, err
	}

	return sessionID, map[string]interface{}{}, nil
}

func (s *SQLServerProvider) ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error) {
	connURL := s.buildConnectionURL(config)

	db, err := s.OpenAndValidate(sqlServerDriver, connURL)
	if err != nil {
		return models.QueryResult{}, err
	}
	defer db.Close()

	return utils.ExecuteDynamicQuery(db, query)
}

func (s *SQLServerProvider) GetExplorerChildren(config models.ConnectionConfig, nodeType string, ctx map[string]string) ([]models.ExplorerNode, error) {
	connURL := s.buildConnectionURL(config)

	db, err := s.OpenAndValidate(sqlServerDriver, connURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	switch nodeType {
	case "root":
		return s.getRootNodes()
	case "databases":
		return s.getDatabaseNodes(db)
	case "database":
		return s.getDatabaseChildFolders(ctx)
	case "tables":
		return s.getTableNodes(db, ctx)
	case "table":
		return s.getTableChildFolders(ctx)
	case "tableColumns":
		return s.getTableColumnNodes(db, ctx)
	case "tableKeys":
		return s.getTableKeyNodes(db, ctx)
	case "tableIndexes":
		return s.getTableIndexNodes(db, ctx)
	case "views":
		return s.getViewNodes(db, ctx)
	case "procedures":
		return s.getProcedureNodes(db, ctx)
	case "databaseTriggers":
		return s.getDatabaseTriggerNodes(db, ctx)
	case "tableConstraints":
		return s.getTableConstraintNodes(db, ctx)
	case "tableTriggers":
		return s.getTableTriggerNodes(db, ctx)
	case "tableStatistics":
		return s.getTableStatisticNodes(db, ctx)
	case "functions":
		return s.getFunctionChildFolders(ctx)
	case "tableValuedFunctions":
		return s.getTableValuedFunctionNodes(db, ctx)
	case "scalarValuedFunctions":
		return s.getScalarValuedFunctionNodes(db, ctx)
	case "security":
		return s.getSecurityChildFolders(ctx)
	case "dbUsers":
		return s.getDbUserNodes(db, ctx)
	case "dbRoles":
		return s.getDbRoleChildFolders(ctx)
	case "databaseRoles":
		return s.getDatabaseRoleNodes(db, ctx)
	case "applicationRoles":
		return s.getApplicationRoleNodes(db, ctx)
	case "dbSchemas":
		return s.getDbSchemaNodes(db, ctx)
	case "serverRoles":
		return s.getServerRoleNodes(db)
	case "logins":
		return s.getLoginNodes(db)
	case "serverTriggers":
		return s.getServerTriggerNodes(db)
	case "serverLogs":
		return s.getServerLogNodes(db)
	default:
		return []models.ExplorerNode{}, nil
	}
}

func (s *SQLServerProvider) getRootNodes() ([]models.ExplorerNode, error) {
	return []models.ExplorerNode{
		{Key: "databases", Label: "Databases", Type: "databases", Icon: "pi pi-database", Leaf: false},
		{Key: "serverRoles", Label: "Server Roles", Type: "serverRoles", Icon: "pi pi-shield", Leaf: false},
		{Key: "logins", Label: "Logins", Type: "logins", Icon: "pi pi-users", Leaf: false},
		{Key: "serverTriggers", Label: "Triggers", Type: "serverTriggers", Icon: "pi pi-bolt", Leaf: false},
		{Key: "serverLogs", Label: "Logs", Type: "serverLogs", Icon: "pi pi-file", Leaf: false},
	}, nil
}

func (s *SQLServerProvider) getDatabaseNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT name, database_id FROM sys.databases WHERE state = 0 AND database_id > 4 ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)

	for rows.Next() {
		var name string
		var dbID int
		if err := rows.Scan(&name, &dbID); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "db:" + name,
			Label: name,
			Type:  "database",
			Icon:  "pi pi-database",
			Leaf:  false,
			Data:  map[string]interface{}{"database": name},
		})
	}

	return nodes, nil
}

func (s *SQLServerProvider) getDatabaseChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	return []models.ExplorerNode{
		{Key: "tables:" + dbName, Label: "Tables", Type: "tables", Icon: "pi pi-table", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "views:" + dbName, Label: "Views", Type: "views", Icon: "pi pi-eye", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "procedures:" + dbName, Label: "Stored Procedures", Type: "procedures", Icon: "pi pi-code", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "triggers:" + dbName, Label: "Database Triggers", Type: "databaseTriggers", Icon: "pi pi-bolt", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "functions:" + dbName, Label: "Functions", Type: "functions", Icon: "pi pi-code", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "security:" + dbName, Label: "Security", Type: "security", Icon: "pi pi-shield", Leaf: false, Data: map[string]interface{}{"database": dbName}},
	}, nil
}

func (s *SQLServerProvider) getTableNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	           SELECT t.object_id, SCHEMA_NAME(t.schema_id) AS schema_name, t.name 
	           FROM sys.tables t WHERE t.is_ms_shipped = 0 ORDER BY t.name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var objectID int
		var schemaName, name string
		if err := rows.Scan(&objectID, &schemaName, &name); err != nil {
			continue
		}
		fullName := schemaName + "." + name
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("table:%s:%s", ctx["database"], fullName),
			Label: fullName,
			Type:  "table",
			Icon:  "pi pi-table",
			Leaf:  false,
			Data:  map[string]interface{}{"database": ctx["database"], "table": name, "schema": schemaName, "objectId": fmt.Sprintf("%d", objectID)},
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getTableChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	tableName := ctx["table"]
	schemaName := ctx["schema"]
	objectId := ctx["objectId"]
	return []models.ExplorerNode{
		{Key: fmt.Sprintf("columns:%s:%s.%s", dbName, schemaName, tableName), Label: "Columns", Type: "tableColumns", Icon: "pi pi-bars", Leaf: false, Data: map[string]interface{}{"database": dbName, "table": tableName, "schema": schemaName, "objectId": objectId}},
		{Key: fmt.Sprintf("keys:%s:%s.%s", dbName, schemaName, tableName), Label: "Keys", Type: "tableKeys", Icon: "pi pi-key", Leaf: false, Data: map[string]interface{}{"database": dbName, "table": tableName, "schema": schemaName, "objectId": objectId}},
		{Key: fmt.Sprintf("indexes:%s:%s.%s", dbName, schemaName, tableName), Label: "Indexes", Type: "tableIndexes", Icon: "pi pi-sitemap", Leaf: false, Data: map[string]interface{}{"database": dbName, "table": tableName, "schema": schemaName, "objectId": objectId}},
		{Key: fmt.Sprintf("constraints:%s:%s.%s", dbName, schemaName, tableName), Label: "Constraints", Type: "tableConstraints", Icon: "pi pi-lock", Leaf: false, Data: map[string]interface{}{"database": dbName, "table": tableName, "schema": schemaName, "objectId": objectId}},
		{Key: fmt.Sprintf("triggers:%s:%s.%s", dbName, schemaName, tableName), Label: "Triggers", Type: "tableTriggers", Icon: "pi pi-bolt", Leaf: false, Data: map[string]interface{}{"database": dbName, "table": tableName, "schema": schemaName, "objectId": objectId}},
		{Key: fmt.Sprintf("statistics:%s:%s.%s", dbName, schemaName, tableName), Label: "Statistics", Type: "tableStatistics", Icon: "pi pi-chart-bar", Leaf: false, Data: map[string]interface{}{"database": dbName, "table": tableName, "schema": schemaName, "objectId": objectId}},
	}, nil
}

func (s *SQLServerProvider) getTableColumnNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT c.name, ty.name AS data_type, c.max_length, c.is_nullable
	          FROM sys.columns c
	          JOIN sys.types ty ON c.user_type_id = ty.user_type_id
	          WHERE c.object_id = @p1
	          ORDER BY c.column_id`, ctx["database"])

	rows, err := db.Query(query, ctx["objectId"])
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name, dataType string
		var maxLen int
		var isNullable bool
		if err := rows.Scan(&name, &dataType, &maxLen, &isNullable); err != nil {
			continue
		}
		nullable := ""
		if isNullable {
			nullable = " (nullable)"
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("col:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)%s", name, dataType, nullable),
			Type:  "column",
			Icon:  "pi pi-tag",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getTableKeyNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT kc.name, kc.type_desc
	          FROM sys.key_constraints kc
	          WHERE kc.parent_object_id = @p1`, ctx["database"])

	rows, err := db.Query(query, ctx["objectId"])
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name, typeDesc string
		if err := rows.Scan(&name, &typeDesc); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("key:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, typeDesc),
			Type:  "key",
			Icon:  "pi pi-key",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getTableIndexNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT i.name, i.type_desc
	          FROM sys.indexes i
	          WHERE i.object_id = @p1
	            AND i.is_primary_key = 0
	            AND i.is_unique_constraint = 0
	            AND i.name IS NOT NULL`, ctx["database"])

	rows, err := db.Query(query, ctx["objectId"])
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name, typeDesc string
		if err := rows.Scan(&name, &typeDesc); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("idx:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, typeDesc),
			Type:  "index",
			Icon:  "pi pi-sitemap",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getViewNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT SCHEMA_NAME(schema_id) AS schema_name, name FROM sys.views WHERE is_ms_shipped = 0 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var schemaName, name string
		if err := rows.Scan(&schemaName, &name); err != nil {
			continue
		}
		fullName := schemaName + "." + name
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("view:%s:%s", ctx["database"], fullName),
			Label: fullName,
			Type:  "view",
			Icon:  "pi pi-eye",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getProcedureNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT SCHEMA_NAME(schema_id) AS schema_name, name FROM sys.procedures WHERE is_ms_shipped = 0 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query procedures: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var schemaName, name string
		if err := rows.Scan(&schemaName, &name); err != nil {
			continue
		}
		fullName := schemaName + "." + name
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("proc:%s:%s", ctx["database"], fullName),
			Label: fullName,
			Type:  "procedure",
			Icon:  "pi pi-code",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getDatabaseTriggerNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name FROM sys.triggers WHERE parent_class = 0 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("dbtrigger:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "trigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getServerRoleNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT name FROM sys.server_principals WHERE type = 'R' ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query server roles: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "role:" + name,
			Label: name,
			Type:  "serverRole",
			Icon:  "pi pi-shield",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getLoginNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT name, type_desc FROM sys.server_principals WHERE type IN ('S', 'U', 'G') ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query logins: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name, typeDesc string
		if err := rows.Scan(&name, &typeDesc); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "login:" + name,
			Label: fmt.Sprintf("%s (%s)", name, typeDesc),
			Type:  "login",
			Icon:  "pi pi-user",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getServerTriggerNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT name FROM sys.server_triggers ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query server triggers: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "srvtrigger:" + name,
			Label: name,
			Type:  "serverTrigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getServerLogNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `EXEC xp_readerrorlog 0, 1, NULL, NULL, NULL, NULL, N'desc'`
	// Try to get logs, if it fails (permissions), return empty
	rows, err := db.Query(query)
	if err != nil {
		// Return a placeholder if user doesn't have permission
		return []models.ExplorerNode{
			{Key: "log:current", Label: "Current - ERRORLOG", Type: "log", Icon: "pi pi-file", Leaf: true},
		}, nil
	}
	defer rows.Close()

	// Just show the log files available
	return []models.ExplorerNode{
		{Key: "log:0", Label: "Current - ERRORLOG", Type: "log", Icon: "pi pi-file", Leaf: true},
		{Key: "log:1", Label: "Archive #1 - ERRORLOG.1", Type: "log", Icon: "pi pi-file", Leaf: true},
		{Key: "log:2", Label: "Archive #2 - ERRORLOG.2", Type: "log", Icon: "pi pi-file", Leaf: true},
	}, nil
}

func (s *SQLServerProvider) getTableConstraintNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name, type_desc FROM sys.objects 
	          WHERE parent_object_id = @p1 AND type IN ('C', 'D', 'F', 'PK', 'UQ') ORDER BY name`, ctx["database"])
	rows, err := db.Query(query, ctx["objectId"])
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name, typeDesc string
		if err := rows.Scan(&name, &typeDesc); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("constraint:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, typeDesc),
			Type:  "constraint",
			Icon:  "pi pi-lock",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getTableTriggerNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name FROM sys.triggers WHERE parent_id = @p1 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query, ctx["objectId"])
	if err != nil {
		return nil, fmt.Errorf("failed to query table triggers: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("tbtrigger:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: name,
			Type:  "trigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getTableStatisticNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name FROM sys.stats WHERE object_id = @p1 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query, ctx["objectId"])
	if err != nil {
		return nil, fmt.Errorf("failed to query statistics: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("stat:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: name,
			Type:  "statistic",
			Icon:  "pi pi-chart-bar",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getFunctionChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	return []models.ExplorerNode{
		{Key: "tf:" + dbName, Label: "Table-valued Functions", Type: "tableValuedFunctions", Icon: "pi pi-table", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "sf:" + dbName, Label: "Scalar-valued Functions", Type: "scalarValuedFunctions", Icon: "pi pi-code", Leaf: false, Data: map[string]interface{}{"database": dbName}},
	}, nil
}

func (s *SQLServerProvider) getTableValuedFunctionNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT SCHEMA_NAME(schema_id) AS schema_name, name FROM sys.objects WHERE type IN ('TF', 'IF', 'FT') AND is_ms_shipped = 0 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table-valued functions: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var schemaName, name string
		if err := rows.Scan(&schemaName, &name); err != nil {
			continue
		}
		fullName := schemaName + "." + name
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("tf:%s:%s", ctx["database"], fullName),
			Label: fullName,
			Type:  "function",
			Icon:  "pi pi-table",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getScalarValuedFunctionNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT SCHEMA_NAME(schema_id) AS schema_name, name FROM sys.objects WHERE type IN ('FN', 'FS') AND is_ms_shipped = 0 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scalar-valued functions: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var schemaName, name string
		if err := rows.Scan(&schemaName, &name); err != nil {
			continue
		}
		fullName := schemaName + "." + name
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("sf:%s:%s", ctx["database"], fullName),
			Label: fullName,
			Type:  "function",
			Icon:  "pi pi-code",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getSecurityChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	return []models.ExplorerNode{
		{Key: "users:" + dbName, Label: "Users", Type: "dbUsers", Icon: "pi pi-users", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "roles:" + dbName, Label: "Roles", Type: "dbRoles", Icon: "pi pi-users", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "schemas:" + dbName, Label: "Schemas", Type: "dbSchemas", Icon: "pi pi-folder", Leaf: false, Data: map[string]interface{}{"database": dbName}},
	}, nil
}

func (s *SQLServerProvider) getDbUserNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name, type_desc FROM sys.database_principals WHERE type IN ('S', 'U', 'G') AND principal_id > 4 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name, typeDesc string
		if err := rows.Scan(&name, &typeDesc); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("dbuser:%s:%s", ctx["database"], name),
			Label: fmt.Sprintf("%s (%s)", name, typeDesc),
			Type:  "user",
			Icon:  "pi pi-user",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getDbRoleChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	return []models.ExplorerNode{
		{Key: "dbRoles:" + dbName, Label: "Database Roles", Type: "databaseRoles", Icon: "pi pi-shield", Leaf: false, Data: map[string]interface{}{"database": dbName}},
		{Key: "appRoles:" + dbName, Label: "Application Roles", Type: "applicationRoles", Icon: "pi pi-shield", Leaf: false, Data: map[string]interface{}{"database": dbName}},
	}, nil
}

func (s *SQLServerProvider) getDatabaseRoleNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name FROM sys.database_principals WHERE type = 'R' AND principal_id > 0 ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query database roles: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("dbrole:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "role",
			Icon:  "pi pi-shield",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getApplicationRoleNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name FROM sys.database_principals WHERE type = 'A' ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query application roles: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("approle:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "role",
			Icon:  "pi pi-shield",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (s *SQLServerProvider) getDbSchemaNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := fmt.Sprintf(`USE [%s];
	          SELECT name FROM sys.schemas ORDER BY name`, ctx["database"])
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	nodes := make([]models.ExplorerNode, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("schema:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "schema",
			Icon:  "pi pi-folder",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func ParseScriptSections(content string) map[string]string {
	sections := make(map[string]string)

	parts := strings.Split(content, "-- name:")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		lines := strings.SplitN(part, "\n", 2)
		if len(lines) < 2 {
			continue
		}

		key := strings.TrimSpace(lines[0])
		query := strings.TrimSpace(lines[1])
		sections[key] = query
	}

	return sections
}
