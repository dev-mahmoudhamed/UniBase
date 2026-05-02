package models

// ExplorerNodeRequest is the request to get child nodes
type ExplorerNodeRequest struct {
	SessionID string            `json:"sessionId" binding:"required"`
	NodeType  string            `json:"nodeType" binding:"required"`
	Context   map[string]string `json:"context,omitempty"`
}

// ExplorerNode represents a single node in the object explorer tree
type ExplorerNode struct {
	Key      string            `json:"key"`
	Label    string            `json:"label"`
	Type     string            `json:"type"`
	Icon     string            `json:"icon"`
	Leaf     bool              `json:"leaf"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Children []ExplorerNode    `json:"children,omitempty"`
}

// ExplorerNodeResponse is the API response for explorer children
type ExplorerNodeResponse struct {
	Nodes []ExplorerNode `json:"nodes"`
}

// CollectionDataRequest is the request to fetch collection/document data
type CollectionDataRequest struct {
	SessionID      string            `json:"sessionId" binding:"required"`
	CollectionName string            `json:"collectionName" binding:"required"`
	Context        map[string]string `json:"context,omitempty"`
}
