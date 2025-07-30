package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/remiges-tech/autocomplete/providers"
)

const testKey = "test"

var (
	sharedContainer testcontainers.Container
	sharedProvider  *Provider
)

// TestMain sets up a shared Redis container for all tests
func TestMain(m *testing.M) {
	// Setup
	ctx := context.Background()
	container, provider, err := setupSharedContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test container: %v", err)
	}

	sharedContainer = container
	sharedProvider = provider

	// Run tests
	code := m.Run()

	// Cleanup
	if sharedContainer != nil {
		if err := sharedContainer.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate container: %v", err)
		}
	}

	os.Exit(code)
}

func setupSharedContainer(ctx context.Context) (testcontainers.Container, *Provider, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return nil, nil, err
	}

	config := Config{
		Addr:     fmt.Sprintf("%s:%s", host, port.Port()),
		Password: "",
		DB:       0,
	}

	provider, err := New(config)
	if err != nil {
		return nil, nil, err
	}

	return container, provider, nil
}

func getTestRedisClient(t *testing.T) *Provider {
	if sharedProvider == nil {
		t.Fatal("Redis provider not initialized")
	}

	// Clear the database before each test
	ctx := context.Background()
	if err := sharedProvider.client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to flush database: %v", err)
	}

	return sharedProvider
}

func TestRedisProvider_Index(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey

	tests := []struct {
		name    string
		id      string
		text    string
		display string
		wantErr bool
	}{
		{
			name:    "simple index",
			id:      "1",
			text:    "John Doe",
			display: "John Doe - john@example.com",
			wantErr: false,
		},
		{
			name:    "index with simple display",
			id:      "2",
			text:    "Jane Smith",
			display: "Jane Smith",
			wantErr: false,
		},
		{
			name:    "update existing entry",
			id:      "1",
			text:    "John Updated",
			display: "John Updated - john.new@example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.Index(ctx, key, tt.id, tt.text, tt.display, providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchSubstring,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("Index() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisProvider_Query(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey
	testData := []struct {
		id      string
		text    string
		display string
	}{
		{"1", "John Doe", "John Doe - Person"},
		{"2", "John Smith", "John Smith - Person"},
		{"3", "Johnny Appleseed", "Johnny Appleseed - Person"},
		{"4", "Jane Doe", "Jane Doe - Person"},
		{"5", "Product: John Deere Tractor", "John Deere Tractor - Product"},
	}

	for _, data := range testData {
		err := provider.Index(ctx, key, data.id, data.text, data.display, providers.IndexOptions{
			Score:         1.0,
			MatchStrategy: providers.MatchSubstring,
		})
		if err != nil {
			t.Fatalf("Failed to index test data: %v", err)
		}
	}

	tests := []struct {
		name        string
		prefix      string
		options     providers.QueryOptions
		wantResults int
		wantIDs     []string
	}{
		{
			name:   "substring 'john'",
			prefix: "john",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 4,
			wantIDs:     []string{"1", "2", "3", "5"},
		},
		{
			name:   "prefix 'jane'",
			prefix: "jane",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 1,
			wantIDs:     []string{"4"},
		},
		{
			name:   "substring 'doe'",
			prefix: "doe",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 2,
			wantIDs:     []string{"1", "4"},
		},
		{
			name:   "substring 'smith'",
			prefix: "smith",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 1,
			wantIDs:     []string{"2"},
		},
		{
			name:   "substring 'apple'",
			prefix: "apple",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 1,
			wantIDs:     []string{"3"},
		},
		{
			name:   "substring 'deere'",
			prefix: "deere",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 1,
			wantIDs:     []string{"5"},
		},
		{
			name:   "limited results",
			prefix: "john",
			options: providers.QueryOptions{
				MaxResults:    2,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 2,
		},
		{
			name:   "no matches",
			prefix: "xyz",
			options: providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchSubstring,
			},
			wantResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := provider.Query(ctx, key, tt.prefix, tt.options)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			if len(results) != tt.wantResults {
				t.Errorf("Query() returned %d results, want %d", len(results), tt.wantResults)
			}
			if tt.wantIDs != nil {
				resultIDs := make(map[string]bool)
				for _, r := range results {
					resultIDs[r.ID] = true
				}

				for _, wantID := range tt.wantIDs {
					if !resultIDs[wantID] {
						t.Errorf("Query() missing expected ID %s", wantID)
					}
				}
			}
		})
	}
}

func TestRedisProvider_MatchStrategies(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()

	tests := []struct {
		name          string
		strategy      providers.MatchStrategy
		ngramSize     int
		indexText     string
		searchQueries []struct {
			query       string
			shouldMatch bool
		}
	}{
		{
			name:      "MatchPrefix",
			strategy:  providers.MatchPrefix,
			indexText: "Apple iPhone Pro",
			searchQueries: []struct {
				query       string
				shouldMatch bool
			}{
				{"app", true},
				{"apple", true},
				{"iphone", false},
				{"phone", false},
				{"pro", false},
			},
		},
		{
			name:      "MatchNGram",
			strategy:  providers.MatchNGram,
			ngramSize: 3,
			indexText: "Apple iPhone",
			searchQueries: []struct {
				query       string
				shouldMatch bool
			}{
				{"app", true},
				{"ppl", true},
				{"ple", true},
				{"pho", true},
				{"hon", true},
				{"one", true},
				{"ap", true},
				{"appl", true},
				{"apple", true},
				{"phone", true},
				{"iphone", true},
				{"xyz", false},
			},
		},
		{
			name:      "MatchNOrMoreGram",
			strategy:  providers.MatchNOrMoreGram,
			ngramSize: 3,
			indexText: "Apple iPhone",
			searchQueries: []struct {
				query       string
				shouldMatch bool
			}{
				{"app", true},
				{"appl", true},
				{"apple", true},
				{"pho", true},
				{"phon", true},
				{"phone", true},
				{"iphone", true},
				{"ap", false},
				{"ph", false},
			},
		},
		{
			name:      "MatchSubstring",
			strategy:  providers.MatchSubstring,
			indexText: "Apple iPhone",
			searchQueries: []struct {
				query       string
				shouldMatch bool
			}{
				{"a", true},
				{"ap", true},
				{"app", true},
				{"ppl", true},
				{"ple", true},
				{"phone", true},
				{"iphone", true},
				{"e i", true},
				{"xyz", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := provider.DeleteAll(ctx, tt.name); err != nil {
				t.Errorf("DeleteAll() error = %v", err)
			}
			err := provider.Index(ctx, tt.name, "1", tt.indexText, tt.indexText, providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: tt.strategy,
				NGramSize:     tt.ngramSize,
			})
			if err != nil {
				t.Fatalf("Failed to index: %v", err)
			}
			for _, sq := range tt.searchQueries {
				results, err := provider.Query(ctx, tt.name, sq.query, providers.QueryOptions{
					MaxResults:    10,
					MatchStrategy: tt.strategy,
					NGramSize:     tt.ngramSize,
				})
				if err != nil {
					t.Errorf("Query failed for '%s': %v", sq.query, err)
					continue
				}

				found := len(results) > 0
				if found != sq.shouldMatch {
					t.Errorf("Query '%s': expected match=%v, got match=%v", sq.query, sq.shouldMatch, found)
				}
			}
		})
	}
}

func TestRedisProvider_NGramSlidingWindow(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := "test_sliding"
	testData := []struct {
		id   string
		text string
	}{
		{"1", "bookshelf"},
		{"2", "notebook"},
		{"3", "facebook"},
		{"4", "shelfware"},
		{"5", "bookkeeper"},
	}

	for _, data := range testData {
		err := provider.Index(ctx, key, data.id, data.text, data.text, providers.IndexOptions{
			Score:         1.0,
			MatchStrategy: providers.MatchNGram,
			NGramSize:     3,
		})
		if err != nil {
			t.Fatalf("Failed to index: %v", err)
		}
	}

	tests := []struct {
		query       string
		wantIDs     []string
		description string
	}{
		{
			query:       "boo",
			wantIDs:     []string{"1", "2", "3", "5"},
			description: "Exact 3-gram match",
		},
		{
			query:       "book",
			wantIDs:     []string{"1", "2", "3", "5"},
			description: "4-char query with sliding window",
		},
		{
			query:       "shelf",
			wantIDs:     []string{"1", "4"},
			description: "5-char query with sliding window",
		},
		{
			query:       "bookshelf",
			wantIDs:     []string{"1"},
			description: "Full word match through sliding window",
		},
		{
			query:       "kee",
			wantIDs:     []string{"5"},
			description: "Exact 3-gram in middle of word",
		},
		{
			query:       "keep",
			wantIDs:     []string{"5"},
			description: "4-char sliding window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			results, err := provider.Query(ctx, key, tt.query, providers.QueryOptions{
				MaxResults:    10,
				MatchStrategy: providers.MatchNGram,
				NGramSize:     3,
			})
			if err != nil {
				t.Errorf("Query failed: %v", err)
				return
			}
			if len(results) != len(tt.wantIDs) {
				t.Errorf("Query '%s': got %d results, want %d", tt.query, len(results), len(tt.wantIDs))
				t.Logf("Got IDs: %v", getResultIDs(results))
				return
			}
			resultIDs := make(map[string]bool)
			for _, r := range results {
				resultIDs[r.ID] = true
			}

			for _, wantID := range tt.wantIDs {
				if !resultIDs[wantID] {
					t.Errorf("Query '%s': missing expected ID %s", tt.query, wantID)
				}
			}
		})
	}
}

func getResultIDs(results []providers.ProviderResult) []string {
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

func TestRedisProvider_Delete(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey
	err := provider.Index(ctx, key, "1", "John Doe", "John Doe", providers.IndexOptions{
		Score:         1.0,
		MatchStrategy: providers.MatchSubstring,
	})
	if err != nil {
		t.Fatalf("Failed to index entry: %v", err)
	}
	results, err := provider.Query(ctx, key, "john", providers.QueryOptions{
		MaxResults:    10,
		MatchStrategy: providers.MatchSubstring,
	})
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	err = provider.Delete(ctx, key, "1")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	results, err = provider.Query(ctx, key, "john", providers.QueryOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("Failed to query after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results after delete, got %d", len(results))
	}
}

func TestRedisProvider_DeleteAll(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey
	entries := []struct {
		id   string
		text string
	}{
		{"1", "John Doe"},
		{"2", "Jane Smith"},
		{"3", "Bob Johnson"},
	}

	for _, e := range entries {
		err := provider.Index(ctx, key, e.id, e.text, e.text, providers.IndexOptions{
			Score:         1.0,
			MatchStrategy: providers.MatchSubstring,
		})
		if err != nil {
			t.Fatalf("Failed to index entry: %v", err)
		}
	}
	err := provider.DeleteAll(ctx, key)
	if err != nil {
		t.Errorf("DeleteAll() error = %v", err)
	}
	for _, query := range []string{"john", "jane", "bob"} {
		results, err := provider.Query(ctx, key, query, providers.QueryOptions{MaxResults: 10})
		if err != nil {
			t.Fatalf("Failed to query after DeleteAll: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for query %s after DeleteAll, got %d", query, len(results))
		}
	}
}

func TestRedisProvider_CaseSensitive(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey

	tests := []struct {
		name           string
		indexOptions   providers.IndexOptions
		queryOptions   providers.QueryOptions
		indexText      string
		queryText      string
		expectedResult bool
		description    string
	}{
		// Case-insensitive tests (default behavior)
		{
			name: "case-insensitive lowercase query matches mixed case",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchPrefix,
				CaseSensitive: false,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: false,
				MatchStrategy: providers.MatchPrefix,
			},
			indexText:      "Hello World",
			queryText:      "hello",
			expectedResult: true,
			description:    "Should match when case-insensitive",
		},
		{
			name: "case-insensitive uppercase query matches mixed case",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchPrefix,
				CaseSensitive: false,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: false,
				MatchStrategy: providers.MatchPrefix,
			},
			indexText:      "Hello World",
			queryText:      "HELLO",
			expectedResult: true,
			description:    "Should match when case-insensitive",
		},
		// Case-sensitive tests
		{
			name: "case-sensitive exact match",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchPrefix,
				CaseSensitive: true,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: true,
				MatchStrategy: providers.MatchPrefix,
			},
			indexText:      "Hello World",
			queryText:      "Hello",
			expectedResult: true,
			description:    "Should match exact case",
		},
		{
			name: "case-sensitive lowercase query doesn't match mixed case",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchPrefix,
				CaseSensitive: true,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: true,
				MatchStrategy: providers.MatchPrefix,
			},
			indexText:      "Hello World",
			queryText:      "hello",
			expectedResult: false,
			description:    "Should not match different case",
		},
		{
			name: "case-sensitive uppercase query doesn't match mixed case",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchPrefix,
				CaseSensitive: true,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: true,
				MatchStrategy: providers.MatchPrefix,
			},
			indexText:      "Hello World",
			queryText:      "HELLO",
			expectedResult: false,
			description:    "Should not match different case",
		},
		// Test with substring matching
		{
			name: "case-sensitive substring exact match",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchSubstring,
				CaseSensitive: true,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: true,
				MatchStrategy: providers.MatchSubstring,
			},
			indexText:      "Hello World",
			queryText:      "World",
			expectedResult: true,
			description:    "Should match exact case substring",
		},
		{
			name: "case-sensitive substring different case no match",
			indexOptions: providers.IndexOptions{
				Score:         1.0,
				MatchStrategy: providers.MatchSubstring,
				CaseSensitive: true,
			},
			queryOptions: providers.QueryOptions{
				MaxResults:    10,
				CaseSensitive: true,
				MatchStrategy: providers.MatchSubstring,
			},
			indexText:      "Hello World",
			queryText:      "world",
			expectedResult: false,
			description:    "Should not match different case substring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			err := provider.DeleteAll(ctx, key)
			if err != nil {
				t.Fatalf("Failed to clean up before test: %v", err)
			}

			// Index the text
			err = provider.Index(ctx, key, "test-id", tt.indexText, "Test Display", tt.indexOptions)
			if err != nil {
				t.Fatalf("Failed to index: %v", err)
			}

			// Query for the text
			results, err := provider.Query(ctx, key, tt.queryText, tt.queryOptions)
			if err != nil {
				t.Fatalf("Failed to query: %v", err)
			}

			gotResult := len(results) > 0
			if gotResult != tt.expectedResult {
				t.Errorf("%s: got %v, want %v", tt.description, gotResult, tt.expectedResult)
			}
		})
	}
}

func TestRedisProvider_CaseSensitiveDelete(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey

	// Test case-sensitive deletion
	t.Run("delete case-sensitive entry", func(t *testing.T) {
		// Index with case sensitivity
		err := provider.Index(ctx, key, "cs-id", "Hello World", "Display", providers.IndexOptions{
			Score:         1.0,
			MatchStrategy: providers.MatchPrefix,
			CaseSensitive: true,
		})
		if err != nil {
			t.Fatalf("Failed to index: %v", err)
		}

		// Delete the entry
		err = provider.Delete(ctx, key, "cs-id")
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		// Verify deletion with exact case query
		results, err := provider.Query(ctx, key, "Hello", providers.QueryOptions{
			MaxResults:    10,
			CaseSensitive: true,
			MatchStrategy: providers.MatchPrefix,
		})
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results after delete, got %d", len(results))
		}
	})

	// Test case-insensitive deletion
	t.Run("delete case-insensitive entry", func(t *testing.T) {
		// Clean up first
		err := provider.DeleteAll(ctx, key)
		if err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}

		// Index without case sensitivity
		err = provider.Index(ctx, key, "ci-id", "Hello World", "Display", providers.IndexOptions{
			Score:         1.0,
			MatchStrategy: providers.MatchPrefix,
			CaseSensitive: false,
		})
		if err != nil {
			t.Fatalf("Failed to index: %v", err)
		}

		// Delete the entry
		err = provider.Delete(ctx, key, "ci-id")
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		// Verify deletion with lowercase query
		results, err := provider.Query(ctx, key, "hello", providers.QueryOptions{
			MaxResults:    10,
			CaseSensitive: false,
			MatchStrategy: providers.MatchPrefix,
		})
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results after delete, got %d", len(results))
		}
	})
}

func TestRedisProvider_BackwardCompatibility(t *testing.T) {
	provider := getTestRedisClient(t)

	ctx := context.Background()
	key := testKey

	// Simulate old data indexed without case sensitivity metadata
	// This tests backward compatibility with existing indexed data
	t.Run("query old data without metadata", func(t *testing.T) {
		// Clean up first
		err := provider.DeleteAll(ctx, key)
		if err != nil {
			t.Fatalf("Failed to clean up: %v", err)
		}

		// Manually add data as if it was indexed with old version (no metadata)
		// This simulates data indexed before CaseSensitive option was added
		pipe := provider.client.Pipeline()

		// Add lowercase tokens (old behavior was always lowercase)
		id := "old-id"
		text := "hello world"
		display := "Hello World Display"

		// Simulate old indexing behavior (always lowercase)
		for i := 1; i <= len(text); i++ {
			prefix := text[:i]
			member := fmt.Sprintf("%s:%s", prefix, id)
			pipe.ZAdd(ctx, "ac:set:"+key, &redis.Z{
				Score:  1.0,
				Member: member,
			})
		}
		pipe.HSet(ctx, "ac:text:"+key, id, "Hello World") // Original text
		pipe.HSet(ctx, "ac:display:"+key, id, display)
		// Note: No metadata entry - simulating old data

		_, err = pipe.Exec(ctx)
		if err != nil {
			t.Fatalf("Failed to set up old data: %v", err)
		}

		// Query with case-insensitive (default behavior)
		results, err := provider.Query(ctx, key, "HELLO", providers.QueryOptions{
			MaxResults:    10,
			CaseSensitive: false,
			MatchStrategy: providers.MatchPrefix,
		})
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result for case-insensitive query on old data, got %d", len(results))
		}

		// Delete should work without metadata
		err = provider.Delete(ctx, key, id)
		if err != nil {
			t.Fatalf("Failed to delete old data: %v", err)
		}

		// Verify deletion
		results, err = provider.Query(ctx, key, "hello", providers.QueryOptions{
			MaxResults:    10,
			CaseSensitive: false,
			MatchStrategy: providers.MatchPrefix,
		})
		if err != nil {
			t.Fatalf("Failed to query after delete: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results after deleting old data, got %d", len(results))
		}
	})
}
