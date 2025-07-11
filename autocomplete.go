// Package autocomplete provides an autocomplete implementation
// with support for multiple storage backends and matching strategies.
//
// The package separates autocomplete logic from storage concerns through a provider interface,
// allowing different backends (Redis, PostgreSQL, etc.) to be used interchangeably.
// Providers self-register during package initialization.
//
// Basic usage:
//
//	import (
//		"github.com/remiges/cvl-kra/autocomplete"
//		"github.com/remiges/cvl-kra/autocomplete/providers/redis"
//	)
//
//	// Configure provider
//	redisConfig := redis.Config{
//		Addr: "localhost:6379",
//		DB:   0,
//	}
//
//	// Create autocomplete instance
//	config := autocomplete.NewConfig(redisConfig)
//	config.Options.MatchStrategy = autocomplete.MatchSubstring  // Optional: customize behavior
//	config.Options.CaseSensitive = true  // Optional: enable case-sensitive matching
//	ac, err := autocomplete.New("redis", config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer ac.Close()
//
//	// Index entries - text parameter is what's searched, display is what's shown
//	ac.Index(ctx, "1", "Mumbai", "Mumbai, Maharashtra")
//	ac.Index(ctx, "2", "Bangalore", "Bangalore, Karnataka")
//
//	// Query entries
//	results, err := ac.Query(ctx, "ban", 10)
package autocomplete

import (
	"context"
	"fmt"
	"strings"

	"github.com/remiges/cvl-kra/autocomplete/providers"
)

// Result represents a single autocomplete result returned from a query.
type Result struct {
	// ID is the unique identifier as provided during indexing.
	ID string `json:"id"`

	// Display is the text shown to users in search results.
	Display string `json:"display"`

	// Score indicates relevance (higher scores rank first).
	Score float64 `json:"score"`
}

// AutoComplete defines the interface for autocomplete functionality.
// All methods are safe for concurrent use.
type AutoComplete interface {
	// Index adds or updates a text entry for autocomplete.
	// If an entry with the given ID already exists, it will be replaced.
	// The text parameter is what gets indexed and matched against queries,
	// while display is what appears in search results.
	// Returns ErrEmptyID, ErrEmptyText, or ErrEmptyDisplay for empty parameters.
	Index(ctx context.Context, id string, text string, display string) error

	// Query searches for entries matching the given query string.
	// Results are sorted by score (highest first). The matching behavior
	// depends on the configured MatchStrategy. If limit is 0 or negative,
	// DefaultLimit is used.
	// Returns ErrQueryTooShort if query is too short, ErrLimitExceeded if
	// limit exceeds MaxLimit, or an empty slice if no matches are found.
	Query(ctx context.Context, query string, limit int) ([]Result, error)

	// Delete removes an entry from the autocomplete index.
	// Deleting a non-existent entry returns nil (idempotent).
	// Returns ErrEmptyID if id is empty.
	Delete(ctx context.Context, id string) error

	// DeleteAll removes all entries from the autocomplete index.
	// This operation is irreversible and only affects entries in the configured namespace.
	DeleteAll(ctx context.Context) error

	// Close closes the autocomplete provider and releases resources.
	// It is safe to call multiple times. After Close, other methods will fail.
	Close() error
}

// autocompleteImpl is the default implementation of AutoComplete.
type autocompleteImpl struct {
	provider providers.Provider
	config   Config
}

// Index adds or updates a text entry for autocomplete.
// See AutoComplete.Index for details.
func (a *autocompleteImpl) Index(ctx context.Context, id, text, display string) error {
	if id == "" {
		return ErrEmptyID
	}
	if text == "" {
		return ErrEmptyText
	}
	if display == "" {
		return ErrEmptyDisplay
	}

	options := providers.IndexOptions{
		Score:         1.0,
		MatchStrategy: providers.MatchStrategy(a.config.Options.MatchStrategy),
		NGramSize:     a.config.Options.NGramSize,
		CaseSensitive: a.config.Options.CaseSensitive,
	}

	return a.provider.Index(ctx, a.config.Options.Namespace, id, text, display, options)
}

// Query searches for entries matching the given query.
// See AutoComplete.Query for details.
func (a *autocompleteImpl) Query(ctx context.Context, query string, limit int) ([]Result, error) {
	if len(query) < a.config.Options.MinPrefixLength {
		return nil, ErrQueryTooShort
	}

	if limit <= 0 {
		limit = a.config.Options.DefaultLimit
	}
	if limit > a.config.Options.MaxLimit {
		return nil, ErrLimitExceeded
	}

	options := providers.QueryOptions{
		MaxResults:    limit,
		CaseSensitive: a.config.Options.CaseSensitive,
		MatchStrategy: providers.MatchStrategy(a.config.Options.MatchStrategy),
		NGramSize:     a.config.Options.NGramSize,
	}

	providerResults, err := a.provider.Query(ctx, a.config.Options.Namespace, query, options)
	if err != nil {
		return nil, err
	}

	results := make([]Result, len(providerResults))
	for i, pr := range providerResults {
		results[i] = Result{
			ID:      pr.ID,
			Display: pr.Display,
			Score:   pr.Score,
		}
	}

	return results, nil
}

// Delete removes an entry from the autocomplete index.
// See AutoComplete.Delete for details.
func (a *autocompleteImpl) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrEmptyID
	}

	return a.provider.Delete(ctx, a.config.Options.Namespace, id)
}

// DeleteAll removes all entries from the autocomplete index.
// See AutoComplete.DeleteAll for details.
func (a *autocompleteImpl) DeleteAll(ctx context.Context) error {
	return a.provider.DeleteAll(ctx, a.config.Options.Namespace)
}

// Close closes the autocomplete provider and releases resources.
// See AutoComplete.Close for details.
func (a *autocompleteImpl) Close() error {
	return a.provider.Close()
}

// New creates a new AutoComplete instance with the specified provider.
// The providerType must be registered (case-insensitive). Config contains
// both provider-specific settings and common options.
// Returns ErrProviderNotFound if the provider is not registered.
//
// Example:
//
//	import _ "github.com/remiges/cvl-kra/autocomplete/providers/redis"
//
//	config := autocomplete.NewConfig(redis.Config{Addr: "localhost:6379"})
//	ac, err := autocomplete.New("redis", config)
//
//nolint:gocritic // hugeParam: Config is 80 bytes but New() is only called once at startup, making the copy negligible
func New(providerType string, config Config) (AutoComplete, error) {
	factory, exists := providerFactories[providerType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerType)
	}

	provider, err := factory(config.ProviderConfig)
	if err != nil {
		return nil, err
	}

	return &autocompleteImpl{
		provider: provider,
		config:   config,
	}, nil
}

// ProviderFactory creates a Provider instance from a configuration.
// The factory must type-assert the config parameter to its expected type.
type ProviderFactory func(config interface{}) (providers.Provider, error)

// providerFactories holds the registered provider factories.
var providerFactories = make(map[string]ProviderFactory)

// RegisterProvider registers a new autocomplete provider factory.
// Typically called from a provider's init() function. The name is
// case-insensitive. Registering with an existing name overwrites it.
//
// Example:
//
//	package myprovider
//
//	func init() {
//	    autocomplete.RegisterProvider("myprovider", NewProvider)
//	}
//
//	func NewProvider(config interface{}) (providers.Provider, error) {
//	    cfg, ok := config.(Config)
//	    if !ok {
//	        return nil, errors.New("invalid config type")
//	    }
//	    return &Provider{config: cfg}, nil
//	}
//
// Example - Conditional registration:
//
//	func init() {
//	    if isSupported() {
//	        autocomplete.RegisterProvider("myprovider", NewProvider)
//	    }
//	}
//
// Thread safety:
//   - Safe to call during init() (single-threaded)
//   - Not safe to call after init() (no mutex protection)
//   - In practice, only called during init() so this is not an issue
func RegisterProvider(name string, factory ProviderFactory) {
	providerFactories[strings.ToLower(name)] = factory
}
