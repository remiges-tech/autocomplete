package elasticsearch

import (
	"fmt"

	"github.com/remiges-tech/autocomplete"
	"github.com/remiges-tech/autocomplete/providers"
)

// init registers the Elasticsearch provider. Import this package with a blank identifier
// to use Elasticsearch as the autocomplete backend:
//
//	import _ "github.com/remiges-tech/autocomplete/providers/elasticsearch"
//
//nolint:gochecknoinits // init() is the idiomatic pattern for provider registration
func init() {
	autocomplete.RegisterProvider("elasticsearch", NewProvider)
}

// NewProvider creates a new Elasticsearch provider from the given configuration.
// It implements ProviderFactory and expects config to be of type elasticsearch.Config.
func NewProvider(config interface{}) (providers.Provider, error) {
	esConfig, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("invalid configuration type for Elasticsearch provider: expected elasticsearch.Config, got %T", config)
	}

	return New(&esConfig)
}
