package providers

import "db-server/models"

type DatabaseProvider interface {
	TestConnection(config models.ConnectionConfig) error
	InitializeConnection(config models.ConnectionConfig) (string, interface{}, error)
	ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error)
	GetExplorerChildren(config models.ConnectionConfig, nodeType string, context map[string]string) ([]models.ExplorerNode, error)
	GetCollectionMetadata(config models.ConnectionConfig, collectionName string, context map[string]string) (models.CollectionMetadata, error)
}


type CollectionDataProvider interface {
	GetCollectionData(config models.ConnectionConfig, collectionName string, context map[string]string, filter, projection, sort string, skip, limit int64) ([]map[string]interface{}, int64, error)
}
