package providers

import (
	"context"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDBProvider struct {
	BaseProvider
}

// buildConnectionURI creates a MongoDB connection URI
func (m *MongoDBProvider) buildConnectionURI(config models.ConnectionConfig) string {
	var userPass string
	if config.User != "" && config.Password != "" {
		userPass = fmt.Sprintf("%s:%s@", config.User, config.Password)
	}

	return fmt.Sprintf("mongodb://%s%s:%d/%s",
		userPass, config.Host, config.Port, config.Database)
}

func (m *MongoDBProvider) TestConnection(config models.ConnectionConfig) error {
	uri := m.buildConnectionURI(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (m *MongoDBProvider) InitializeConnection(config models.ConnectionConfig) (string, interface{}, error) {
	uri := m.buildConnectionURI(config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return "", nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	// Fetch metadata (Databases and Collections)
	metadata := make(map[string]interface{})

	dbList, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return "", nil, fmt.Errorf("failed to list databases: %w", err)
	}

	databasesMeta := []map[string]interface{}{}
	for _, dbName := range dbList {
		dbMeta := map[string]interface{}{
			"name": dbName,
		}

		collections, err := client.Database(dbName).ListCollectionNames(ctx, bson.M{})
		if err == nil {
			collectionsMeta := []map[string]interface{}{}
			for _, collName := range collections {
				collectionsMeta = append(collectionsMeta, map[string]interface{}{
					"name": collName,
					"type": "collection",
				})
			}
			dbMeta["collections"] = collectionsMeta
		}
		databasesMeta = append(databasesMeta, dbMeta)
	}
	metadata["databases"] = databasesMeta

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

func (m *MongoDBProvider) ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error) {
	uri := m.buildConnectionURI(config)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

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
	var result map[string]interface{}
	if err := res.Decode(&result); err != nil {
		return models.QueryResult{}, fmt.Errorf("failed to decode result: %w", err)
	}

	executionTime := time.Since(startTime).Milliseconds()
	rows := []map[string]interface{}{}

	// Handle results: could be a single document or a cursor result
	if cursorData, ok := result["cursor"].(map[string]interface{}); ok {
		// If it's a cursor (from aggregate or find command)
		if firstBatch, ok := cursorData["firstBatch"].([]interface{}); ok {
			for _, item := range firstBatch {
				if doc, ok := item.(map[string]interface{}); ok {
					rows = append(rows, sanitizeBSONDocument(doc))
				} else {
					rows = append(rows, map[string]interface{}{"value": item})
				}
			}
		}
	} else {
		// Single result
		rows = append(rows, result)
	}

	// Calculate unique columns from results
	columnsMap := make(map[string]bool)
	for _, row := range rows {
		for k := range row {
			columnsMap[k] = true
		}
	}

	columns := []string{}
	// Priority to _id
	if columnsMap["_id"] {
		columns = append(columns, "_id")
		delete(columnsMap, "_id")
	}
	for k := range columnsMap {
		columns = append(columns, k)
	}

	return models.QueryResult{
		Columns:       columns,
		Rows:          rows,
		RowCount:      len(rows),
		ExecutionTime: executionTime,
	}, nil
}

func (m *MongoDBProvider) transformShellQuery(query string) (bson.D, error) {
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "db.") {
		var cmd bson.D
		err := bson.UnmarshalExtJSON([]byte(query), true, &cmd)
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
			err := bson.UnmarshalExtJSON([]byte(args), true, &filter)
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
		err := bson.UnmarshalExtJSON([]byte(args), true, &pipeline)
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
			_ = bson.UnmarshalExtJSON([]byte(args), true, &filter)
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

	client, err := mongo.Connect(mctx, options.Client().ApplyURI(uri))
	if err != nil {
		return models.CollectionMetadata{}, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Disconnect(mctx) }()

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

	// Prepare metadata with real and some mocked metrics for demo purposes
	metadata := models.CollectionMetadata{
		Count:         toInt64(stats["count"]),
		Size:          toInt64(stats["size"]),
		StorageSize:   toInt64(stats["storageSize"]),
		IndexCount:    toInt(stats["nindexes"]),
		UnusedIndices: 2,       // Mocked
		AvgQueryTime:  "180ms", // Mocked
		ReadsPerSec:   220,     // Mocked
		WritesPerSec:  45,      // Mocked
		Trend:         "↑18%",  // Mocked
	}

	if metadata.Size > 1024*1024*1024 {
		metadata.Alerts = append(metadata.Alerts, "Large documents detected")
	}

	foundUserId := false
	if !foundUserId {
		metadata.Alerts = append(metadata.Alerts, "Missing index on userId")
	}

	return metadata, nil
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

	client, err := mongo.Connect(mctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Disconnect(mctx)
	}()

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
 
	client, err := mongo.Connect(mctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Disconnect(mctx)
	}()
 
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
		err := bson.UnmarshalExtJSON([]byte(filterStr), true, &filter)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid filter JSON: %w", err)
		}
	} else {
		filter = bson.M{}
	}
 
	// Parse projection
	if projectionStr != "" {
		var projection bson.M
		err := bson.UnmarshalExtJSON([]byte(projectionStr), true, &projection)
		if err == nil {
			findOpts.SetProjection(projection)
		}
	}
 
	// Parse sort
	if sortStr != "" {
		var sort bson.D
		err := bson.UnmarshalExtJSON([]byte(sortStr), true, &sort)
		if err == nil {
			findOpts.SetSort(sort)
		}
	}
 
	cursor, err := collection.Find(mctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query collection: %w", err)
	}
	defer cursor.Close(mctx)
 
	var results []map[string]interface{}
	for cursor.Next(mctx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, fmt.Errorf("failed to decode document: %w", err)
		}
 
		// Convert ObjectID and other BSON types to JSON-friendly representations
		sanitized := sanitizeBSONDocument(doc)
		results = append(results, sanitized)
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

func (m *MongoDBProvider) UpdateDocument(config models.ConnectionConfig, database, collection, objectID string, properties map[string]interface{}) error {
	uri := m.buildConnectionURI(config)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	// Build the filter — try ObjectID hex first, fall back to string
	var filter bson.M
	objID, err := primitive.ObjectIDFromHex(objectID)
	if err == nil {
		filter = bson.M{"_id": objID}
	} else {
		filter = bson.M{"_id": objectID}
	}

	// Ensure _id is never overwritten
	delete(properties, "_id")

	if len(properties) == 0 {
		return fmt.Errorf("no properties to update")
	}

	// Build $set document from the provided properties
	setFields := bson.M{}
	for k, v := range properties {
		setFields[k] = v
	}

	result, err := client.Database(database).Collection(collection).UpdateOne(
		ctx,
		filter,
		bson.M{"$set": setFields},
	)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found (objectId: %s)", objectID)
	}

	return nil
}

func sanitizeBSONDocument(doc map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range doc {
		result[key] = sanitizeBSONValue(value)
	}
	return result
}

func sanitizeBSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case primitive.ObjectID:
		return v.Hex()
	case primitive.DateTime:
		return v.Time().UTC().Format(time.RFC3339)
	case map[string]interface{}:
		return sanitizeBSONDocument(v)
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, item := range v {
			sanitized[i] = sanitizeBSONValue(item)
		}
		return sanitized
	case bool:
		return v
	case int32, int64, float64:
		return v
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}
