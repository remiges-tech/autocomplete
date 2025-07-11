package autocomplete

// defaultLimit is the default number of results to return.
const defaultLimit = 10

// defaultMaxLimit is the maximum allowed results.
const defaultMaxLimit = 100

// defaultNGramSize is the default n-gram size (trigrams).
const defaultNGramSize = 3

// MatchStrategy defines how search terms are matched against indexed text.
type MatchStrategy int

const (
	// MatchPrefix matches from the beginning of words only.
	// Example: "mum" matches "Mumbai" but not "Jammu".
	MatchPrefix MatchStrategy = iota
	// MatchNGram uses fixed-length n-grams (overlapping substrings) for matching.
	// Example: With n=3, "Bangalore" -> ["Ban", "ang", "nga", "gal", "alo", "lor", "ore"].
	MatchNGram
	// MatchNOrMoreGram uses variable-length n-grams (n or longer).
	// Example: With n=3, "test" -> ["tes", "test", "est"].
	MatchNOrMoreGram
	// MatchSubstring matches any substring within the text.
	// Example: "test" -> ["t", "te", "tes", "test", "e", "es", "est", "s", "st", "t"].
	MatchSubstring
)

// Config holds configuration for the autocomplete instance.
type Config struct {
	// ProviderConfig contains provider-specific configuration.
	// Each provider defines its own config struct type.
	ProviderConfig interface{}

	// Options contains common autocomplete behavior settings.
	Options Options
}

// Options contains common autocomplete behavior settings.
// Use DefaultOptions() for default values.
type Options struct {
	// DefaultLimit is the default number of results when limit is not specified.
	DefaultLimit int

	// MaxLimit is the maximum number of results that can be requested.
	MaxLimit int

	// CaseSensitive determines if searches are case-sensitive.
	// When false (default), both indexing and querying convert text to lowercase.
	// When true, text preserves its original case during indexing and queries must match exactly.
	// Note: Changing this value requires reindexing all data.
	// Default: false.
	CaseSensitive bool

	// MinPrefixLength is the minimum query length required.
	// Default: 1.
	MinPrefixLength int

	// Namespace prefixes all keys in the storage backend.
	// Enables multiple datasets to coexist (e.g., "prod_users", "staging_products").
	// Default: "autocomplete".
	Namespace string

	// MatchStrategy defines how search terms are matched.
	// Changing this requires reindexing all data.
	// Default: MatchSubstring.
	MatchStrategy MatchStrategy

	// NGramSize is the n-gram size for MatchNGram and MatchNOrMoreGram strategies.
	// Default: 3 (trigrams). Ignored for other strategies.
	NGramSize int
}

// DefaultOptions returns default options with MatchSubstring strategy.
func DefaultOptions() Options {
	return Options{
		DefaultLimit:    defaultLimit,
		MaxLimit:        defaultMaxLimit,
		CaseSensitive:   false,
		MinPrefixLength: 1,
		Namespace:       "autocomplete",
		MatchStrategy:   MatchSubstring,
		NGramSize:       defaultNGramSize,
	}
}

// NewConfig creates a new configuration with default options.
func NewConfig(providerConfig interface{}) Config {
	return Config{
		ProviderConfig: providerConfig,
		Options:        DefaultOptions(),
	}
}

// NewConfigWithOptions creates a new configuration with custom options.
func NewConfigWithOptions(providerConfig interface{}, options Options) Config {
	return Config{
		ProviderConfig: providerConfig,
		Options:        options,
	}
}
