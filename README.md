# Autocomplete Package

An autocomplete package for Go with support for multiple storage backends. Currently supports Redis with plans for PostgreSQL (trigram), Elasticsearch, and other RDBMS backends.

## Features

- **Common Interface**: API for indexing and querying autocomplete data
- **Multiple Providers**: Supports different storage backends
- **Substring Matching**: Search for any part of the text, not just prefixes
- **Display Text Support**: Store custom display text with each entry
- **Case-Insensitive Search**: Default case-insensitive matching
- **Configurable Options**: Customizable limits, prefix lengths, and namespaces

## Installation

```bash
go get github.com/remiges-tech/autocomplete
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/remiges-tech/autocomplete"
    "github.com/remiges-tech/autocomplete/providers/redis"
    _ "github.com/remiges-tech/autocomplete/providers/redis" // Register provider
)

func main() {
    // Configure Redis provider
    redisConfig := redis.Config{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    }

    // Create autocomplete instance
    config := autocomplete.NewConfig(redisConfig)
    ac, err := autocomplete.New("redis", config)
    if err != nil {
        log.Fatal(err)
    }
    defer ac.Close()

    ctx := context.Background()

    // Index Indian postal codes with location details
    err = ac.Index(ctx, "400001", "400001 Mumbai", "400001 - Mumbai, Maharashtra")
    if err != nil {
        log.Fatal(err)
    }

    err = ac.Index(ctx, "110001", "110001 Delhi", "110001 - New Delhi, Delhi")
    if err != nil {
        log.Fatal(err)
    }

    err = ac.Index(ctx, "560001", "560001 Bangalore", "560001 - Bangalore, Karnataka")
    if err != nil {
        log.Fatal(err)
    }

    // Query - supports substring matching!
    results, err := ac.Query(ctx, "mumbai", 10)  // Will find Mumbai postal code
    if err != nil {
        log.Fatal(err)
    }

    for _, result := range results {
        log.Printf("Found: %s (ID: %s)", result.Display, result.ID)
    }
}
```

## API

### AutoComplete Interface

```go
type AutoComplete interface {
    // Index adds or updates a text entry for autocomplete
    Index(ctx context.Context, id string, text string, display string) error

    // Query searches for entries matching the given search term (substring matching)
    Query(ctx context.Context, searchTerm string, limit int) ([]Result, error)

    // Delete removes an entry from the autocomplete index
    Delete(ctx context.Context, id string) error

    // DeleteAll removes all entries from the autocomplete index
    DeleteAll(ctx context.Context) error

    // Close closes the autocomplete provider and releases resources
    Close() error
}
```

### Result Structure

```go
type Result struct {
    ID      string  // Unique identifier for the entry
    Display string  // Display text for the entry
    Score   float64 // Relevance score (higher is better)
}
```

### Configuration

```go
// Create config with default options
config := autocomplete.NewConfig(providerConfig)

// Or customize options
config := autocomplete.NewConfigWithOptions(providerConfig, autocomplete.Options{
    DefaultLimit:    10,
    MaxLimit:        100,
    CaseSensitive:   false,
    MinPrefixLength: 1,
    Namespace:       "myapp",
    MatchStrategy:   autocomplete.MatchSubstring,
    NGramSize:       3,
})
```

## Match Strategies

The package supports multiple matching strategies to balance between functionality and storage:

### 1. Prefix Matching (`MatchPrefix`)
- Traditional autocomplete behavior
- Matches only from the beginning of the text
- Lowest storage overhead (O(n) where n is text length)
- Best for: Classic autocomplete use cases

### 2. N-Gram Matching (`MatchNGram`)
- Indexes fixed-length character sequences (e.g., 3-character sequences)
- **Sliding Window for Long Queries**: Queries longer than n use AND logic
  - Query "apple" with n=3 -> Finds entries containing "app" AND "ppl" AND "ple"
  - Ensures all parts of the query match, improving precision
- Good for typo tolerance and partial matches
- Moderate storage overhead (O(n))
- Best for: When you need fuzzy matching with controlled storage and precise results

### 3. N-or-More-Gram Matching (`MatchNOrMoreGram`)
- Indexes all substrings of length n or greater
- Balances between flexibility and storage
- Higher storage overhead (O(n^2) but less than full substring)
- Best for: When you need substring matching but want to limit short matches

### 4. Substring Matching (`MatchSubstring`)
- Indexes all possible substrings
- Maximum flexibility - find any part of the text
- Highest storage overhead (O(n^2))
- Best for: When you need to find any substring regardless of position

### Example: Setting Match Strategy

```go
// Use prefix matching (traditional autocomplete)
config.Options.MatchStrategy = autocomplete.MatchPrefix

// Use n-gram matching with 3-character sequences
config.Options.MatchStrategy = autocomplete.MatchNGram
config.Options.NGramSize = 3

// Use substring matching (default)
config.Options.MatchStrategy = autocomplete.MatchSubstring
```

### Query Behavior Examples

Given indexed text: "Apple iPhone Pro"

| Strategy | Query | Result | Explanation |
|----------|-------|--------|-------------|
| MatchPrefix | "app" | Match | Starts with "app" |
| MatchPrefix | "iphone" | No match | Doesn't start with "iphone" |
| MatchNGram (n=3) | "pho" | Match | Contains the 3-gram "pho" |
| MatchNGram (n=3) | "phone" | Match | Contains "pho" AND "hon" AND "one" |
| MatchNOrMoreGram (n=3) | "phone" | Match | Contains substring "phone" (>=3 chars) |
| MatchSubstring | "phone" | Match | Contains substring "phone" |

## Redis Provider

The Redis provider uses sorted sets for efficient matching:

- Adapts storage strategy based on selected match type
- Uses ZRANGEBYLEX for efficient queries
- Stores display text separately for custom presentation
- Handles all match strategies efficiently

### Redis Configuration

```go
redisConfig := redis.Config{
    Addr:     "localhost:6379",
    Password: "optional-password",  // pragma: allowlist secret
    DB:       0,
}
```

### Storage and Performance Comparison

For a 20-character text like "Apple iPhone 14 Pro":

| Strategy | Storage Entries | Index Time | Query Time | Use Case |
|----------|----------------|------------|------------|----------|
| MatchPrefix | ~20 | O(n) | O(log n) | Traditional autocomplete |
| MatchNGram (n=3) | ~18 | O(n) | O(log n) | Fuzzy matching |
| MatchNOrMoreGram (n=3) | ~171 | O(n^2) | O(log n) | Flexible substring search |
| MatchSubstring | ~210 | O(n^2) | O(log n) | Full substring search |

### Choosing the Right Strategy

1. **Use MatchPrefix when:**
   - You need traditional autocomplete behavior
   - Storage space is limited
   - Users type from the beginning of words

2. **Use MatchNGram when:**
   - You want to handle typos
   - You need predictable storage usage
   - You want partial word matching

3. **Use MatchSubstring when:**
   - Users might search for any part of the text
   - Storage space is not a primary concern
   - Maximum search flexibility is required

## Running Tests

```bash
# Ensure Redis is running locally
docker run -d -p 6379:6379 redis:latest

# Run tests
go test ./...
```

## Running the Examples

### Basic Example
```bash
cd examples/basic
go run main.go
```

### Indian Postal Codes Example
```bash
cd examples/indian-postal-codes
go run main.go
```

This example demonstrates autocomplete functionality for Indian postal codes (PIN codes) with location information. Features include:
- 6-digit Indian PIN codes from major cities
- Search by PIN code, city, district, or state name
- Support for fuzzy matching with typos
- Custom display text showing location details

For more details, see the [Indian Postal Codes Example README](examples/indian-postal-codes/README.md).

## Future Providers

Planned support for:
- PostgreSQL with pg_trgm extension
- Elasticsearch
- Generic SQL databases
- In-memory provider for testing

## License

[Add your license here]
