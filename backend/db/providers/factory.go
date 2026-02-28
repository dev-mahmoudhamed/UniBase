package providers

import (
	"errors"
	"strings"
)

// GetProvider returns the appropriate database provider based on the provider string
func GetProvider(provider string) (DatabaseProvider, error) {
	switch strings.ToLower(provider) {
	case "postgresql", "postgres":
		return &PostgresProvider{}, nil
	case "mssql", "sqlserver":
		return &SQLServerProvider{}, nil
	case "mysql":
		return &MySQLProvider{}, nil
	case "mongodb":
		return &MongoDBProvider{}, nil
	case "redis":
		return &RedisProvider{}, nil
	default:
		return nil, errors.New("unsupported database provider")
	}
}
