package providers

import (
	"context"
	"db-server/db/utils"
	"db-server/models"
	"encoding/json"
	"fmt"
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

	// Try to parse query as Extended JSON for MongoDB commands
	var cmd bson.D
	err = bson.UnmarshalExtJSON([]byte(query), true, &cmd)
	if err != nil {
		return models.QueryResult{}, fmt.Errorf("MongoDB queries must be valid Extended JSON commands: %w", err)
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

	rows := []map[string]interface{}{result}

	return models.QueryResult{
		Columns:       []string{"result"},
		Rows:          rows,
		RowCount:      len(rows),
		ExecutionTime: executionTime,
	}, nil
}

// GetExplorerChildren returns child nodes for the MongoDB object explorer tree
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
				Data:  map[string]string{"database": dbName},
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
			nodes = append(nodes, models.ExplorerNode{
				Key:   fmt.Sprintf("coll:%s:%s", dbName, collName),
				Label: collName,
				Type:  "collection",
				Icon:  "pi pi-list",
				Leaf:  true,
				Data:  map[string]string{"database": dbName, "collection": collName},
			})
		}
		return nodes, nil
	default:
		return []models.ExplorerNode{}, nil
	}
}

// GetCollectionData fetches documents from a MongoDB collection
func (m *MongoDBProvider) GetCollectionData(config models.ConnectionConfig, collectionName string, ctx map[string]string) ([]map[string]interface{}, error) {
	uri := m.buildConnectionURI(config)
	mctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(mctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
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

	// Fetch up to 100 documents
	findOpts := options.Find().SetLimit(100)
	cursor, err := collection.Find(mctx, bson.M{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}
	defer cursor.Close(mctx)

	var results []map[string]interface{}
	for cursor.Next(mctx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}

		// Convert ObjectID and other BSON types to JSON-friendly representations
		sanitized := sanitizeBSONDocument(doc)
		results = append(results, sanitized)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if results == nil {
		results = []map[string]interface{}{}
	}

	return results, nil
}

// UpdateDocument updates specific fields on a document identified by objectId.
// Existing fields are updated; new fields are created. The _id field is never modified.
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

// sanitizeBSONDocument converts BSON-specific types to JSON-friendly representations
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
