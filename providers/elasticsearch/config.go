// Package elasticsearch implements the autocomplete Provider interface using Elasticsearch.
package elasticsearch

// Config holds Elasticsearch connection parameters and provider-specific options.
type Config struct {
	// URLs is the list of Elasticsearch node URLs.
	URLs []string

	// Index is the name of the Elasticsearch index to use for autocomplete data.
	Index string

	// Username for basic authentication.
	Username string

	// Password for basic authentication.
	Password string

	// CloudID for connecting to Elastic Cloud.
	CloudID string

	// APIKey for API key authentication (alternative to username/password).
	APIKey string

	// RefreshPolicy controls when changes are visible to search.
	// Options: "true" (immediate), "false" (default), "wait_for" (wait for next refresh).
	RefreshPolicy string

	// NumberOfShards configures the number of primary shards for the index.
	// This setting is ONLY used when the index is automatically created by the provider.
	// If the index already exists, this setting is ignored.
	// For production use, it is recommended to pre-create indices with appropriate settings.
	// Default: 1
	NumberOfShards int

	// NumberOfReplicas configures the number of replica shards.
	// This setting is ONLY used when the index is automatically created by the provider.
	// If the index already exists, this setting is ignored.
	// For production use, it is recommended to pre-create indices with appropriate settings.
	// Default: 0
	NumberOfReplicas int
}

// setDefaults applies default values to config fields.
func (c *Config) setDefaults() {
	if c.RefreshPolicy == "" {
		c.RefreshPolicy = "false"
	}
	if c.NumberOfShards == 0 {
		c.NumberOfShards = 1
	}
}
