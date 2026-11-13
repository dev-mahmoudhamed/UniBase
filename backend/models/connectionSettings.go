package models

import (
	"encoding/json"
	"fmt"
)

type ConnectionConfig struct {
	Provider string `json:"provider"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode,omitempty"`
}

func (c *ConnectionConfig) UnmarshalJSON(data []byte) error {
	type Alias ConnectionConfig
	aux := &struct {
		Database interface{} `json:"database"`
		Port     interface{} `json:"port"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Database != nil {
		switch v := aux.Database.(type) {
		case string:
			c.Database = v
		case float64:
			c.Database = fmt.Sprintf("%.0f", v)
		default:
			c.Database = fmt.Sprintf("%v", v)
		}
	}

	if aux.Port != nil {
		switch v := aux.Port.(type) {
		case float64:
			c.Port = int(v)
		case string:
			var p int
			if _, err := fmt.Sscanf(v, "%d", &p); err == nil {
				c.Port = p
			}
		}
	}

	return nil
}

// ResultSet is one table of results. Rows are positional, so duplicate column
// names (a.id, b.id in a join) don't overwrite each other.
type ResultSet struct {
	Columns   []string        `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	RowCount  int             `json:"rowCount"`
	Truncated bool            `json:"truncated"`
}

// QueryResult holds every result set a script returned, in order.
type QueryResult struct {
	ResultSets    []ResultSet `json:"resultSets"`
	ExecutionTime int64       `json:"executionTime"`
}
