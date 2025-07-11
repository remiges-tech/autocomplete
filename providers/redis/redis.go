// Package redis implements the autocomplete Provider interface using Redis as the storage backend.
// It uses Redis sorted sets for autocomplete operations with support for multiple
// matching strategies including prefix, n-gram, and substring matching.
package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"

	"github.com/remiges/cvl-kra/autocomplete/providers"
)

const (
	// prefixSet is the Redis key prefix for sorted sets storing tokens → IDs with scores.
	prefixSet = "ac:set:"

	// prefixDisplay is the Redis key prefix for hash maps storing ID → display text.
	prefixDisplay = "ac:display:"

	// prefixText is the Redis key prefix for hash maps storing ID → original text.
	prefixText = "ac:text:"

	// prefixMeta is the Redis key prefix for hash maps storing ID → metadata.
	prefixMeta = "ac:meta:"

	// defaultNGramSize is the default n-gram size when not specified in options.
	defaultNGramSize = 3

	// lexicographicMaxChar is the lexicographic maximum character for ZRANGEBYLEX upper bound.
	lexicographicMaxChar = "\xff"

	// resultMultiplierForDuplicates is the score multiplier when same ID appears multiple times.
	resultMultiplierForDuplicates = 10

	// resultMultiplierForIntersection is the score multiplier for IDs matching all query tokens.
	resultMultiplierForIntersection = 20

	// memberFormatPrefix is the basic format for sorted set entries: token:id.
	memberFormatPrefix = "%s:%s"

	// memberFormatWithPosition is the format with position: token:id:position.
	memberFormatWithPosition = "%s:%s:%d"

	// minMemberPartsForID is the minimum parts after split for basic format.
	minMemberPartsForID = 2

	// minMemberPartsForPositionalID is the minimum parts for positional format.
	minMemberPartsForPositionalID = 3
)

// Provider implements the autocomplete Provider interface using Redis.
// It uses Redis sorted sets for storage and retrieval of autocomplete entries.
// All methods are safe for concurrent use.
type Provider struct {
	client *redis.Client
}

// Config holds Redis connection parameters.
type Config struct {
	// Addr is the Redis server address in the format "host:port".
	Addr string

	// Password is the Redis password (empty string for no password).
	Password string

	// DB is the Redis database number (0-15, default is 0).
	// Redis Cluster only supports DB 0.
	DB int
}

// New creates a new Redis provider with the given configuration.
// It establishes a connection to Redis and verifies connectivity with a PING command.
func New(config Config) (*Provider, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password, // pragma: allowlist secret
		DB:       config.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Provider{
		client: client,
	}, nil
}

// intersectIDSets returns IDs that appear in all sets
func intersectIDSets(sets []map[string]bool) []string {
	if len(sets) == 0 {
		return []string{}
	}
	if len(sets) == 1 {
		return extractKeysFromSet(sets[0])
	}

	intersection := copySet(sets[0])
	removeNonIntersectingIDs(intersection, sets[1:])
	return extractKeysFromSet(intersection)
}

// queryNGramSlidingWindow performs sliding window search for n-gram queries longer than n
func (p *Provider) queryNGramSlidingWindow(
	ctx context.Context, key, searchQuery string, n int, options providers.QueryOptions,
) ([]providers.ProviderResult, error) {
	var ngramSets []map[string]bool

	for i := 0; i <= len(searchQuery)-n; i++ {
		ngram := searchQuery[i : i+n]
		start := createLexicographicStartKey(ngram)
		end := createLexicographicEndKey(ngram)

		results, err := p.client.ZRangeByLex(ctx, prefixSet+key, &redis.ZRangeBy{
			Min:    start,
			Max:    end,
			Offset: 0,
			Count:  int64(options.MaxResults * resultMultiplierForIntersection),
		}).Result()

		if err != nil {
			return nil, fmt.Errorf("failed to query n-gram '%s': %w", ngram, err)
		}
		idSet := extractIDsFromResults(results, minMemberPartsForPositionalID)
		if isEmptySet(idSet) {
			return []providers.ProviderResult{}, nil
		}

		ngramSets = append(ngramSets, idSet)
	}
	ids := intersectIDSets(ngramSets)
	ids = limitResults(ids, options.MaxResults)
	return p.fetchProviderResults(ctx, key, ids)
}

// fetchProviderResults fetches full data for given IDs
func (p *Provider) fetchProviderResults(
	ctx context.Context, key string, ids []string,
) ([]providers.ProviderResult, error) {
	if len(ids) == 0 {
		return []providers.ProviderResult{}, nil
	}

	providerResults := make([]providers.ProviderResult, 0, len(ids))

	displayList, err := p.client.HMGet(ctx, prefixDisplay+key, ids...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch display texts: %w", err)
	}
	for i, id := range ids {
		if displayList[i] == nil {
			continue
		}

		display, ok := displayList[i].(string)
		if !ok {
			continue
		}

		result := providers.ProviderResult{
			ID:      id,
			Display: display,
			Score:   1.0,
		}

		providerResults = append(providerResults, result)
	}

	return providerResults, nil
}

// Index adds or updates an entry in the Redis autocomplete index
func (p *Provider) Index(ctx context.Context, key, id, text, display string, options providers.IndexOptions) error {
	pipe := p.client.Pipeline()

	// Store both original and lowercase versions if needed
	textToIndex := text
	if !options.CaseSensitive {
		textToIndex = strings.ToLower(text)
	}

	switch options.MatchStrategy {
	case providers.MatchPrefix:
		for i := 1; i <= len(textToIndex); i++ {
			prefix := textToIndex[:i]
			member := createPrefixMember(prefix, id)
			pipe.ZAdd(ctx, prefixSet+key, &redis.Z{
				Score:  options.Score,
				Member: member,
			})
		}

	case providers.MatchNGram:
		n := getNGramSizeOrDefault(options.NGramSize)
		for i := 0; i <= len(textToIndex)-n; i++ {
			ngram := textToIndex[i : i+n]
			member := createPositionalMember(ngram, id, i)
			pipe.ZAdd(ctx, prefixSet+key, &redis.Z{
				Score:  options.Score,
				Member: member,
			})
		}

	case providers.MatchNOrMoreGram:
		n := getNGramSizeOrDefault(options.NGramSize)
		for start := 0; start < len(textToIndex); start++ {
			for end := start + n; end <= len(textToIndex); end++ {
				substring := textToIndex[start:end]
				member := createPositionalMember(substring, id, start)
				pipe.ZAdd(ctx, prefixSet+key, &redis.Z{
					Score:  options.Score,
					Member: member,
				})
			}
		}

	case providers.MatchSubstring:
		for start := 0; start < len(textToIndex); start++ {
			for end := start + 1; end <= len(textToIndex); end++ {
				substring := textToIndex[start:end]
				member := createPositionalMember(substring, id, start)
				pipe.ZAdd(ctx, prefixSet+key, &redis.Z{
					Score:  options.Score,
					Member: member,
				})
			}
		}
	}
	pipe.HSet(ctx, prefixText+key, id, text)
	pipe.HSet(ctx, prefixDisplay+key, id, display)
	// Store case sensitivity metadata
	if options.CaseSensitive {
		pipe.HSet(ctx, prefixMeta+key, id, "1")
	} else {
		pipe.HDel(ctx, prefixMeta+key, id)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Query searches for entries matching the given query
func (p *Provider) Query(ctx context.Context, key, query string, options providers.QueryOptions) ([]providers.ProviderResult, error) {
	searchQuery := query
	if !options.CaseSensitive {
		searchQuery = strings.ToLower(query)
	}

	if options.MatchStrategy == providers.MatchNGram {
		n := getNGramSizeOrDefault(options.NGramSize)

		if len(searchQuery) < 1 {
			return []providers.ProviderResult{}, nil
		}
		if len(searchQuery) > n {
			return p.queryNGramSlidingWindow(ctx, key, searchQuery, n, options)
		}
	}
	if options.MatchStrategy == providers.MatchNOrMoreGram {
		n := getNGramSizeOrDefault(options.NGramSize)
		if len(searchQuery) < n {
			return []providers.ProviderResult{}, nil
		}
	}
	start := createLexicographicStartKey(searchQuery)
	end := createLexicographicEndKey(searchQuery)
	results, err := p.client.ZRangeByLex(ctx, prefixSet+key, &redis.ZRangeBy{
		Min:    start,
		Max:    end,
		Offset: 0,
		Count:  int64(options.MaxResults * resultMultiplierForDuplicates),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to query autocomplete: %w", err)
	}
	ids := extractUniqueIDsFromResults(results, options)
	return p.fetchProviderResults(ctx, key, ids)
}

// Delete removes an entry from the index
func (p *Provider) Delete(ctx context.Context, key, id string) error {
	pipe := p.client.Pipeline()

	text, err := p.client.HGet(ctx, prefixText+key, id).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get text for deletion: %w", err)
	}

	if text != "" {
		// Check if entry was indexed with case sensitivity
		caseSensitive := false
		meta, metaErr := p.client.HGet(ctx, prefixMeta+key, id).Result()
		if metaErr == nil && meta == "1" {
			caseSensitive = true
		}

		textToDelete := text
		if !caseSensitive {
			textToDelete = strings.ToLower(text)
		}
		removePrefixMembers(pipe, ctx, prefixSet+key, textToDelete, id)
		removePositionalMembers(pipe, ctx, prefixSet+key, textToDelete, id)
	}
	pipe.HDel(ctx, prefixText+key, id)
	pipe.HDel(ctx, prefixDisplay+key, id)
	pipe.HDel(ctx, prefixMeta+key, id)

	_, err = pipe.Exec(ctx)
	return err
}

// DeleteAll removes all entries for a given key
func (p *Provider) DeleteAll(ctx context.Context, key string) error {
	pipe := p.client.Pipeline()

	deleteAllKeysForNamespace(pipe, ctx, key)

	_, err := pipe.Exec(ctx)
	return err
}

// Close closes the Redis connection
func (p *Provider) Close() error {
	return p.client.Close()
}

func createLexicographicStartKey(query string) string {
	return fmt.Sprintf("[%s", query)
}

func createLexicographicEndKey(query string) string {
	return fmt.Sprintf("[%s%s", query, lexicographicMaxChar)
}

func getNGramSizeOrDefault(size int) int {
	if size <= 0 {
		return defaultNGramSize
	}
	return size
}

func createPrefixMember(prefix, id string) string {
	return fmt.Sprintf(memberFormatPrefix, prefix, id)
}

func createPositionalMember(text, id string, position int) string {
	return fmt.Sprintf(memberFormatWithPosition, text, id, position)
}

func extractIDsFromResults(results []string, minParts int) map[string]bool {
	idSet := make(map[string]bool)
	for _, result := range results {
		if id := extractIDFromMember(result, minParts); id != "" {
			idSet[id] = true
		}
	}
	return idSet
}

func extractIDFromMember(member string, minParts int) string {
	parts := strings.Split(member, ":")
	if len(parts) >= minParts {
		return parts[1]
	}
	return ""
}

func isEmptySet(set map[string]bool) bool {
	return len(set) == 0
}

func limitResults(ids []string, maxResults int) []string {
	if len(ids) > maxResults {
		return ids[:maxResults]
	}
	return ids
}

func extractUniqueIDsFromResults(results []string, options providers.QueryOptions) []string {
	uniqueIDs := make(map[string]bool)
	var ids []string

	minParts := getMinPartsForStrategy(options.MatchStrategy)

	for _, result := range results {
		id := extractIDFromMember(result, minParts)

		if id != "" && !uniqueIDs[id] {
			uniqueIDs[id] = true
			ids = append(ids, id)
			if len(ids) >= options.MaxResults {
				break
			}
		}
	}

	return ids
}

func getMinPartsForStrategy(strategy providers.MatchStrategy) int {
	if strategy == providers.MatchPrefix {
		return minMemberPartsForID
	}
	return minMemberPartsForPositionalID
}

func removePrefixMembers(pipe redis.Pipeliner, ctx context.Context, key, text, id string) {
	for i := 1; i <= len(text); i++ {
		prefix := text[:i]
		member := createPrefixMember(prefix, id)
		pipe.ZRem(ctx, key, member)
	}
}

func removePositionalMembers(pipe redis.Pipeliner, ctx context.Context, key, text, id string) {
	for start := 0; start < len(text); start++ {
		for end := start + 1; end <= len(text); end++ {
			substring := text[start:end]
			member := createPositionalMember(substring, id, start)
			pipe.ZRem(ctx, key, member)
		}
	}
}

func deleteAllKeysForNamespace(pipe redis.Pipeliner, ctx context.Context, key string) {
	pipe.Del(ctx, prefixSet+key)
	pipe.Del(ctx, prefixText+key)
	pipe.Del(ctx, prefixDisplay+key)
	pipe.Del(ctx, prefixMeta+key)
}

func copySet(source map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for id := range source {
		result[id] = true
	}
	return result
}

func removeNonIntersectingIDs(intersection map[string]bool, remainingSets []map[string]bool) {
	for _, set := range remainingSets {
		for id := range intersection {
			if !set[id] {
				delete(intersection, id)
			}
		}
	}
}

func extractKeysFromSet(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	return result
}
