package models

// MongoUpdateRequest is the request to update a MongoDB document's fields
type MongoUpdateRequest struct {
	SessionID  string                 `json:"sessionId" binding:"required"`
	Database   string                 `json:"database" binding:"required"`
	Collection string                 `json:"collection" binding:"required"`
	ObjectID   string                 `json:"objectId" binding:"required"`
	Properties map[string]interface{} `json:"properties" binding:"required"`
}
