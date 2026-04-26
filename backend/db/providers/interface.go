package providers

import "db-server/models"

type DatabaseProvider interface {
	TestConnection(config models.ConnectionConfig) error
	InitializeConnection(config models.ConnectionConfig) (string, interface{}, error)
	ExecuteQuery(config models.ConnectionConfig, query string) (models.QueryResult, error)
	GetExplorerChildren(config models.ConnectionConfig, nodeType string, context map[string]string) ([]models.ExplorerNode, error)
}

type CollectionDataProvider interface {
	GetCollectionData(config models.ConnectionConfig, collectionName string, context map[string]string) ([]map[string]interface{}, error)
}
