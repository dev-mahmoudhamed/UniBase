package models

import "encoding/json"

// MongoUpdateRequest updates the changed top-level fields of one document.
// ID and Changes are relaxed Extended JSON (e.g. {"$oid": "..."}, {"$date": "..."})
// so BSON types survive the round trip through the browser.
type MongoUpdateRequest struct {
	SessionID  string          `json:"sessionId" binding:"required"`
	Database   string          `json:"database" binding:"required"`
	Collection string          `json:"collection" binding:"required"`
	ID         json.RawMessage `json:"id" binding:"required"`
	Changes    json.RawMessage `json:"changes" binding:"required"`
}
