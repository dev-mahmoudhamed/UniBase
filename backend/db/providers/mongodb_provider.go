package providers

import (
	"bytes"
	"context"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDBProvider struct {
	BaseProvider
}

// buildConnectionURI returns a mongodb:// URI with the user and password
// URL-escaped, so passwords containing @, : or / work.
func (m *MongoDBProvider) buildConnectionURI(config models.ConnectionConfig) string {
	u := url.URL{
		Scheme: "mongodb",
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Path:   "/" + config.Database,
	}
	if config.User != "" {
		u.User = url.UserPassword(config.User, config.Password)
	}
	return u.String()
}

func (m *MongoDBProvider) ping(config models.ConnectionConfig) error {
	client, err := mongoClient(m.buildConnectionURI(config))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

func (m *MongoDBProvider) TestConnection(config models.ConnectionConfig) error {
	return m.ping(config)
}

func (m *MongoDBProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	if err := m.ping(config); err != nil {
		return "", nil, err
	}
	sessionID, err := utils.StoreSession(config)
	return sessionID, nil, err
}

func (m *MongoDBProvider) ExecuteQuery(ctx context.Context, config models.ConnectionConfig, query string) (models.QueryResult, error) {
	uri := m.buildConnectionURI(config)

	client, err := mongoClient(uri)
	if err != nil {
		return models.QueryResult{}, err
	}

	startTime := time.Now()

	// Try to transform shell-like query or parse as Extended JSON
	cmd, err := m.transformShellQuery(query)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to parse MongoDB query: %w", err)
	}

	dbName := config.Database
	if dbName == "" {
		dbName = "admin"
	}

	res := client.Database(dbName).RunCommand(ctx, cmd)
	if res.Err() != nil {
		return models.QueryResult{}, fmt.Errorf("command execution failed: %w", res.Err())
	}
	raw, err := res.Raw()
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to decode result: %w", err)
	}

	executionTime := time.Since(startTime).Milliseconds()
	rows := []map[string]interface{}{}

	// Cursor results (find, aggregate) carry documents in cursor.firstBatch;
	// anything else is a single result document.
	if batch, err := raw.LookupErr("cursor", "firstBatch"); err == nil {
		values, _ := batch.Array().Values()
		for _, v := range values {
			if doc, ok := v.DocumentOK(); ok {
				row, err := toRelaxedDoc(doc)
				if err != nil {
					return models.QueryResult{}, err
				}
				rows = append(rows, row)
			} else {
				rows = append(rows, map[string]interface{}{"value": v.String()})
			}
		}
	} else {
		row, err := toRelaxedDoc(raw)
		if err != nil {
			return models.QueryResult{}, err
		}
		rows = append(rows, row)
	}

	columns := sortedColumns(rows)
	table := make([][]interface{}, len(rows))
	for i, row := range rows {
		table[i] = make([]interface{}, len(columns))
		for j, col := range columns {
			table[i][j] = row[col]
		}
	}

	return models.QueryResult{
		ResultSets:    []models.ResultSet{{Columns: columns, Rows: table, RowCount: len(table)}},
		ExecutionTime: executionTime,
	}, nil
}

func (m *MongoDBProvider) transformShellQuery(query string) (bson.D, error) {
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "db.") {
		var cmd bson.D
		err := bson.UnmarshalExtJSON([]byte(query), false, &cmd)
		return cmd, err
	}

	// Simple regex to extract collection, method, and arguments
	re := regexp.MustCompile(`^db\.([^.]+)\.([^.]+)\(([\s\S]*)\)$`)
	matches := re.FindStringSubmatch(query)
	if len(matches) < 4 {
		return nil, fmt.Errorf("unsupported or invalid MongoDB shell-like query format")
	}

	collName := matches[1]
	method := matches[2]
	args := strings.TrimSpace(matches[3])

	switch method {
	case "find":
		var filter bson.M
		if args == "" {
			filter = bson.M{}
		} else {
			err := bson.UnmarshalExtJSON([]byte(args), false, &filter)
			if err != nil {
				// Handle multiple arguments or invalid JSON
				// For now, assume first arg is filter
				return nil, fmt.Errorf("invalid filter JSON: %w", err)
			}
		}
		return bson.D{
			{Key: "find", Value: collName},
			{Key: "filter", Value: filter},
		}, nil
	case "aggregate":
		var pipeline []bson.M
		err := bson.UnmarshalExtJSON([]byte(args), false, &pipeline)
		if err != nil {
			return nil, fmt.Errorf("invalid pipeline JSON (must be an array): %w", err)
		}
		return bson.D{
			{Key: "aggregate", Value: collName},
			{Key: "pipeline", Value: pipeline},
			{Key: "cursor", Value: bson.D{}},
		}, nil
	case "countDocuments", "count":
		var filter bson.M
		if args != "" {
			_ = bson.UnmarshalExtJSON([]byte(args), false, &filter)
		}
		return bson.D{
			{Key: "count", Value: collName},
			{Key: "query", Value: filter},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

func (m *MongoDBProvider) GetCollectionMetadata(config models.ConnectionConfig, collectionName string, ctx map[string]string) (models.CollectionMetadata, error) {
	uri := m.buildConnectionURI(config)
	mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongoClient(uri)
	if err != nil {
		return models.CollectionMetadata{}, err
	}

	dbName := ctx["database"]
	if dbName == "" {
		dbName = config.Database
	}
	if dbName == "" {
		dbName = "admin"
	}

	db := client.Database(dbName)

	// collStats command
	res := db.RunCommand(mctx, bson.D{{Key: "collStats", Value: collectionName}})
	var stats bson.M
	if err := res.Decode(&stats); err != nil {
		return models.CollectionMetadata{}, fmt.Errorf("failed to get collection stats: %w", err)
	}

	// Only values MongoDB actually reports; nothing estimated or invented.
	return models.CollectionMetadata{
		Count:          toInt64(stats["count"]),
		Size:           toInt64(stats["size"]),
		StorageSize:    toInt64(stats["storageSize"]),
		AvgObjSize:     toInt64(stats["avgObjSize"]),
		IndexCount:     toInt(stats["nindexes"]),
		TotalIndexSize: toInt64(stats["totalIndexSize"]),
	}, nil
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int32:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func (m *MongoDBProvider) GetExplorerChildren(config models.ConnectionConfig, nodeType string, ctx map[string]string) ([]models.ExplorerNode, error) {

	uri := m.buildConnectionURI(config)
	mctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoClient(uri)
	if err != nil {
		return nil, err
	}

	switch nodeType {
	case "root":
		return []models.ExplorerNode{
			{Key: "databases", Label: "Databases", Type: "databases", Icon: "pi pi-database", Leaf: false},
		}, nil
	case "databases":
		dbList, err := client.ListDatabaseNames(mctx, bson.M{})
		if err != nil {
			return nil, fmt.Errorf("failed to list databases: %w", err)
		}
		var nodes []models.ExplorerNode
		for _, dbName := range dbList {
			nodes = append(nodes, models.ExplorerNode{
				Key:   "db:" + dbName,
				Label: dbName,
				Type:  "database",
				Icon:  "pi pi-database",
				Leaf:  false,
				Data:  map[string]interface{}{"database": dbName},
			})
		}
		return nodes, nil
	case "database":
		dbName := ctx["database"]
		collections, err := client.Database(dbName).ListCollectionNames(mctx, bson.M{})
		if err != nil {
			return nil, fmt.Errorf("failed to list collections: %w", err)
		}
		var nodes []models.ExplorerNode
		for _, collName := range collections {
			stats := m.getCollectionStats(mctx, client.Database(dbName), collName)

			nodes = append(nodes, models.ExplorerNode{
				Key:   fmt.Sprintf("coll:%s:%s", dbName, collName),
				Label: collName,
				Type:  "collection",
				Icon:  "pi pi-list",
				Leaf:  true,
				Data: map[string]interface{}{
					"database":   dbName,
					"collection": collName,
					"stats":      stats,
				},
			})
		}
		return nodes, nil
	default:
		return []models.ExplorerNode{}, nil
	}
}

func (m *MongoDBProvider) getCollectionStats(ctx context.Context, db *mongo.Database, collName string) map[string]interface{} {
	stats := make(map[string]interface{})

	res := db.RunCommand(ctx, bson.D{{Key: "collStats", Value: collName}})
	var result bson.M
	if err := res.Decode(&result); err == nil {
		stats["count"] = result["count"]
		stats["size"] = result["size"]
		stats["storageSize"] = result["storageSize"]
		stats["nindexes"] = result["nindexes"]
	}

	cursor, err := db.Collection(collName).Aggregate(ctx, mongo.Pipeline{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
	})
	if err == nil {
		defer cursor.Close(ctx)
		if cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err == nil {
				fields := []string{}
				for k := range doc {
					if len(fields) < 5 {
						fields = append(fields, k)
					}
				}
				stats["fields"] = fields
			}
		}
	}

	return stats
}

func (m *MongoDBProvider) GetCollectionData(config models.ConnectionConfig, collectionName string, ctx map[string]string, filterStr, projectionStr, sortStr string, skip, limit int64) ([]map[string]interface{}, int64, error) {
	uri := m.buildConnectionURI(config)
	mctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
 
	client, err := mongoClient(uri)
	if err != nil {
		return nil, 0, err
	}
 
	// Use database from context, fallback to config
	dbName := ctx["database"]
	if dbName == "" {
		dbName = config.Database
	}
	if dbName == "" {
		dbName = "admin"
	}
 
	collection := client.Database(dbName).Collection(collectionName)
 
	// Prepare options
	findOpts := options.Find()
	if limit > 0 {
		findOpts.SetLimit(limit)
	} else {
		findOpts.SetLimit(100) // default fallback
	}
	if skip > 0 {
		findOpts.SetSkip(skip)
	}
 
	// Parse filter
	var filter bson.M
	if filterStr != "" {
		err := bson.UnmarshalExtJSON([]byte(filterStr), false, &filter)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid filter JSON: %w", err)
		}
	} else {
		filter = bson.M{}
	}
 
	// Parse projection
	if projectionStr != "" {
		var projection bson.M
		if err := bson.UnmarshalExtJSON([]byte(projectionStr), false, &projection); err != nil {
			return nil, 0, fmt.Errorf("invalid projection JSON: %w", err)
		}
		findOpts.SetProjection(projection)
	}

	// Parse sort
	if sortStr != "" {
		var sort bson.D
		if err := bson.UnmarshalExtJSON([]byte(sortStr), false, &sort); err != nil {
			return nil, 0, fmt.Errorf("invalid sort JSON: %w", err)
		}
		findOpts.SetSort(sort)
	}
 
	cursor, err := collection.Find(mctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query collection: %w", err)
	}
	defer cursor.Close(mctx)
 
	var results []map[string]interface{}
	for cursor.Next(mctx) {
		doc, err := toRelaxedDoc(cursor.Current)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, doc)
	}
 
	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}
 
	if results == nil {
		results = []map[string]interface{}{}
	}
 
	// Get total count based on filter
	totalCount, _ := collection.CountDocuments(mctx, filter)
 
	return results, totalCount, nil
}

// UpdateDocument applies $set with the changed top-level fields of one document.
// id and changes are relaxed Extended JSON exactly as the UI received them, so
// ObjectIds, dates, arrays and numeric types are written back unchanged.
func (m *MongoDBProvider) UpdateDocument(config models.ConnectionConfig, database, collection string, id, changes []byte) error {
	var filter bson.D
	if err := bson.UnmarshalExtJSON(append(append([]byte(`{"_id":`), id...), '}'), false, &filter); err != nil {
		return fmt.Errorf("invalid document _id: %w", err)
	}
	var set bson.D
	if err := bson.UnmarshalExtJSON(changes, false, &set); err != nil {
		return fmt.Errorf("invalid changes JSON: %w", err)
	}
	// Never overwrite _id.
	kept := set[:0]
	for _, e := range set {
		if e.Key != "_id" {
			kept = append(kept, e)
		}
	}
	set = kept
	if len(set) == 0 {
		return fmt.Errorf("no fields to update")
	}

	uri := m.buildConnectionURI(config)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongoClient(uri)
	if err != nil {
		return err
	}

	coll := client.Database(database).Collection(collection)

	// Read the stored types of the changed fields. JSON cannot tell 10 from 10.0,
	// so a number typed in the editor keeps the numeric type the field already had.
	projection := bson.D{}
	for _, e := range set {
		projection = append(projection, bson.E{Key: e.Key, Value: 1})
	}
	var current bson.Raw
	err = coll.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&current)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("document not found")
	}
	if err != nil {
		return fmt.Errorf("failed to read document: %w", err)
	}
	for i, e := range set {
		if old, err := current.LookupErr(e.Key); err == nil {
			set[i].Value = keepNumericType(old.Type, e.Value)
		}
	}

	result, err := coll.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found")
	}
	return nil
}

// keepNumericType converts an edited number to the BSON number type the field
// already has (double, int64 or decimal). Non-numeric values pass through.
func keepNumericType(stored bsontype.Type, v interface{}) interface{} {
	var asFloat float64
	var asInt int64
	isInt := false
	switch n := v.(type) {
	case int32:
		asFloat, asInt, isInt = float64(n), int64(n), true
	case int64:
		asFloat, asInt, isInt = float64(n), n, true
	case float64:
		asFloat = n
	default:
		return v
	}
	switch stored {
	case bsontype.Double:
		return asFloat
	case bsontype.Int64:
		if isInt {
			return asInt
		}
	case bsontype.Decimal128:
		s := strconv.FormatFloat(asFloat, 'f', -1, 64)
		if isInt {
			s = strconv.FormatInt(asInt, 10)
		}
		if d, err := primitive.ParseDecimal128(s); err == nil {
			return d
		}
	}
	return v
}

// toRelaxedDoc converts a BSON document to a JSON-ready map in relaxed Extended
// JSON: ObjectIds become {"$oid"}, dates {"$date"}, arrays stay arrays, and number
// literals are kept as written (UseNumber), so 10.0 stays a double.
func toRelaxedDoc(doc interface{}) (map[string]interface{}, error) {
	ext, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to encode document: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(ext))
	dec.UseNumber()
	var out map[string]interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to encode document: %w", err)
	}
	return out, nil
}

// sortedColumns lists every key seen across rows: _id first, then alphabetical.
func sortedColumns(rows []map[string]interface{}) []string {
	seen := map[string]bool{}
	for _, row := range rows {
		for k := range row {
			seen[k] = true
		}
	}
	columns := []string{}
	if seen["_id"] {
		columns = append(columns, "_id")
		delete(seen, "_id")
	}
	rest := make([]string, 0, len(seen))
	for k := range seen {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	return append(columns, rest...)
}
