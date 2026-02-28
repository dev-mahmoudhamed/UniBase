package models

// ProviderMetadata contains all metadata for a database provider
type ProviderMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Icon        string            `json:"icon"`
	Description string            `json:"description"`
	DefaultPort int               `json:"defaultPort"`
	Fields      []ConnectionField `json:"fields"`
	Features    ProviderFeatures  `json:"features"`
}

// ConnectionField defines a configuration field for a provider
type ConnectionField struct {
	Key          string      `json:"key"`
	Label        string      `json:"label"`
	Type         string      `json:"type"` // text, number, password, select
	Placeholder  string      `json:"placeholder,omitempty"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Required     bool        `json:"required"`
	Options      []string    `json:"options,omitempty"` // For select type
	HelpText     string      `json:"helpText,omitempty"`
}

// ProviderFeatures defines what features a provider supports
type ProviderFeatures struct {
	SupportsSSL        bool `json:"supportsSSL"`
	SupportsMultipleDB bool `json:"supportsMultipleDB"`
	SupportsSchemas    bool `json:"supportsSchemas"`
	SupportsProcedures bool `json:"supportsProcedures"`
	SupportsFunctions  bool `json:"supportsFunctions"`
	SupportsViews      bool `json:"supportsViews"`
}

// ProvidersResponse is the API response for provider metadata
type ProvidersResponse struct {
	Providers []ProviderMetadata `json:"providers"`
	Version   string             `json:"version"`
}
