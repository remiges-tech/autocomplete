// Package providers defines the interface that all autocomplete storage providers must implement.
package providers

import (
	"context"
)

// MatchStrategy defines how search terms are matched against indexed text.
// This mirrors autocomplete.MatchStrategy to avoid circular dependencies.
type MatchStrategy int

const (
	// MatchPrefix matches from the beginning of words only.
	MatchPrefix MatchStrategy = iota

	// MatchNGram uses fixed-length n-grams for matching.
	MatchNGram

	// MatchNOrMoreGram uses variable-length n-grams (n or longer).
	MatchNOrMoreGram

	// MatchSubstring matches any substring within the text.
	MatchSubstring
)

// IndexOptions contains options for indexing operations.
type IndexOptions struct {
	// Score is the default relevance score for this entry.
	// Higher scores indicate more relevant results.
	Score float64

	// MatchStrategy determines how the text should be tokenized.
	MatchStrategy MatchStrategy

	// NGramSize is the n-gram size for MatchNGram and MatchNOrMoreGram strategies.
	// Ignored for MatchPrefix and MatchSubstring.
	NGramSize int

	// CaseSensitive determines if the indexed text preserves case.
	CaseSensitive bool
}

// QueryOptions contains options for query operations.
type QueryOptions struct {
	// MinScore filters out results with scores below this threshold.
	MinScore float64

	// MaxResults limits the number of results returned.
	// Results must be sorted by score (highest first).
	MaxResults int

	// CaseSensitive controls whether searches are case-sensitive.
	CaseSensitive bool

	// IncludeScores determines if result scores should be populated.
	IncludeScores bool

	// FilterMetadata allows provider-specific filtering (currently unused).
	FilterMetadata map[string]interface{}

	// MatchStrategy must match the strategy used during indexing.
	MatchStrategy MatchStrategy

	// NGramSize must match the size used during indexing.
	NGramSize int
}

// Provider defines the interface that all autocomplete providers must implement.
// All methods must be safe for concurrent use. The 'key' parameter acts as
// a namespace to allow multiple datasets to coexist.
type Provider interface {
	// Index adds or updates an entry in the autocomplete index.
	// If an entry with the given key+id exists, it will be replaced.
	// The text is tokenized according to options.MatchStrategy.
	Index(ctx context.Context, key, id, text, display string, options IndexOptions) error

	// Query searches for entries matching the given query.
	// Results must be sorted by score (highest first) and limited to MaxResults.
	// Returns an empty slice (not nil) if no matches are found.
	Query(ctx context.Context, key, query string, options QueryOptions) ([]ProviderResult, error)

	// Delete removes an entry from the index.
	// Deleting a non-existent entry succeeds without error (idempotent).
	Delete(ctx context.Context, key, id string) error

	// DeleteAll removes all entries for a given key namespace.
	// This operation cannot be undone.
	DeleteAll(ctx context.Context, key string) error

	// Close closes the provider connection and releases resources.
	// It is safe to call multiple times. After Close, other methods will fail.
	Close() error
}

// ProviderResult represents a single search result from a provider.
type ProviderResult struct {
	// ID is the unique identifier provided during indexing.
	ID string

	// Display is the text to show to users.
	Display string

	// Score indicates relevance (higher is better).
	Score float64
}
