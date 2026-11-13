package models

// RedisKeyValueRequest is the request to get or update a Redis key's value
type RedisKeyValueRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	Database  string `json:"database"`
	Key       string `json:"key" binding:"required"`
}

// RedisKeyValueUpdateRequest is the request to update a Redis key's value (and optionally rename key)
type RedisKeyValueUpdateRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	Database  string `json:"database"`
	OldKey    string `json:"oldKey" binding:"required"`
	NewKey    string `json:"newKey" binding:"required"`
	Value     string `json:"value"`
	TTL       *int64 `json:"ttl"` // nil keeps the current expiry, -1 removes it, >0 sets it in seconds
}

// RedisKeyDeleteRequest is the request to delete a Redis key
type RedisKeyDeleteRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	Database  string `json:"database"`
	Key       string `json:"key" binding:"required"`
}

// RedisKeyValueResponse is the response containing a Redis key's value details
type RedisKeyValueResponse struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	TTL      int64  `json:"ttl"` // seconds left, -1 means no expiry
	Database string `json:"database"`
}
