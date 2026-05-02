package providers

import (
	"context"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisProvider struct {
	BaseProvider
}

// buildClient creates a Redis client based on config
func (r *RedisProvider) buildClient(config models.ConnectionConfig) *redis.Client {
	dbIdx := 0
	if config.Database != "" {
		if idx, err := strconv.Atoi(config.Database); err == nil {
			dbIdx = idx
		}
	}

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Username: config.User,
		Password: config.Password,
		DB:       dbIdx,
	})
}

func (r *RedisProvider) TestConnection(config models.ConnectionConfig) error {
	client := r.buildClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

func (r *RedisProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	client := r.buildClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Optionally execute lua script if requested, though redis logic doesn't strictly need one
	// Here we try read the script to mimic others
	scriptContent, err := os.ReadFile("./scripts/redis/metadata.lua")
	var metadata map[string]interface{}

	if err == nil && len(scriptContent) > 0 {
		// Run lua script if present
		res, err := client.Eval(ctx, string(scriptContent), []string{}).Result()
		if err == nil {
			if strRes, ok := res.(string); ok {
				var parsed interface{}
				json.Unmarshal([]byte(strRes), &parsed)
				metadata = map[string]interface{}{"databases": parsed}
			}
		}
	}

	if metadata == nil {
		// Fallback to fetch info
		infoStr := client.Info(ctx, "keyspace").Val()
		databases := []map[string]interface{}{}
		lines := strings.Split(infoStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "db") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					dbName := parts[0]
					databases = append(databases, map[string]interface{}{
						"name": dbName,
						"type": "database",
					})
				}
			}
		}
		metadata = map[string]interface{}{
			"databases": databases,
		}
	}

	sessionID, err := utils.StoreSession(config)
	if err != nil {
		return "", nil, err
	}

	rawJson, err := json.Marshal(metadata)
	if err != nil {
		return "", nil, err
	}

	return sessionID, json.RawMessage(rawJson), nil
}

func (r *RedisProvider) ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error) {
	client := r.buildClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	startTime := time.Now()

	// Try splitting the query by spaces for redis command executing
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return models.QueryResult{}, fmt.Errorf("empty query")
	}

	var args []interface{}
	for _, p := range parts {
		args = append(args, p)
	}

	res, err := client.Do(ctx, args...).Result()
	if err != nil {
		return models.QueryResult{}, err
	}

	executionTime := time.Since(startTime).Milliseconds()

	rows := []map[string]interface{}{
		{"result": fmt.Sprintf("%v", res)},
	}

	return models.QueryResult{
		Columns:       []string{"result"},
		Rows:          rows,
		RowCount:      1,
		ExecutionTime: executionTime,
	}, nil
}

func (r *RedisProvider) GetExplorerChildren(config models.ConnectionConfig, nodeType string, ctx map[string]string) ([]models.ExplorerNode, error) {
	if dbName, ok := ctx["database"]; ok && dbName != "" {
		if strings.HasPrefix(dbName, "db") {
			config.Database = strings.TrimPrefix(dbName, "db")
		} else {
			config.Database = dbName
		}
	}

	client := r.buildClient(config)
	defer client.Close()

	mctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch nodeType {
	case "root":
		return []models.ExplorerNode{
			{Key: "databases", Label: "Databases", Type: "databases", Icon: "pi pi-database", Leaf: false},
		}, nil
	case "databases":
		infoStr, err := client.Info(mctx, "keyspace").Result()
		if err != nil {
			return nil, err
		}
		var nodes []models.ExplorerNode
		lines := strings.Split(infoStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "db") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					dbName := parts[0]
					dbIdx := strings.TrimPrefix(dbName, "db")
					// Parse key count
					var keyCount string
					props := strings.Split(strings.TrimSpace(parts[1]), ",")
					for _, prop := range props {
						if strings.HasPrefix(prop, "keys=") {
							keyCount = strings.TrimPrefix(prop, "keys=")
						}
					}

					label := dbName
					if keyCount != "" && keyCount != "0" {
						label += fmt.Sprintf(" (%s keys)", keyCount)
					}

					nodes = append(nodes, models.ExplorerNode{
						Key:   "db:" + dbIdx,
						Label: label,
						Type:  "database",
						Icon:  "pi pi-server",
						Leaf:  false,
						Data:  map[string]interface{}{"database": dbIdx},
					})
				}
			}
		}
		return nodes, nil
	case "database":
		var cursor uint64
		var keys []string
		var err error
		var nodes []models.ExplorerNode
		// Limit to 1000 keys for exploration to prevent overload
		keys, _, err = client.Scan(mctx, cursor, "*", 1000).Result()
		if err != nil {
			return nil, err
		}

		for _, k := range keys {
			// type of key
			kType := client.Type(mctx, k).Val()
			icon := "pi pi-key"
			switch kType {
			case "hash":
				icon = "pi pi-list"
			case "list":
				icon = "pi pi-bars"
			case "set":
				icon = "pi pi-chart-pie"
			case "zset":
				icon = "pi pi-chart-bar"
			}

			nodes = append(nodes, models.ExplorerNode{
				Key:   fmt.Sprintf("key:%s:%s", config.Database, k),
				Label: fmt.Sprintf("%s (%s)", k, kType),
				Type:  "key",
				Icon:  icon,
				Leaf:  true,
				Data:  map[string]interface{}{"database": config.Database, "key": k},
			})
		}
		return nodes, nil
	default:
		return []models.ExplorerNode{}, nil
	}
}

// GetKeyValue retrieves the value of a Redis key depending on its type
func (r *RedisProvider) GetKeyValue(config models.ConnectionConfig, key string) (*models.RedisKeyValueResponse, error) {
	client := r.buildClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check key type
	kType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get key type: %w", err)
	}

	// Get TTL
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTL: %w", err)
	}
	ttlSeconds := int64(ttl.Seconds())
	switch ttl {
	case -1: // No expiry
		ttlSeconds = -1
	case -2: // Key does not exist
		ttlSeconds = -2
	}

	var value string
	switch kType {
	case "string":
		value, err = client.Get(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get string value: %w", err)
		}
	case "list":
		items, err := client.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get list value: %w", err)
		}
		jsonBytes, _ := json.Marshal(items)
		value = string(jsonBytes)
	case "set":
		items, err := client.SMembers(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get set value: %w", err)
		}
		jsonBytes, _ := json.Marshal(items)
		value = string(jsonBytes)
	case "zset":
		items, err := client.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get zset value: %w", err)
		}
		type zsetEntry struct {
			Member string  `json:"member"`
			Score  float64 `json:"score"`
		}
		var entries []zsetEntry
		for _, item := range items {
			entries = append(entries, zsetEntry{
				Member: fmt.Sprintf("%v", item.Member),
				Score:  item.Score,
			})
		}
		jsonBytes, _ := json.Marshal(entries)
		value = string(jsonBytes)
	case "hash":
		items, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get hash value: %w", err)
		}
		jsonBytes, _ := json.Marshal(items)
		value = string(jsonBytes)
	default:
		value = fmt.Sprintf("(unsupported type: %s)", kType)
	}

	return &models.RedisKeyValueResponse{
		Key:      key,
		Value:    value,
		Type:     kType,
		TTL:      ttlSeconds,
		Database: config.Database,
	}, nil
}

// SetKeyValue updates a Redis key's value (string type) and optionally renames it
func (r *RedisProvider) SetKeyValue(config models.ConnectionConfig, oldKey, newKey, value string, ttl int64) error {
	client := r.buildClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set the new value on the old key first
	if err := client.Set(ctx, oldKey, value, 0).Err(); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	// Rename key if needed
	if oldKey != newKey && newKey != "" {
		if err := client.Rename(ctx, oldKey, newKey).Err(); err != nil {
			return fmt.Errorf("failed to rename key: %w", err)
		}
	}

	// Set TTL
	finalKey := newKey
	if finalKey == "" {
		finalKey = oldKey
	}
	if ttl > 0 {
		client.Expire(ctx, finalKey, time.Duration(ttl)*time.Second)
	} else if ttl == -1 {
		client.Persist(ctx, finalKey)
	}

	return nil
}

// DeleteKey removes a Redis key
func (r *RedisProvider) DeleteKey(config models.ConnectionConfig, key string) error {
	client := r.buildClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return client.Del(ctx, key).Err()
}
