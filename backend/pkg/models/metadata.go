package models

// ProxyMetadata represents the metadata for a proxy that's not stored in Caddy config.
type ProxyMetadata struct {
	ID                        string            `json:"id"`
	HealthCheckEnabled        bool              `json:"health_check_enabled"`
	HealthCheckInterval       string            `json:"health_check_interval"`
	HealthCheckPath           string            `json:"health_check_path"`
	HealthCheckExpectedStatus int               `json:"health_check_expected_status"`
	ChallengeType             string            `json:"challenge_type"`
	DNSProvider               string            `json:"dns_provider"`
	DNSCredentials            map[string]string `json:"dns_credentials"`
	CustomHeaders             map[string]string `json:"custom_headers"`
	BasicAuth                 *BasicAuth        `json:"basic_auth"`
	CreatedAt                 string            `json:"created_at"`
	UpdatedAt                 string            `json:"updated_at"`
}

// MetadataStore manages proxy metadata storage.
type MetadataStore struct {
	Data map[string]ProxyMetadata `json:"proxies"`
}

// NewMetadataStore creates a new metadata store
func NewMetadataStore() *MetadataStore {
	return &MetadataStore{
		Data: make(map[string]ProxyMetadata),
	}
}

// Set stores metadata for a proxy
func (ms *MetadataStore) Set(proxy Proxy) {
	metadata := ProxyMetadata{
		ID:                        proxy.ID,
		HealthCheckEnabled:        proxy.HealthCheckEnabled,
		HealthCheckInterval:       proxy.HealthCheckInterval,
		HealthCheckPath:           proxy.HealthCheckPath,
		HealthCheckExpectedStatus: proxy.HealthCheckExpectedStatus,
		ChallengeType:             proxy.ChallengeType,
		DNSProvider:               proxy.DNSProvider,
		DNSCredentials:            proxy.DNSCredentials,
		CustomHeaders:             proxy.CustomHeaders,
		BasicAuth:                 proxy.BasicAuth,
		CreatedAt:                 proxy.CreatedAt,
		UpdatedAt:                 proxy.UpdatedAt,
	}
	ms.Data[proxy.ID] = metadata
}

// Get retrieves metadata for a proxy
func (ms *MetadataStore) Get(proxyID string) (ProxyMetadata, bool) {
	metadata, exists := ms.Data[proxyID]

	return metadata, exists
}

// Delete removes metadata for a proxy
func (ms *MetadataStore) Delete(proxyID string) {
	delete(ms.Data, proxyID)
}

// ApplyToProxy applies stored metadata to a proxy object
func (ms *MetadataStore) ApplyToProxy(proxy *Proxy) {
	if metadata, exists := ms.Data[proxy.ID]; exists {
		proxy.HealthCheckEnabled = metadata.HealthCheckEnabled
		proxy.HealthCheckInterval = metadata.HealthCheckInterval
		proxy.HealthCheckPath = metadata.HealthCheckPath
		proxy.HealthCheckExpectedStatus = metadata.HealthCheckExpectedStatus
		proxy.ChallengeType = metadata.ChallengeType
		proxy.DNSProvider = metadata.DNSProvider
		proxy.DNSCredentials = metadata.DNSCredentials
		proxy.CustomHeaders = metadata.CustomHeaders
		proxy.BasicAuth = metadata.BasicAuth
		proxy.CreatedAt = metadata.CreatedAt
		proxy.UpdatedAt = metadata.UpdatedAt
	}
}
