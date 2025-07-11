package redis

import (
	"fmt"

	"github.com/remiges/cvl-kra/autocomplete"
	"github.com/remiges/cvl-kra/autocomplete/providers"
)

// init registers the Redis provider. Import this package with a blank identifier
// to use Redis as the autocomplete backend:
//
//	import _ "github.com/remiges/cvl-kra/autocomplete/providers/redis"
//
//nolint:gochecknoinits // init() is the idiomatic pattern for provider registration
func init() {
	autocomplete.RegisterProvider("redis", NewProvider)
}

// NewProvider creates a new Redis provider from the given configuration.
// It implements ProviderFactory and expects config to be of type redis.Config.
func NewProvider(config interface{}) (providers.Provider, error) {
	redisConfig, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("invalid configuration type for Redis provider: expected redis.Config, got %T", config)
	}

	return New(redisConfig)
}
