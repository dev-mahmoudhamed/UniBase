package registry

import (
	"db-server/models"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const (
	defaultConfigPath = "./config/providers.json"
)

var (
	instance     *ProviderRegistry
	once         sync.Once
	registryLock sync.RWMutex
)

type ProviderRegistry struct {
	providers  map[string]models.ProviderMetadata
	version    string
	configPath string
}

func GetRegistry() *ProviderRegistry {
	once.Do(func() {
		instance = &ProviderRegistry{
			providers:  make(map[string]models.ProviderMetadata),
			configPath: defaultConfigPath,
		}
	})
	return instance
}

func (r *ProviderRegistry) LoadFromFile(path string) error {
	if path != "" {
		r.configPath = path
	}

	file, err := os.ReadFile(r.configPath)
	if err != nil {
		return fmt.Errorf("failed to read providers config: %w", err)
	}

	var config struct {
		Version   string                    `json:"version"`
		Providers []models.ProviderMetadata `json:"providers"`
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("failed to parse providers config: %w", err)
	}

	registryLock.Lock()
	defer registryLock.Unlock()

	r.version = config.Version
	r.providers = make(map[string]models.ProviderMetadata)

	for _, provider := range config.Providers {
		r.providers[provider.ID] = provider
	}

	return nil
}

func (r *ProviderRegistry) GetProvider(id string) (models.ProviderMetadata, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()

	provider, exists := r.providers[id]
	if !exists {
		return models.ProviderMetadata{}, fmt.Errorf("provider '%s' not found", id)
	}

	return provider, nil
}

func (r *ProviderRegistry) GetAllProviders() []models.ProviderMetadata {
	registryLock.RLock()
	defer registryLock.RUnlock()

	providers := make([]models.ProviderMetadata, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}

func (r *ProviderRegistry) GetProvidersResponse() models.ProvidersResponse {
	return models.ProvidersResponse{
		Providers: r.GetAllProviders(),
		Version:   r.version,
	}
}

func (r *ProviderRegistry) IsProviderSupported(id string) bool {
	registryLock.RLock()
	defer registryLock.RUnlock()

	_, exists := r.providers[id]
	return exists
}

func (r *ProviderRegistry) GetSupportedProviderIDs() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}

	return ids
}

func (r *ProviderRegistry) ValidateConnection(providerID string, config map[string]interface{}) []string {
	provider, err := r.GetProvider(providerID)
	if err != nil {
		return []string{err.Error()}
	}

	var errors []string

	for _, field := range provider.Fields {
		if !field.Required {
			continue
		}

		value, exists := config[field.Key]
		if !exists || value == nil || value == "" {
			errors = append(errors, fmt.Sprintf("%s is required", field.Label))
		}
	}

	return errors
}

func (r *ProviderRegistry) GetProviderIcon(id string) string {
	provider, err := r.GetProvider(id)
	if err != nil {
		return ""
	}
	return provider.Icon
}

func (r *ProviderRegistry) GetDefaultPort(id string) int {
	provider, err := r.GetProvider(id)
	if err != nil {
		return 0
	}
	return provider.DefaultPort
}
