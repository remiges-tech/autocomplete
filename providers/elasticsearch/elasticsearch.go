package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/remiges/cvl-kra/autocomplete/providers"
)

const (
	// defaultMaxResults is the default maximum number of results if not specified.
	defaultMaxResults = 10

	// indexMappingTemplate is the Elasticsearch index mapping for autocomplete.
	indexMappingTemplate = `{
		"settings": {
			"number_of_shards": %d,
			"number_of_replicas": %d,
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
	}`
)

// Provider implements the autocomplete Provider interface using Elasticsearch.
type Provider struct {
	client        *elasticsearch.Client
	index         string
	refreshPolicy string
}

// document represents the structure stored in Elasticsearch.
type document struct {
	ID            string  `json:"id"`
	Key           string  `json:"key"`
	Text          string  `json:"text"`
	Display       string  `json:"display"`
	Score         float64 `json:"score"`
	CaseSensitive bool    `json:"case_sensitive"`
}

// searchHit represents a single search result from Elasticsearch.
type searchHit struct {
	Score  float64  `json:"_score"`
	Source document `json:"_source"`
}

// searchResponse represents the Elasticsearch search response.
type searchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []searchHit `json:"hits"`
	} `json:"hits"`
}

// New creates a new Elasticsearch provider with the given configuration.
func New(config *Config) (*Provider, error) {
	config.setDefaults()

	// Build Elasticsearch configuration
	esConfig := elasticsearch.Config{
		Addresses: config.URLs,
		Username:  config.Username,
		Password:  config.Password,
		CloudID:   config.CloudID,
		APIKey:    config.APIKey,
	}

	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Test connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch connection error: %s", res.String())
	}

	provider := &Provider{
		client:        client,
		index:         config.Index,
		refreshPolicy: config.RefreshPolicy,
	}

	// Create index if it doesn't exist
	if err := provider.createIndexIfNotExists(config); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return provider, nil
}

// createIndexIfNotExists creates the index with appropriate mappings if it doesn't exist.
func (p *Provider) createIndexIfNotExists(config *Config) error {
	exists, err := p.indexExists()
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	mapping := fmt.Sprintf(indexMappingTemplate, config.NumberOfShards, config.NumberOfReplicas)

	req := esapi.IndicesCreateRequest{
		Index: p.index,
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(context.Background(), p.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil
}

// indexExists checks if the index exists.
func (p *Provider) indexExists() (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{p.index},
	}

	res, err := req.Do(context.Background(), p.client)
	if err != nil {
		return false, err
	}
	defer func() { _ = res.Body.Close() }()

	const httpOK = 200
	return res.StatusCode == httpOK, nil
}

// Index adds or updates an entry in the Elasticsearch autocomplete index.
func (p *Provider) Index(ctx context.Context, key, id, text, display string, options providers.IndexOptions) error {
	doc := document{
		ID:            id,
		Key:           key,
		Text:          text,
		Display:       display,
		Score:         options.Score,
		CaseSensitive: options.CaseSensitive,
	}

	// Prepare document for indexing
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	// Index document
	req := esapi.IndexRequest{
		Index:      p.index,
		DocumentID: generateDocumentID(key, id),
		Body:       bytes.NewReader(docJSON),
		Refresh:    p.refreshPolicy,
	}

	res, err := req.Do(ctx, p.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		return fmt.Errorf("failed to index document: %s", res.String())
	}

	return nil
}

// Query searches for entries matching the given query.
func (p *Provider) Query(ctx context.Context, key, query string, options providers.QueryOptions) ([]providers.ProviderResult, error) {
	// Build query based on match strategy
	esQuery := p.buildQuery(key, query, options)

	// Prepare search request
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(esQuery); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	// Execute search
	size := options.MaxResults
	if size <= 0 {
		size = defaultMaxResults
	}

	req := esapi.SearchRequest{
		Index: []string{p.index},
		Body:  &buf,
		Size:  &size,
	}

	res, err := req.Do(ctx, p.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		return nil, fmt.Errorf("search failed: %s", res.String())
	}

	// Parse response
	return p.parseSearchResponse(res.Body)
}

// buildQuery constructs the Elasticsearch query based on match strategy.
func (p *Provider) buildQuery(key, query string, options providers.QueryOptions) map[string]interface{} {
	// Base query with key filter
	baseQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"key": key,
						},
					},
				},
			},
		},
	}

	// Prepare query text
	queryText := query
	if !options.CaseSensitive {
		queryText = strings.ToLower(query)
	}

	// Add match query based on strategy
	var matchQuery map[string]interface{}

	switch options.MatchStrategy {
	case providers.MatchPrefix:
		matchQuery = map[string]interface{}{
			"match": map[string]interface{}{
				"text.prefix": queryText,
			},
		}
	case providers.MatchNGram:
		matchQuery = map[string]interface{}{
			"match": map[string]interface{}{
				"text.ngram": queryText,
			},
		}
	case providers.MatchSubstring:
		matchQuery = map[string]interface{}{
			"match": map[string]interface{}{
				"text.substring": queryText,
			},
		}
	case providers.MatchNOrMoreGram:
		// Use substring matching for variable-length n-grams
		matchQuery = map[string]interface{}{
			"match": map[string]interface{}{
				"text.substring": queryText,
			},
		}
	default:
		// Default to prefix matching
		matchQuery = map[string]interface{}{
			"match": map[string]interface{}{
				"text": queryText,
			},
		}
	}

	// Add the match query to must clause only if we have a query
	boolQuery := baseQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})
	if query != "" {
		boolQuery["must"] = []interface{}{matchQuery}
	}

	// Add minimum score filter if specified
	if options.MinScore > 0 {
		baseQuery["min_score"] = options.MinScore
	}

	return baseQuery
}

// parseSearchResponse parses the Elasticsearch response into provider results.
func (p *Provider) parseSearchResponse(body io.Reader) ([]providers.ProviderResult, error) {
	var response searchResponse
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]providers.ProviderResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result := providers.ProviderResult{
			ID:      hit.Source.ID,
			Display: hit.Source.Display,
			Score:   hit.Score,
		}
		results = append(results, result)
	}

	return results, nil
}

// Delete removes an entry from the index.
func (p *Provider) Delete(ctx context.Context, key, id string) error {
	req := esapi.DeleteRequest{
		Index:      p.index,
		DocumentID: generateDocumentID(key, id),
		Refresh:    p.refreshPolicy,
	}

	res, err := req.Do(ctx, p.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	// 404 is not an error for delete (idempotent)
	const httpNotFound = 404
	if res.IsError() && res.StatusCode != httpNotFound {
		return fmt.Errorf("failed to delete document: %s", res.String())
	}

	return nil
}

// DeleteAll removes all entries for a given key namespace.
func (p *Provider) DeleteAll(ctx context.Context, key string) error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"key": key,
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return fmt.Errorf("failed to encode query: %w", err)
	}

	req := esapi.DeleteByQueryRequest{
		Index:   []string{p.index},
		Body:    &buf,
		Refresh: &[]bool{p.refreshPolicy == "true"}[0],
	}

	res, err := req.Do(ctx, p.client)
	if err != nil {
		return fmt.Errorf("failed to delete by query: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		return fmt.Errorf("failed to delete by query: %s", res.String())
	}

	return nil
}

// Close closes the provider connection.
func (p *Provider) Close() error {
	// The Elasticsearch Go client doesn't have a Close method
	// as it uses standard HTTP connections that are managed by Go's http package
	return nil
}

// generateDocumentID creates a unique document ID from key and id.
func generateDocumentID(key, id string) string {
	return fmt.Sprintf("%s:%s", key, id)
}
