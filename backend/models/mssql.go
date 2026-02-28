package models

type ExplorerTree struct {
	SystemDatabases []Database `json:"System Databases"`
	UserDatabases   []Database `json:"My OWN Databases"`
}

type Database struct {
	Name             string       `json:"DatabaseName"`
	Tables           []Table      `json:"Tables"`
	Views            []SimpleNode `json:"Views"`
	StoredProcedures []SimpleNode `json:"StoredProcedures,omitempty"`
	DatabaseTriggers []SimpleNode `json:"DatabaseTriggers,omitempty"`
	IsSystem         bool         `json:"-"`
}

type Table struct {
	ObjectID int      `json:"-"`
	Name     string   `json:"TableName"`
	Columns  []Column `json:"Columns"`
	Keys     []Key    `json:"Keys"`
	Indexes  []Index  `json:"Indexes"`
}

type Column struct {
	ObjectID int    `json:"-"`
	Name     string `json:"ColumnName"`
	DataType string `json:"DataType"`
}

type Key struct {
	ObjectID int    `json:"-"`
	Name     string `json:"KeyName"`
	TypeDesc string `json:"KeyType"`
}

type Index struct {
	ObjectID int    `json:"-"`
	Name     string `json:"IndexName"`
	TypeDesc string `json:"IndexType"`
}

type SimpleNode struct {
	ObjectID int    `json:"-"`
	Name     string `json:"Name"`
}
