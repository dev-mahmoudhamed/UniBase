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
	return nil
}

type QueryResult struct {
	Columns       []string                 `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	RowCount      int                      `json:"rowCount"`
	ExecutionTime int64                    `json:"executionTime"`
}
