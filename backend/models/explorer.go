package models

// ExplorerNodeRequest is the request to get child nodes
type ExplorerNodeRequest struct {
	SessionID string            `json:"sessionId" binding:"required"`
	NodeType  string            `json:"nodeType" binding:"required"`
	Context   map[string]string `json:"context,omitempty"`
}

// ExplorerNode represents a single node in the object explorer tree
type ExplorerNode struct {
	Key      string                 `json:"key"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Icon     string                 `json:"icon"`
	Leaf     bool                   `json:"leaf"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Children []ExplorerNode         `json:"children,omitempty"`
}

type ExplorerNodeResponse struct {
	Nodes []ExplorerNode `json:"nodes"`
}

type CollectionDataRequest struct {
	SessionID      string            `json:"sessionId" binding:"required"`
	CollectionName string            `json:"collectionName" binding:"required"`
	Context        map[string]string `json:"context,omitempty"`
	Filter         string            `json:"filter,omitempty"`
	Projection     string            `json:"projection,omitempty"`
	Sort           string            `json:"sort,omitempty"`
	Skip           int64             `json:"skip,omitempty"`
	Limit          int64             `json:"limit,omitempty"`
}

// CollectionMetadata holds the stats the database itself reports for a collection.
type CollectionMetadata struct {
	Count          int64 `json:"count"`
	Size           int64 `json:"size"`
	StorageSize    int64 `json:"storageSize"`
	AvgObjSize     int64 `json:"avgObjSize"`
	IndexCount     int   `json:"indexCount"`
	TotalIndexSize int64 `json:"totalIndexSize"`
}

// CollectionMetadataResponse is the API response for collection metadata
type CollectionMetadataResponse struct {
	Collection string             `json:"collection"`
	Metadata   CollectionMetadata `json:"metadata"`
}
