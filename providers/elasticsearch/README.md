# Elasticsearch Autocomplete Provider

The Elasticsearch provider implements the autocomplete interface using Elasticsearch as the storage backend. It supports all standard autocomplete features while leveraging Elasticsearch's powerful search capabilities.

## Features

- Full-text search with multiple matching strategies (prefix, n-gram, substring)
- High-performance search with Elasticsearch's inverted indexes
- Scalable to millions of entries
- Support for complex queries and filtering
- Built-in relevance scoring

## Installation

```go
import (
    _ "github.com/remiges/cvl-kra/autocomplete/providers/elasticsearch"
)
```

## Configuration

```go
config := elasticsearch.Config{
    // Elasticsearch node URLs
    URLs: []string{"http://localhost:9200"},

    // Index name for autocomplete data
    Index: "autocomplete",

    // Authentication (optional)
    Username: "elastic",
    Password: "password", // pragma: allowlist secret

    // Elastic Cloud (alternative to URLs)
    CloudID: "deployment-name:...",

    // API Key authentication (alternative to username/password)
    APIKey: "base64-encoded-key", // pragma: allowlist secret

    // When changes are visible: "true", "false", "wait_for" (default: "false")
    RefreshPolicy: "false",

    // Index settings (only used during automatic index creation)
    NumberOfShards:   1,
    NumberOfReplicas: 0,
}

ac, err := autocomplete.New("elasticsearch", config)
```

## Basic Usage

### Simple String Mode

```go
// Index simple strings
err := ac.Index(ctx, "user-123", "John Doe", "John Doe - Software Engineer")

// Search
results, err := ac.Query(ctx, "john", 10)
```

## Match Strategies

All standard match strategies are supported:

```go
options := autocomplete.WithMatchStrategy(autocomplete.MatchPrefix)
// or MatchNGram, MatchSubstring

results, err := ac.QueryWithOptions(ctx, "query", 10, options)
```

## Index Mapping

The provider creates an optimized index mapping with multiple analyzers:

- **Prefix Analyzer**: Edge n-gram tokenizer for prefix matching
- **N-gram Analyzer**: Full n-gram tokenizer for substring matching
- **Standard Analyzer**: Default text analysis

Each text field is indexed with multiple analyzers for optimal search performance.

## Performance Considerations

### Indexing Performance

- Use `RefreshPolicy: "false"` (default) for best indexing performance
- Batch operations are automatically optimized by Elasticsearch

### Search Performance

- Elasticsearch provides sub-millisecond search for most queries
- Performance scales with cluster size
- Caching is handled automatically by Elasticsearch

### Memory Usage

- Elasticsearch manages memory through its JVM heap
- Index size depends on text length and number of entries

## Elasticsearch Version Compatibility

- Supports Elasticsearch 7.x and 8.x
- Tested with Elasticsearch 8.11+
- Uses the official Elasticsearch Go client

## Comparison with Redis Provider

| Feature | Elasticsearch | Redis |
|---------|--------------|-------|
| Setup Complexity | Higher (requires ES cluster) | Lower (single Redis instance) |
| Search Capabilities | Advanced (full-text, scoring) | Basic (prefix, exact match) |
| Scalability | Horizontal scaling | Vertical scaling |
| Memory Usage | Higher | Lower |
| Query Performance | Sub-millisecond | Sub-millisecond |
| Rich Queries | Yes | Limited |

## Migration from Redis

The Elasticsearch provider maintains full API compatibility:

```go
// Change only the provider name and config
// From:
ac, err := autocomplete.New("redis", redisConfig)

// To:
ac, err := autocomplete.New("elasticsearch", esConfig)
```

All existing code continues to work without changes.

## Advanced Features

### Custom Scoring

Elasticsearch automatically scores results by relevance. The provider uses the score from `IndexOptions`:

```go
options := providers.IndexOptions{
    Score: 2.0, // Higher score = more relevant
}
ac.Index(ctx, "id", "text", "display", options)
```

### Cluster Configuration

For production, configure multiple nodes:

```go
config := elasticsearch.Config{
    URLs: []string{
        "http://node1:9200",
        "http://node2:9200",
        "http://node3:9200",
    },
}
```

### Security

Enable authentication for production:

```go
// Basic auth
config.Username = "elastic"
config.Password = "secure-password" // pragma: allowlist secret

// Or API key
config.APIKey = "base64-encoded-api-key" // pragma: allowlist secret

// Or Elastic Cloud
config.CloudID = "deployment:base64-encoded-info"
```

## Index Management

### Automatic Index Creation

The provider automatically creates the index if it doesn't exist, using the `NumberOfShards` and `NumberOfReplicas` settings from the configuration. This is convenient for development and testing.

**Important Notes:**
- `NumberOfShards` and `NumberOfReplicas` are **only** used when the index is automatically created
- If the index already exists, these settings are ignored
- You cannot change the number of primary shards after index creation

### Production Deployment

For production environments, it's recommended to pre-create indices with appropriate settings:

```bash
# Create index with custom settings
curl -X PUT "localhost:9200/autocomplete" -H 'Content-Type: application/json' -d'
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "index.max_ngram_diff": 20,
    "analysis": {
      "analyzer": {
        "prefix_analyzer": {
          "tokenizer": "standard",
          "filter": ["lowercase", "edge_ngram_filter"]
        },
        "ngram_analyzer": {
          "tokenizer": "ngram_tokenizer",
          "filter": ["lowercase"]
        },
        "substring_analyzer": {
          "tokenizer": "standard",
          "filter": ["lowercase", "substring_filter"]
        }
      },
      "tokenizer": {
        "ngram_tokenizer": {
          "type": "ngram",
          "min_gram": 3,
          "max_gram": 20
        }
      },
      "filter": {
        "edge_ngram_filter": {
          "type": "edge_ngram",
          "min_gram": 1,
          "max_gram": 20
        },
        "substring_filter": {
          "type": "ngram",
          "min_gram": 3,
          "max_gram": 20
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": {"type": "keyword"},
      "key": {"type": "keyword"},
      "text": {
        "type": "text",
        "fields": {
          "prefix": {
            "type": "text",
            "analyzer": "prefix_analyzer",
            "search_analyzer": "standard"
          },
          "ngram": {
            "type": "text",
            "analyzer": "ngram_analyzer"
          },
          "substring": {
            "type": "text",
            "analyzer": "substring_analyzer"
          },
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "display": {"type": "text"},
      "score": {"type": "float"},
      "case_sensitive": {"type": "boolean"}
    }
  }
}'
```

### When to Use Auto-Creation vs Pre-Creation

**Use Auto-Creation (rely on config settings) when:**
- Developing locally
- Running tests
- Deploying simple applications with predictable load
- Using default settings (1 shard, 0 replicas) is acceptable

**Pre-Create Indices when:**
- Deploying to production
- Need specific shard/replica counts based on cluster size
- Require custom analyzers or mappings
- Managing multiple environments with different settings

## Troubleshooting

### Connection Issues

```go
// The provider tests connection during creation
ac, err := autocomplete.New("elasticsearch", config)
if err != nil {
    // Connection failed - check URLs and credentials
}
```

### Index Not Created

The provider automatically creates the index with optimal settings if it doesn't exist. If you need custom settings, create the index manually before using the provider. The auto-creation uses the `NumberOfShards` and `NumberOfReplicas` values from your config.

### Search Not Finding Results

1. Check `RefreshPolicy` - set to "true" for immediate visibility during testing
2. Check Elasticsearch logs for errors

## Example Applications

See the `examples/elasticsearch/` directory for:

- `basic/` - Simple string-based autocomplete
- `advanced/` - Advanced autocomplete with structured data
