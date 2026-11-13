package providers

import (
	"context"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisKeyLimit caps how many keys one database node lists in the explorer.
const redisKeyLimit = 1000

type RedisProvider struct {
	BaseProvider
}

// client returns the pooled client for the connection's database index ("3" or "db3").
func (r *RedisProvider) client(config models.ConnectionConfig) *redis.Client {
	db, _ := strconv.Atoi(strings.TrimPrefix(config.Database, "db"))
	return redisClient(&redis.Options{
		Addr:     net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Username: config.User,
		Password: config.Password,
		DB:       db,
	})
}

func (r *RedisProvider) TestConnection(config models.ConnectionConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := r.client(config).Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

func (r *RedisProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	if err := r.TestConnection(config); err != nil {
		return "", nil, err
	}
	sessionID, err := utils.StoreSession(config)
	return sessionID, nil, err
}

func (r *RedisProvider) ExecuteQuery(ctx context.Context, config models.ConnectionConfig, query string) (models.QueryResult, error) {
	startTime := time.Now()

	parts := strings.Fields(query)
	if len(parts) == 0 {
		return models.QueryResult{}, fmt.Errorf("empty query")
	}
	args := make([]interface{}, len(parts))
	for i, p := range parts {
		args[i] = p
	}

	res, err := r.client(config).Do(ctx, args...).Result()
	if err != nil {
		return models.QueryResult{}, err
	}

	return models.QueryResult{
		ResultSets: []models.ResultSet{{
			Columns:  []string{"result"},
			Rows:     [][]interface{}{{fmt.Sprintf("%v", res)}},
			RowCount: 1,
		}},
		ExecutionTime: time.Since(startTime).Milliseconds(),
	}, nil
}

func (r *RedisProvider) GetExplorerChildren(config models.ConnectionConfig, nodeType string, ctx map[string]string) ([]models.ExplorerNode, error) {
	if dbName := ctx["database"]; dbName != "" {
		config.Database = strings.TrimPrefix(dbName, "db")
	}
	c := r.client(config)

	mctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch nodeType {
	case "root":
		return []models.ExplorerNode{
			{Key: "databases", Label: "Databases", Type: "databases", Icon: "pi pi-database", Leaf: false},
		}, nil
	case "databases":
		return r.databaseNodes(mctx, c)
	case "database":
		return r.keyNodes(mctx, c, config.Database)
	default:
		return []models.ExplorerNode{}, nil
	}
}

// databaseNodes lists the databases that hold keys, from INFO keyspace ("db0:keys=12,expires=0,...").
func (r *RedisProvider) databaseNodes(ctx context.Context, c *redis.Client) ([]models.ExplorerNode, error) {
	info, err := c.Info(ctx, "keyspace").Result()
	if err != nil {
		return nil, err
	}
	nodes := []models.ExplorerNode{}
	for _, line := range strings.Split(info, "\n") {
		name, stats, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || !strings.HasPrefix(name, "db") {
			continue
		}
		idx := strings.TrimPrefix(name, "db")
		label := name
		for _, prop := range strings.Split(stats, ",") {
			if n, ok := strings.CutPrefix(prop, "keys="); ok {
				label = fmt.Sprintf("%s (%s keys)", name, n)
			}
		}
		nodes = append(nodes, models.ExplorerNode{
			Key: "db:" + idx, Label: label, Type: "database", Icon: "pi pi-server",
			Data: map[string]interface{}{"database": idx},
		})
	}
	return nodes, nil
}

// keyNodes lists up to redisKeyLimit keys, sorted, with their types fetched in one round trip.
func (r *RedisProvider) keyNodes(ctx context.Context, c *redis.Client, db string) ([]models.ExplorerNode, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := c.Scan(ctx, cursor, "*", 500).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 || len(keys) >= redisKeyLimit {
			break
		}
	}
	truncated := cursor != 0 || len(keys) > redisKeyLimit
	if len(keys) > redisKeyLimit {
		keys = keys[:redisKeyLimit]
	}
	sort.Strings(keys)

	types := make([]*redis.StatusCmd, len(keys))
	pipe := c.Pipeline()
	for i, k := range keys {
		types[i] = pipe.Type(ctx, k)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	icons := map[string]string{"hash": "pi pi-list", "list": "pi pi-bars", "set": "pi pi-chart-pie", "zset": "pi pi-chart-bar"}
	nodes := make([]models.ExplorerNode, 0, len(keys)+1)
	for i, k := range keys {
		kType := types[i].Val()
		icon := icons[kType]
		if icon == "" {
			icon = "pi pi-key"
		}
		nodes = append(nodes, models.ExplorerNode{
			Key: fmt.Sprintf("key:%s:%s", db, k), Label: fmt.Sprintf("%s (%s)", k, kType), Type: "key", Icon: icon, Leaf: true,
			Data: map[string]interface{}{"database": db, "key": k},
		})
	}
	if truncated {
		nodes = append(nodes, models.ExplorerNode{
			Key: "more:" + db, Label: fmt.Sprintf("Showing the first %d keys", redisKeyLimit), Type: "info", Icon: "pi pi-info-circle", Leaf: true,
		})
	}
	return nodes, nil
}

// GetKeyValue returns a key's value as text: strings as-is, other types as JSON.
func (r *RedisProvider) GetKeyValue(config models.ConnectionConfig, key string) (*models.RedisKeyValueResponse, error) {
	c := r.client(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kType, err := c.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get key type: %w", err)
	}
	if kType == "none" {
		return nil, fmt.Errorf("key %q does not exist (it may have expired or been deleted)", key)
	}

	ttl, err := c.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTL: %w", err)
	}
	ttlSeconds := int64(ttl.Seconds())
	if ttl < 0 {
		ttlSeconds = -1
	}

	var value interface{}
	switch kType {
	case "string":
		value, err = c.Get(ctx, key).Result()
	case "list":
		value, err = c.LRange(ctx, key, 0, -1).Result()
	case "set":
		var members []string
		members, err = c.SMembers(ctx, key).Result()
		sort.Strings(members)
		value = members
	case "zset":
		var items []redis.Z
		items, err = c.ZRangeWithScores(ctx, key, 0, -1).Result()
		entries := make([]zsetEntry, len(items))
		for i, z := range items {
			entries[i] = zsetEntry{Member: fmt.Sprint(z.Member), Score: z.Score}
		}
		value = entries
	case "hash":
		value, err = c.HGetAll(ctx, key).Result()
	default:
		value = fmt.Sprintf("(viewing %s keys is not supported)", kType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s value: %w", kType, err)
	}

	text, ok := value.(string)
	if !ok {
		b, _ := json.Marshal(value)
		text = string(b)
	}
	return &models.RedisKeyValueResponse{Key: key, Value: text, Type: kType, TTL: ttlSeconds, Database: config.Database}, nil
}

type zsetEntry struct {
	Member string  `json:"member"`
	Score  float64 `json:"score"`
}

// SetKeyValue saves a key in its own type in one transaction, optionally renaming it.
// ttl: nil keeps the current expiry, -1 removes it, >0 sets it in seconds.
func (r *RedisProvider) SetKeyValue(config models.ConnectionConfig, oldKey, newKey, value string, ttl *int64) error {
	c := r.client(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kType, err := c.Type(ctx, oldKey).Result()
	if err != nil {
		return err
	}
	if kType == "none" {
		return fmt.Errorf("key %q does not exist (it may have expired or been deleted)", oldKey)
	}
	rename := newKey != "" && newKey != oldKey
	if rename {
		if n, _ := c.Exists(ctx, newKey).Result(); n > 0 {
			return fmt.Errorf("a key named %q already exists", newKey)
		}
	}
	keepMs, _ := c.PTTL(ctx, oldKey).Result()

	_, err = c.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := writeValue(ctx, p, kType, oldKey, value); err != nil {
			return err
		}
		switch {
		case ttl == nil && keepMs > 0 && kType != "string": // DEL + rewrite dropped the expiry
			p.PExpire(ctx, oldKey, keepMs)
		case ttl != nil && *ttl > 0:
			p.Expire(ctx, oldKey, time.Duration(*ttl)*time.Second)
		case ttl != nil && *ttl == -1:
			p.Persist(ctx, oldKey)
		}
		if rename {
			p.Rename(ctx, oldKey, newKey)
		}
		return nil
	})
	return err
}

// writeValue queues the commands that store value (as returned by GetKeyValue) in the key's type.
// Parsing happens before anything is queued, so invalid input changes nothing.
func writeValue(ctx context.Context, p redis.Pipeliner, kType, key, value string) error {
	bad := func(want string) error { return fmt.Errorf("a %s value must be %s", kType, want) }
	switch kType {
	case "string":
		p.Set(ctx, key, value, redis.KeepTTL)
	case "hash":
		var m map[string]string
		if json.Unmarshal([]byte(value), &m) != nil {
			return bad(`a JSON object of strings, e.g. {"field": "value"}`)
		}
		p.Del(ctx, key)
		if len(m) > 0 {
			p.HSet(ctx, key, m)
		}
	case "list", "set":
		var items []string
		if json.Unmarshal([]byte(value), &items) != nil {
			return bad(`a JSON array of strings, e.g. ["a", "b"]`)
		}
		args := make([]interface{}, len(items))
		for i, it := range items {
			args[i] = it
		}
		p.Del(ctx, key)
		if len(args) > 0 && kType == "list" {
			p.RPush(ctx, key, args...)
		} else if len(args) > 0 {
			p.SAdd(ctx, key, args...)
		}
	case "zset":
		var entries []zsetEntry
		if json.Unmarshal([]byte(value), &entries) != nil {
			return bad(`a JSON array like [{"member": "a", "score": 1}]`)
		}
		zs := make([]redis.Z, len(entries))
		for i, e := range entries {
			zs[i] = redis.Z{Member: e.Member, Score: e.Score}
		}
		p.Del(ctx, key)
		if len(zs) > 0 {
			p.ZAdd(ctx, key, zs...)
		}
	default:
		return fmt.Errorf("editing %s keys is not supported", kType)
	}
	return nil
}

// DeleteKey removes a Redis key.
func (r *RedisProvider) DeleteKey(config models.ConnectionConfig, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client(config).Del(ctx, key).Err()
}
