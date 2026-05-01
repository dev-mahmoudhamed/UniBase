package providers

import (
	"database/sql"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

const (
	postgresDriver     = "postgres"
	postgresDefaultSSL = "disable"
	postgresTimeout    = 5 * time.Second
	postgresScriptPath = "./scripts/pgsql/metadata.sql"
)

type PostgresProvider struct {
	BaseProvider
}

// buildConnectionString creates a PostgreSQL connection string
func (p *PostgresProvider) buildConnectionString(config models.ConnectionConfig) string {
	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = postgresDefaultSSL
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, sslMode)
}

func (p *PostgresProvider) TestConnection(config models.ConnectionConfig) error {
	connStr := p.buildConnectionString(config)

	db, err := p.OpenAndValidate(postgresDriver, connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	return p.TestConnectionWithTimeout(db, postgresTimeout)
}

func (p *PostgresProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	connStr := p.buildConnectionString(config)

	db, err := p.OpenAndValidate(postgresDriver, connStr)
	if err != nil {
		return "", nil, err
	}
	defer db.Close()

	scriptContent, err := os.ReadFile(postgresScriptPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read script: %w", err)
	}

	rawJson, err := p.ExecuteScript(db, string(scriptContent))
	if err != nil {
		return "", nil, err
	}

	sessionID, err := utils.StoreSession(config)
	if err != nil {
		return "", nil, err
	}

	return sessionID, json.RawMessage(rawJson), nil
}

func (p *PostgresProvider) ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error) {
	connStr := p.buildConnectionString(config)

	db, err := p.OpenAndValidate(postgresDriver, connStr)
	if err != nil {
		return models.QueryResult{}, err
	}
	defer db.Close()

	return utils.ExecuteDynamicQuery(db, query)
}

func (p *PostgresProvider) GetExplorerChildren(config models.ConnectionConfig, nodeType string, ctx map[string]string) ([]models.ExplorerNode, error) {
	// Let's connect to the specific database if one is in the context
	if dbName, ok := ctx["database"]; ok && dbName != "" {
		config.Database = dbName
	}

	connStr := p.buildConnectionString(config)

	db, err := p.OpenAndValidate(postgresDriver, connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	switch nodeType {
	case "root":
		return p.pgGetRootNodes()
	case "databases":
		return p.pgGetDatabaseNodes(db)
	case "database":
		return p.pgGetSchemaNodes(db)
	case "schema":
		return p.pgGetSchemaChildFolders(ctx)
	case "tables":
		return p.pgGetTableNodes(db, ctx)
	case "table":
		return p.pgGetTableChildFolders(ctx)
	case "tableColumns":
		return p.pgGetTableColumnNodes(db, ctx)
	case "tableKeys":
		return p.pgGetTableKeyNodes(db, ctx)
	case "tableConstraints":
		return p.pgGetTableConstraintNodes(db, ctx)
	case "tableIndexes":
		return p.pgGetTableIndexNodes(db, ctx)
	case "tableTriggers":
		return p.pgGetTableTriggerNodes(db, ctx)
	case "views":
		return p.pgGetViewNodes(db, ctx)
	case "functions":
		return p.pgGetFunctionNodes(db, ctx)
	case "procedures":
		return p.pgGetProcedureNodes(db, ctx)
	case "schemaTriggers":
		return p.pgGetSchemaTriggerNodes(db, ctx)
	case "roles":
		return p.pgGetRoleNodes(db)
	case "tablespaces":
		return p.pgGetTablespaceNodes(db)
	case "extensions":
		return p.pgGetExtensionNodes(db)
	default:
		return []models.ExplorerNode{}, nil
	}
}

func (p *PostgresProvider) pgGetRootNodes() ([]models.ExplorerNode, error) {
	return []models.ExplorerNode{
		{Key: "databases", Label: "Databases", Type: "databases", Icon: "pi pi-database", Leaf: false},
		{Key: "roles", Label: "Roles", Type: "roles", Icon: "pi pi-users", Leaf: false},
		{Key: "tablespaces", Label: "Tablespaces", Type: "tablespaces", Icon: "pi pi-server", Leaf: false},
		{Key: "extensions", Label: "Extensions", Type: "extensions", Icon: "pi pi-box", Leaf: false},
	}, nil
}

func (p *PostgresProvider) pgGetDatabaseNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT datname, datistemplate FROM pg_database WHERE datallowconn = true ORDER BY datname`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %w", err)
	}
	defer rows.Close()

	var systemDBs []models.ExplorerNode
	var userDBs []models.ExplorerNode

	for rows.Next() {
		var name string
		var isTemplate bool
		if err := rows.Scan(&name, &isTemplate); err != nil {
			continue
		}

		node := models.ExplorerNode{
			Key:   "db:" + name,
			Label: name,
			Type:  "database",
			Icon:  "pi pi-database",
			Leaf:  false,
			Data:  map[string]interface{}{"database": name},
		}

		if isTemplate || name == "postgres" || name == "rdsadmin" || name == "azure_maintenance" || name == "azure_sys" {
			systemDBs = append(systemDBs, node)
		} else {
			userDBs = append(userDBs, node)
		}
	}

	var nodes []models.ExplorerNode
	if len(systemDBs) > 0 {
		nodes = append(nodes, models.ExplorerNode{
			Key:      "systemDatabases",
			Label:    "System Databases",
			Type:     "folder",
			Icon:     "pi pi-folder",
			Leaf:     true,
			Children: systemDBs,
		})
	}
	nodes = append(nodes, userDBs...)

	return nodes, nil
}

func (p *PostgresProvider) pgGetSchemaNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT nspname FROM pg_namespace 
	          WHERE nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast') 
	            AND nspname NOT LIKE 'pg_temp_%' 
	            AND nspname NOT LIKE 'pg_toast_temp_%'
	          ORDER BY nspname`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "schema:" + name,
			Label: name,
			Type:  "schema",
			Icon:  "pi pi-folder",
			Leaf:  false,
			Data:  map[string]interface{}{"schema": name},
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetSchemaChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	schema := ctx["schema"]
	return []models.ExplorerNode{
		{Key: "tables:" + schema, Label: "Tables", Type: "tables", Icon: "pi pi-table", Leaf: false, Data: map[string]interface{}{"schema": schema}},
		{Key: "views:" + schema, Label: "Views", Type: "views", Icon: "pi pi-eye", Leaf: false, Data: map[string]interface{}{"schema": schema}},
		{Key: "functions:" + schema, Label: "Functions", Type: "functions", Icon: "pi pi-code", Leaf: false, Data: map[string]interface{}{"schema": schema}},
		{Key: "procedures:" + schema, Label: "Procedures", Type: "procedures", Icon: "pi pi-cog", Leaf: false, Data: map[string]interface{}{"schema": schema}},
		{Key: "schemaTriggers:" + schema, Label: "Triggers", Type: "schemaTriggers", Icon: "pi pi-bolt", Leaf: false, Data: map[string]interface{}{"schema": schema}},
	}, nil
}

func (p *PostgresProvider) pgGetTableNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT c.relname FROM pg_class c 
	          JOIN pg_namespace n ON c.relnamespace = n.oid 
	          WHERE n.nspname = $1 AND c.relkind = 'r' 
	          ORDER BY c.relname`
	rows, err := db.Query(query, ctx["schema"])
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("table:%s:%s", ctx["schema"], name),
			Label: name,
			Type:  "table",
			Icon:  "pi pi-table",
			Leaf:  false,
			Data:  map[string]interface{}{"schema": ctx["schema"], "table": name},
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetTableChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	schema := ctx["schema"]
	table := ctx["table"]
	return []models.ExplorerNode{
		{Key: fmt.Sprintf("columns:%s:%s", schema, table), Label: "Columns", Type: "tableColumns", Icon: "pi pi-list", Leaf: false, Data: map[string]interface{}{"schema": schema, "table": table}},
		{Key: fmt.Sprintf("keys:%s:%s", schema, table), Label: "Keys", Type: "tableKeys", Icon: "pi pi-key", Leaf: false, Data: map[string]interface{}{"schema": schema, "table": table}},
		{Key: fmt.Sprintf("constraints:%s:%s", schema, table), Label: "Constraints", Type: "tableConstraints", Icon: "pi pi-lock", Leaf: false, Data: map[string]interface{}{"schema": schema, "table": table}},
		{Key: fmt.Sprintf("indexes:%s:%s", schema, table), Label: "Indexes", Type: "tableIndexes", Icon: "pi pi-sort-alt", Leaf: false, Data: map[string]interface{}{"schema": schema, "table": table}},
		{Key: fmt.Sprintf("triggers:%s:%s", schema, table), Label: "Triggers", Type: "tableTriggers", Icon: "pi pi-bolt", Leaf: false, Data: map[string]interface{}{"schema": schema, "table": table}},
	}, nil
}

func (p *PostgresProvider) pgGetTableColumnNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT a.attname, format_type(a.atttypid, a.atttypmod), NOT a.attnotnull AS is_nullable
	          FROM pg_attribute a
	          JOIN pg_class c ON a.attrelid = c.oid
	          JOIN pg_namespace n ON c.relnamespace = n.oid
	          WHERE n.nspname = $1 AND c.relname = $2 AND a.attnum > 0 AND NOT a.attisdropped
	          ORDER BY a.attnum`
	rows, err := db.Query(query, ctx["schema"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, dataType string
		var isNullable bool
		if err := rows.Scan(&name, &dataType, &isNullable); err != nil {
			continue
		}
		nullable := ""
		if isNullable {
			nullable = " (nullable)"
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("col:%s:%s:%s", ctx["schema"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)%s", name, dataType, nullable),
			Type:  "column",
			Icon:  "pi pi-minus",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetTableConstraintNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT con.conname, 
	            CASE con.contype 
	              WHEN 'p' THEN 'PRIMARY KEY'
	              WHEN 'f' THEN 'FOREIGN KEY'
	              WHEN 'u' THEN 'UNIQUE'
	              WHEN 'c' THEN 'CHECK'
	              ELSE con.contype::text 
	            END AS constraint_type
	          FROM pg_constraint con
	          JOIN pg_class c ON con.conrelid = c.oid
	          JOIN pg_namespace n ON c.relnamespace = n.oid
	          WHERE n.nspname = $1 AND c.relname = $2
	          ORDER BY con.conname`
	rows, err := db.Query(query, ctx["schema"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, conType string
		if err := rows.Scan(&name, &conType); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("constraint:%s:%s:%s", ctx["schema"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, conType),
			Type:  "constraint",
			Icon:  "pi pi-key",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetTableIndexNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT i.relname
	          FROM pg_index idx
	          JOIN pg_class i ON i.oid = idx.indexrelid
	          JOIN pg_class c ON c.oid = idx.indrelid
	          JOIN pg_namespace n ON c.relnamespace = n.oid
	          WHERE n.nspname = $1 AND c.relname = $2
	          ORDER BY i.relname`
	rows, err := db.Query(query, ctx["schema"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("idx:%s:%s:%s", ctx["schema"], ctx["table"], name),
			Label: name,
			Type:  "index",
			Icon:  "pi pi-sort-alt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetViewNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT c.relname FROM pg_class c 
	          JOIN pg_namespace n ON c.relnamespace = n.oid 
	          WHERE n.nspname = $1 AND c.relkind = 'v' 
	          ORDER BY c.relname`
	rows, err := db.Query(query, ctx["schema"])
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("view:%s:%s", ctx["schema"], name),
			Label: name,
			Type:  "view",
			Icon:  "pi pi-eye",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetFunctionNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT p.proname, format_type(p.prorettype, null) AS return_type
	          FROM pg_proc p
	          JOIN pg_namespace n ON p.pronamespace = n.oid
	          WHERE n.nspname = $1 AND p.prokind = 'f'
	          ORDER BY p.proname`
	rows, err := db.Query(query, ctx["schema"])
	if err != nil {
		return nil, fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, retType string
		if err := rows.Scan(&name, &retType); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("func:%s:%s", ctx["schema"], name),
			Label: fmt.Sprintf("%s → %s", name, retType),
			Type:  "function",
			Icon:  "pi pi-code",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetProcedureNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT p.proname
	          FROM pg_proc p
	          JOIN pg_namespace n ON p.pronamespace = n.oid
	          WHERE n.nspname = $1 AND p.prokind = 'p'
	          ORDER BY p.proname`
	rows, err := db.Query(query, ctx["schema"])
	if err != nil {
		return nil, fmt.Errorf("failed to query procedures: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("proc:%s:%s", ctx["schema"], name),
			Label: name,
			Type:  "procedure",
			Icon:  "pi pi-cog",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetRoleNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT rolname, rolsuper, rolcanlogin FROM pg_roles ORDER BY rolname`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		var isSuper, canLogin bool
		if err := rows.Scan(&name, &isSuper, &canLogin); err != nil {
			continue
		}
		roleType := "Role"
		if isSuper {
			roleType = "Superuser"
		} else if canLogin {
			roleType = "Login"
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "role:" + name,
			Label: fmt.Sprintf("%s (%s)", name, roleType),
			Type:  "role",
			Icon:  "pi pi-user",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetTablespaceNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT spcname FROM pg_tablespace ORDER BY spcname`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tablespaces: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "tablespace:" + name,
			Label: name,
			Type:  "tablespace",
			Icon:  "pi pi-server",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetExtensionNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT extname, extversion FROM pg_extension ORDER BY extname`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query extensions: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, version string
		if err := rows.Scan(&name, &version); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "ext:" + name,
			Label: fmt.Sprintf("%s (v%s)", name, version),
			Type:  "extension",
			Icon:  "pi pi-box",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetTableKeyNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT con.conname,
	            CASE con.contype
	              WHEN 'p' THEN 'PRIMARY KEY'
	              WHEN 'f' THEN 'FOREIGN KEY'
	              WHEN 'u' THEN 'UNIQUE'
	              ELSE con.contype::text
	            END AS constraint_type
	          FROM pg_constraint con
	          JOIN pg_class c ON con.conrelid = c.oid
	          JOIN pg_namespace n ON c.relnamespace = n.oid
	          WHERE n.nspname = $1 AND c.relname = $2
	            AND con.contype IN ('p', 'f', 'u')
	          ORDER BY con.conname`
	rows, err := db.Query(query, ctx["schema"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, conType string
		if err := rows.Scan(&name, &conType); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("key:%s:%s:%s", ctx["schema"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, conType),
			Type:  "key",
			Icon:  "pi pi-key",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetTableTriggerNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT t.tgname,
	            CASE t.tgtype & 2 WHEN 2 THEN 'BEFORE' ELSE 'AFTER' END AS timing,
	            CASE
	              WHEN t.tgtype & 4  = 4 THEN 'INSERT'
	              WHEN t.tgtype & 8  = 8 THEN 'DELETE'
	              WHEN t.tgtype & 16 = 16 THEN 'UPDATE'
	              ELSE 'UNKNOWN'
	            END AS event
	          FROM pg_trigger t
	          JOIN pg_class c ON t.tgrelid = c.oid
	          JOIN pg_namespace n ON c.relnamespace = n.oid
	          WHERE n.nspname = $1 AND c.relname = $2
	            AND NOT t.tgisinternal
	          ORDER BY t.tgname`
	rows, err := db.Query(query, ctx["schema"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, timing, event string
		if err := rows.Scan(&name, &timing, &event); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("tbtrigger:%s:%s:%s", ctx["schema"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s %s)", name, timing, event),
			Type:  "trigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (p *PostgresProvider) pgGetSchemaTriggerNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT t.tgname, c.relname AS table_name,
	            CASE t.tgtype & 2 WHEN 2 THEN 'BEFORE' ELSE 'AFTER' END AS timing,
	            CASE
	              WHEN t.tgtype & 4  = 4 THEN 'INSERT'
	              WHEN t.tgtype & 8  = 8 THEN 'DELETE'
	              WHEN t.tgtype & 16 = 16 THEN 'UPDATE'
	              ELSE 'UNKNOWN'
	            END AS event
	          FROM pg_trigger t
	          JOIN pg_class c ON t.tgrelid = c.oid
	          JOIN pg_namespace n ON c.relnamespace = n.oid
	          WHERE n.nspname = $1
	            AND NOT t.tgisinternal
	          ORDER BY t.tgname`
	rows, err := db.Query(query, ctx["schema"])
	if err != nil {
		return nil, fmt.Errorf("failed to query schema triggers: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, tableName, timing, event string
		if err := rows.Scan(&name, &tableName, &timing, &event); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("trigger:%s:%s", ctx["schema"], name),
			Label: fmt.Sprintf("%s on %s (%s %s)", name, tableName, timing, event),
			Type:  "trigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}
