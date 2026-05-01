package providers

import (
	"database/sql"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	mysqlDriver     = "mysql"
	mysqlTimeout    = 5 * time.Second
	mysqlScriptPath = "./scripts/mysql/metadata.sql"
)

type MySQLProvider struct {
	BaseProvider
}

func (m *MySQLProvider) buildConnectionString(config models.ConnectionConfig) string {
	// format: user:password@tcp(host:port)/dbname?parseTime=true&multiStatements=true
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true", config.User, config.Password, config.Host, config.Port, config.Database)
}

func (m *MySQLProvider) TestConnection(config models.ConnectionConfig) error {
	connStr := m.buildConnectionString(config)

	db, err := m.OpenAndValidate(mysqlDriver, connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	return m.TestConnectionWithTimeout(db, mysqlTimeout)
}

func (m *MySQLProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	connStr := m.buildConnectionString(config)

	db, err := m.OpenAndValidate(mysqlDriver, connStr)
	if err != nil {
		return "", nil, err
	}
	defer db.Close()

	scriptContent, err := os.ReadFile(mysqlScriptPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read script: %w", err)
	}

	rawJson, err := m.ExecuteScript(db, string(scriptContent))
	if err != nil {
		return "", nil, err
	}

	sessionID, err := utils.StoreSession(config)
	if err != nil {
		return "", nil, err
	}

	return sessionID, json.RawMessage(rawJson), nil
}

func (m *MySQLProvider) ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error) {
	connStr := m.buildConnectionString(config)

	db, err := m.OpenAndValidate(mysqlDriver, connStr)
	if err != nil {
		return models.QueryResult{}, err
	}
	defer db.Close()

	return utils.ExecuteDynamicQuery(db, query)
}

func (m *MySQLProvider) GetExplorerChildren(config models.ConnectionConfig, nodeType string, ctx map[string]string) ([]models.ExplorerNode, error) {
	if dbName, ok := ctx["database"]; ok && dbName != "" {
		config.Database = dbName
	}

	connStr := m.buildConnectionString(config)

	db, err := m.OpenAndValidate(mysqlDriver, connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	switch nodeType {
	case "root":
		return m.myGetRootNodes()
	case "databases":
		return m.myGetDatabaseNodes(db)
	case "database":
		return m.myGetDatabaseChildFolders(ctx)
	case "tables":
		return m.myGetTableNodes(db, ctx)
	case "table":
		return m.myGetTableChildFolders(ctx)
	case "tableColumns":
		return m.myGetTableColumnNodes(db, ctx)
	case "tableKeys":
		return m.myGetTableKeyNodes(db, ctx)
	case "tableConstraints":
		return m.myGetTableConstraintNodes(db, ctx)
	case "tableIndexes":
		return m.myGetTableIndexNodes(db, ctx)
	case "tableTriggers":
		return m.myGetTableTriggerNodes(db, ctx)
	case "views":
		return m.myGetViewNodes(db, ctx)
	case "functions":
		return m.myGetFunctionNodes(db, ctx)
	case "procedures":
		return m.myGetProcedureNodes(db, ctx)
	case "databaseTriggers":
		return m.myGetDatabaseTriggerNodes(db, ctx)
	case "events":
		return m.myGetEventNodes(db, ctx)
	case "users":
		return m.myGetUserNodes(db)
	case "variables":
		return m.myGetVariableNodes(db)
	default:
		return []models.ExplorerNode{}, nil
	}
}

func (m *MySQLProvider) myGetRootNodes() ([]models.ExplorerNode, error) {
	return []models.ExplorerNode{
		{Key: "databases", Label: "Databases", Type: "databases", Icon: "pi pi-database", Leaf: false},
		{Key: "users", Label: "Users", Type: "users", Icon: "pi pi-users", Leaf: false},
		{Key: "variables", Label: "System Variables", Type: "variables", Icon: "pi pi-cog", Leaf: false},
	}, nil
}

func (m *MySQLProvider) myGetDatabaseNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SHOW DATABASES`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %w", err)
	}
	defer rows.Close()

	var systemDBs []models.ExplorerNode
	var userDBs []models.ExplorerNode

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}

		node := models.ExplorerNode{
			Key:   "db:" + name,
			Label: name,
			Type:  "database",
			Icon:  "pi pi-database",
			Leaf:  false,
			Data:  map[string]string{"database": name},
		}

		if name == "information_schema" || name == "mysql" || name == "performance_schema" || name == "sys" {
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

func (m *MySQLProvider) myGetDatabaseChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	return []models.ExplorerNode{
		{Key: "tables:" + dbName, Label: "Tables", Type: "tables", Icon: "pi pi-table", Leaf: false, Data: map[string]string{"database": dbName}},
		{Key: "views:" + dbName, Label: "Views", Type: "views", Icon: "pi pi-eye", Leaf: false, Data: map[string]string{"database": dbName}},
		{Key: "functions:" + dbName, Label: "Functions", Type: "functions", Icon: "pi pi-code", Leaf: false, Data: map[string]string{"database": dbName}},
		{Key: "procedures:" + dbName, Label: "Procedures", Type: "procedures", Icon: "pi pi-cog", Leaf: false, Data: map[string]string{"database": dbName}},
		{Key: "triggers:" + dbName, Label: "Triggers", Type: "databaseTriggers", Icon: "pi pi-bolt", Leaf: false, Data: map[string]string{"database": dbName}},
		{Key: "events:" + dbName, Label: "Events", Type: "events", Icon: "pi pi-calendar", Leaf: false, Data: map[string]string{"database": dbName}},
	}, nil
}

func (m *MySQLProvider) myGetTableNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT TABLE_NAME FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE' ORDER BY TABLE_NAME`
	rows, err := db.Query(query, ctx["database"])
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
			Key:   fmt.Sprintf("table:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "table",
			Icon:  "pi pi-table",
			Leaf:  false,
			Data:  map[string]string{"database": ctx["database"], "table": name},
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetTableChildFolders(ctx map[string]string) ([]models.ExplorerNode, error) {
	dbName := ctx["database"]
	table := ctx["table"]
	return []models.ExplorerNode{
		{Key: fmt.Sprintf("columns:%s:%s", dbName, table), Label: "Columns", Type: "tableColumns", Icon: "pi pi-list", Leaf: false, Data: map[string]string{"database": dbName, "table": table}},
		{Key: fmt.Sprintf("keys:%s:%s", dbName, table), Label: "Keys", Type: "tableKeys", Icon: "pi pi-key", Leaf: false, Data: map[string]string{"database": dbName, "table": table}},
		{Key: fmt.Sprintf("constraints:%s:%s", dbName, table), Label: "Constraints", Type: "tableConstraints", Icon: "pi pi-lock", Leaf: false, Data: map[string]string{"database": dbName, "table": table}},
		{Key: fmt.Sprintf("indexes:%s:%s", dbName, table), Label: "Indexes", Type: "tableIndexes", Icon: "pi pi-sort-alt", Leaf: false, Data: map[string]string{"database": dbName, "table": table}},
		{Key: fmt.Sprintf("triggers:%s:%s", dbName, table), Label: "Triggers", Type: "tableTriggers", Icon: "pi pi-bolt", Leaf: false, Data: map[string]string{"database": dbName, "table": table}},
	}, nil
}

func (m *MySQLProvider) myGetTableColumnNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ORDINAL_POSITION`
	rows, err := db.Query(query, ctx["database"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, dataType, isNullableStr string
		if err := rows.Scan(&name, &dataType, &isNullableStr); err != nil {
			continue
		}
		nullable := ""
		if isNullableStr == "YES" {
			nullable = " (nullable)"
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("col:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)%s", name, dataType, nullable),
			Type:  "column",
			Icon:  "pi pi-minus",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetTableIndexNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SHOW INDEXES FROM ` + "`" + ctx["table"] + "`" + ` FROM ` + "`" + ctx["database"] + "`"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	// Columns in SHOW INDEXES: Table, Non_unique, Key_name, Seq_in_index, Column_name, Collation, Cardinality, Sub_part, Packed, Null, Index_type, Comment, Index_comment
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	vals := make([]interface{}, len(cols))
	for i := range vals {
		var v sql.RawBytes
		vals[i] = &v
	}

	indexMap := make(map[string]bool)
	var nodes []models.ExplorerNode

	for rows.Next() {
		if err := rows.Scan(vals...); err != nil {
			continue
		}

		var keyName, indexType string
		for i, col := range cols {
			switch col {
			case "Key_name":
				val := vals[i].(*sql.RawBytes)
				keyName = string(*val)
			case "Index_type":
				val := vals[i].(*sql.RawBytes)
				indexType = string(*val)
			}
		}

		if _, exists := indexMap[keyName]; !exists {
			indexMap[keyName] = true
			nodes = append(nodes, models.ExplorerNode{
				Key:   fmt.Sprintf("idx:%s:%s:%s", ctx["database"], ctx["table"], keyName),
				Label: fmt.Sprintf("%s (%s)", keyName, indexType),
				Type:  "index",
				Icon:  "pi pi-sort-alt",
				Leaf:  true,
			})
		}
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetViewNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT TABLE_NAME FROM information_schema.views WHERE table_schema = ? ORDER BY TABLE_NAME`
	rows, err := db.Query(query, ctx["database"])
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
			Key:   fmt.Sprintf("view:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "view",
			Icon:  "pi pi-eye",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetFunctionNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT ROUTINE_NAME FROM information_schema.routines WHERE routine_schema = ? AND routine_type = 'FUNCTION' ORDER BY ROUTINE_NAME`
	rows, err := db.Query(query, ctx["database"])
	if err != nil {
		return nil, fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("func:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "function",
			Icon:  "pi pi-code",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetProcedureNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT ROUTINE_NAME FROM information_schema.routines WHERE routine_schema = ? AND routine_type = 'PROCEDURE' ORDER BY ROUTINE_NAME`
	rows, err := db.Query(query, ctx["database"])
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
			Key:   fmt.Sprintf("proc:%s:%s", ctx["database"], name),
			Label: name,
			Type:  "procedure",
			Icon:  "pi pi-cog",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetUserNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SELECT User, Host FROM mysql.user ORDER BY User`
	rows, err := db.Query(query)
	if err != nil {
		// Possibly no permission
		return []models.ExplorerNode{}, nil
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var user, host string
		if err := rows.Scan(&user, &host); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("user:%s@%s", user, host),
			Label: fmt.Sprintf("%s@%s", user, host),
			Type:  "user",
			Icon:  "pi pi-user",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetTableKeyNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT CONSTRAINT_NAME, CONSTRAINT_TYPE
	          FROM information_schema.TABLE_CONSTRAINTS
	          WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	            AND CONSTRAINT_TYPE IN ('PRIMARY KEY', 'UNIQUE', 'FOREIGN KEY')
	          ORDER BY CONSTRAINT_NAME`
	rows, err := db.Query(query, ctx["database"], ctx["table"])
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
			Key:   fmt.Sprintf("key:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, conType),
			Type:  "key",
			Icon:  "pi pi-key",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetTableConstraintNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT CONSTRAINT_NAME, CONSTRAINT_TYPE
	          FROM information_schema.TABLE_CONSTRAINTS
	          WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	          ORDER BY CONSTRAINT_NAME`
	rows, err := db.Query(query, ctx["database"], ctx["table"])
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
			Key:   fmt.Sprintf("constraint:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s)", name, conType),
			Type:  "constraint",
			Icon:  "pi pi-lock",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetTableTriggerNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT TRIGGER_NAME, EVENT_MANIPULATION, ACTION_TIMING
	          FROM information_schema.TRIGGERS
	          WHERE TRIGGER_SCHEMA = ? AND EVENT_OBJECT_TABLE = ?
	          ORDER BY TRIGGER_NAME`
	rows, err := db.Query(query, ctx["database"], ctx["table"])
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, event, timing string
		if err := rows.Scan(&name, &event, &timing); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("tbtrigger:%s:%s:%s", ctx["database"], ctx["table"], name),
			Label: fmt.Sprintf("%s (%s %s)", name, timing, event),
			Type:  "trigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetDatabaseTriggerNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE, EVENT_MANIPULATION, ACTION_TIMING
	          FROM information_schema.TRIGGERS
	          WHERE TRIGGER_SCHEMA = ?
	          ORDER BY TRIGGER_NAME`
	rows, err := db.Query(query, ctx["database"])
	if err != nil {
		return nil, fmt.Errorf("failed to query database triggers: %w", err)
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, table, event, timing string
		if err := rows.Scan(&name, &table, &event, &timing); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("dbtrigger:%s:%s", ctx["database"], name),
			Label: fmt.Sprintf("%s on %s (%s %s)", name, table, timing, event),
			Type:  "trigger",
			Icon:  "pi pi-bolt",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetEventNodes(db *sql.DB, ctx map[string]string) ([]models.ExplorerNode, error) {
	query := `SELECT EVENT_NAME, STATUS, EVENT_TYPE
	          FROM information_schema.EVENTS
	          WHERE EVENT_SCHEMA = ?
	          ORDER BY EVENT_NAME`
	rows, err := db.Query(query, ctx["database"])
	if err != nil {
		// Events may not be available in all MySQL editions
		return []models.ExplorerNode{}, nil
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, status, eventType string
		if err := rows.Scan(&name, &status, &eventType); err != nil {
			continue
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   fmt.Sprintf("event:%s:%s", ctx["database"], name),
			Label: fmt.Sprintf("%s (%s, %s)", name, eventType, status),
			Type:  "event",
			Icon:  "pi pi-calendar",
			Leaf:  true,
		})
	}
	return nodes, nil
}

func (m *MySQLProvider) myGetVariableNodes(db *sql.DB) ([]models.ExplorerNode, error) {
	query := `SHOW VARIABLES`
	rows, err := db.Query(query)
	if err != nil {
		return []models.ExplorerNode{}, nil
	}
	defer rows.Close()

	var nodes []models.ExplorerNode
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		valPreview := value
		if len(valPreview) > 30 {
			valPreview = valPreview[:27] + "..."
		}
		nodes = append(nodes, models.ExplorerNode{
			Key:   "var:" + name,
			Label: fmt.Sprintf("%s = %s", name, valPreview),
			Type:  "variable",
			Icon:  "pi pi-cog",
			Leaf:  true,
		})
	}
	return nodes, nil
}
