package autocomplete

import (
	"context"
	"strings"
	"testing"

	"github.com/remiges-tech/autocomplete/providers"
)

// mockProvider is an in-memory provider for testing.
type mockProvider struct {
	data map[string]map[string]*mockEntry
}

type mockEntry struct {
	text          string
	result        *providers.ProviderResult
	caseSensitive bool
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		data: make(map[string]map[string]*mockEntry),
	}
}

func (m *mockProvider) Index(ctx context.Context, key, id, text, display string, options providers.IndexOptions) error {
	if m.data[key] == nil {
		m.data[key] = make(map[string]*mockEntry)
	}
	indexText := text
	if !options.CaseSensitive {
		indexText = strings.ToLower(text)
	}
	m.data[key][id] = &mockEntry{
		text: indexText,
		result: &providers.ProviderResult{
			ID:      id,
			Display: display,
			Score:   options.Score,
		},
		caseSensitive: options.CaseSensitive,
	}
	return nil
}

func (m *mockProvider) Query(ctx context.Context, key, query string, options providers.QueryOptions) ([]providers.ProviderResult, error) {
	var results []providers.ProviderResult
	if keyData, exists := m.data[key]; exists {
		searchQuery := query
		if !options.CaseSensitive {
			searchQuery = strings.ToLower(query)
		}
		for _, entry := range keyData {
			if len(entry.text) >= len(searchQuery) && entry.text[:len(searchQuery)] == searchQuery {
				results = append(results, *entry.result)
				if len(results) >= options.MaxResults {
					break
				}
			}
		}
	}
	return results, nil
}

func (m *mockProvider) Delete(ctx context.Context, key, id string) error {
	if keyData, exists := m.data[key]; exists {
		delete(keyData, id)
	}
	return nil
}

func (m *mockProvider) DeleteAll(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockProvider) Close() error {
	return nil
}

//nolint:cyclop // Test function with table-driven tests can have higher complexity
func TestAutoComplete(t *testing.T) {
	// Register mock provider
	RegisterProvider("mock", func(config interface{}) (providers.Provider, error) {
		return newMockProvider(), nil
	})

	config := NewConfig(nil)
	ac, err := New("mock", config)
	if err != nil {
		t.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer func() {
		if closeErr := ac.Close(); closeErr != nil {
			t.Errorf("Failed to close autocomplete: %v", closeErr)
		}
	}()

	ctx := context.Background()

	// Test Index
	err = ac.Index(ctx, "1", "Hello World", "Hello World - Display")
	if err != nil {
		t.Errorf("Index() error = %v", err)
	}

	// Test Index with empty ID
	err = ac.Index(ctx, "", "Text", "Display")
	if err != ErrEmptyID {
		t.Errorf("Index() with empty ID error = %v, want %v", err, ErrEmptyID)
	}

	// Test Index with empty text
	err = ac.Index(ctx, "2", "", "Display")
	if err != ErrEmptyText {
		t.Errorf("Index() with empty text error = %v, want %v", err, ErrEmptyText)
	}

	// Test Index with empty display
	err = ac.Index(ctx, "3", "Text", "")
	if err != ErrEmptyDisplay {
		t.Errorf("Index() with empty display error = %v, want %v", err, ErrEmptyDisplay)
	}

	// Test Query
	results, err := ac.Query(ctx, "Hello", 10)
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Query() returned %d results, want 1", len(results))
	}
	if results[0].ID != "1" || results[0].Display != "Hello World - Display" {
		t.Errorf("Query() returned unexpected result: %+v", results[0])
	}

	// Test Query with short prefix
	_, err = ac.Query(ctx, "", 10)
	if err != ErrQueryTooShort {
		t.Errorf("Query() with short query error = %v, want %v", err, ErrQueryTooShort)
	}

	// Test Query with exceeded limit
	_, err = ac.Query(ctx, "Hello", 1000)
	if err != ErrLimitExceeded {
		t.Errorf("Query() with exceeded limit error = %v, want %v", err, ErrLimitExceeded)
	}

	// Test Delete
	err = ac.Delete(ctx, "1")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deletion
	results, err = ac.Query(ctx, "Hello", 10)
	if err != nil {
		t.Errorf("Query() after delete error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Query() after delete returned %d results, want 0", len(results))
	}

	// Test Delete with empty ID
	err = ac.Delete(ctx, "")
	if err != ErrEmptyID {
		t.Errorf("Delete() with empty ID error = %v, want %v", err, ErrEmptyID)
	}
}

func TestProviderRegistration(t *testing.T) {
	// Test unregistered provider
	_, err := New("nonexistent", NewConfig(nil))
	if err == nil {
		t.Error("New() with unregistered provider should return error")
	}
}

func TestCaseSensitive(t *testing.T) {
	// Register mock provider
	RegisterProvider("mock-case", func(config interface{}) (providers.Provider, error) {
		return newMockProvider(), nil
	})

	tests := []struct {
		name          string
		caseSensitive bool
		indexText     string
		queryText     string
		expectMatch   bool
	}{
		{
			name:          "case-insensitive: lowercase query matches mixed case",
			caseSensitive: false,
			indexText:     "Hello World",
			queryText:     "hello",
			expectMatch:   true,
		},
		{
			name:          "case-insensitive: uppercase query matches mixed case",
			caseSensitive: false,
			indexText:     "Hello World",
			queryText:     "HELLO",
			expectMatch:   true,
		},
		{
			name:          "case-insensitive: mixed case query matches mixed case",
			caseSensitive: false,
			indexText:     "Hello World",
			queryText:     "HeLLo",
			expectMatch:   true,
		},
		{
			name:          "case-sensitive: exact match",
			caseSensitive: true,
			indexText:     "Hello World",
			queryText:     "Hello",
			expectMatch:   true,
		},
		{
			name:          "case-sensitive: lowercase query doesn't match mixed case",
			caseSensitive: true,
			indexText:     "Hello World",
			queryText:     "hello",
			expectMatch:   false,
		},
		{
			name:          "case-sensitive: uppercase query doesn't match mixed case",
			caseSensitive: true,
			indexText:     "Hello World",
			queryText:     "HELLO",
			expectMatch:   false,
		},
		{
			name:          "case-sensitive: different case doesn't match",
			caseSensitive: true,
			indexText:     "Hello World",
			queryText:     "HeLLo",
			expectMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig(nil)
			config.Options.CaseSensitive = tt.caseSensitive

			ac, err := New("mock-case", config)
			if err != nil {
				t.Fatalf("Failed to create autocomplete: %v", err)
			}
			defer func() {
				if closeErr := ac.Close(); closeErr != nil {
					t.Errorf("Failed to close autocomplete: %v", closeErr)
				}
			}()

			ctx := context.Background()

			// Index the text
			err = ac.Index(ctx, "test-id", tt.indexText, "Test Display")
			if err != nil {
				t.Fatalf("Index() error = %v", err)
			}

			// Query for the text
			results, err := ac.Query(ctx, tt.queryText, 10)
			if err != nil {
				t.Fatalf("Query() error = %v", err)
			}

			gotMatch := len(results) > 0
			if gotMatch != tt.expectMatch {
				t.Errorf("Query() match = %v, want %v", gotMatch, tt.expectMatch)
			}

			// Clean up
			err = ac.DeleteAll(ctx)
			if err != nil {
				t.Errorf("DeleteAll() error = %v", err)
			}
		})
	}
}
