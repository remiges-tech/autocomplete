package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/remiges/cvl-kra/autocomplete"
	"github.com/remiges/cvl-kra/autocomplete/providers/redis"
	_ "github.com/remiges/cvl-kra/autocomplete/providers/redis"
)

//nolint:cyclop // Test function with table-driven tests can have higher complexity
func TestIndianPostalCodeAutocomplete(t *testing.T) {
	ctx := context.Background()

	redisConfig := redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	config := autocomplete.NewConfig(redisConfig)
	config.Options.MatchStrategy = autocomplete.MatchNGram
	config.Options.NGramSize = 3
	config.Options.Namespace = "test_postal_codes"

	ac, err := autocomplete.New("redis", config)
	if err != nil {
		t.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer func() {
		if err := ac.Close(); err != nil {
			t.Errorf("Failed to close autocomplete: %v", err)
		}
	}()

	postalCodes := []PostalCode{
		{Pincode: "110001", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110002", City: "New Delhi", District: "North Delhi", State: "Delhi"},
		{Pincode: "400001", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400002", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "560001", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "600001", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "700001", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "500001", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
	}

	for i, pc := range postalCodes {
		searchableText := fmt.Sprintf("%s %s %s %s", pc.Pincode, pc.City, pc.District, pc.State)
		displayText := fmt.Sprintf("%s - %s, %s (%s)", pc.Pincode, pc.City, pc.District, pc.State)
		err := ac.Index(ctx, fmt.Sprintf("test_postal_%d", i), searchableText, displayText)
		if err != nil {
			t.Errorf("Failed to index %s: %v", pc.Pincode, err)
		}
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		validateFirst func(t *testing.T, result autocomplete.Result)
	}{
		{
			name:          "Search by PIN prefix - Delhi",
			query:         "110",
			expectedCount: 2,
			validateFirst: func(t *testing.T, result autocomplete.Result) {
				// Check that display contains Delhi
				if !strings.Contains(result.Display, "Delhi") {
					t.Errorf("Expected display to contain Delhi, got %s", result.Display)
				}
				// Check that display starts with 110
				if len(result.Display) < 3 || result.Display[:3] != "110" {
					t.Errorf("Expected display starting with 110, got %s", result.Display)
				}
			},
		},
		{
			name:          "Search by PIN prefix - Mumbai",
			query:         "400",
			expectedCount: 2,
			validateFirst: func(t *testing.T, result autocomplete.Result) {
				// Check that display contains Maharashtra and Mumbai
				if !strings.Contains(result.Display, "Maharashtra") {
					t.Errorf("Expected display to contain Maharashtra, got %s", result.Display)
				}
				if !strings.Contains(result.Display, "Mumbai") {
					t.Errorf("Expected display to contain Mumbai, got %s", result.Display)
				}
			},
		},
		{
			name:          "Search by city name with NGram",
			query:         "enn",
			expectedCount: 1,
			validateFirst: func(t *testing.T, result autocomplete.Result) {
				if !strings.Contains(result.Display, "Chennai") {
					t.Errorf("Expected display to contain Chennai, got %s", result.Display)
				}
			},
		},
		{
			name:          "Search by partial state name",
			query:         "arnat",
			expectedCount: 1,
			validateFirst: func(t *testing.T, result autocomplete.Result) {
				if !strings.Contains(result.Display, "Karnataka") {
					t.Errorf("Expected display to contain Karnataka, got %s", result.Display)
				}
			},
		},
		{
			name:          "Search by partial district",
			query:         "tral",
			expectedCount: 1,
			validateFirst: func(t *testing.T, result autocomplete.Result) {
				if !strings.Contains(result.Display, "Central Delhi") {
					t.Errorf("Expected display to contain Central Delhi, got %s", result.Display)
				}
			},
		},
		{
			name:          "Search with no results",
			query:         "xyz",
			expectedCount: 0,
			validateFirst: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ac.Query(ctx, tt.query, 10)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && tt.validateFirst != nil {
				tt.validateFirst(t, results[0])
			}
		})
	}
}

func TestNGramPartialMatching(t *testing.T) {
	ctx := context.Background()

	redisConfig := redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	config := autocomplete.NewConfig(redisConfig)
	config.Options.MatchStrategy = autocomplete.MatchNGram
	config.Options.NGramSize = 3
	config.Options.Namespace = "test_ngram_postal"

	ac, err := autocomplete.New("redis", config)
	if err != nil {
		t.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer func() {
		if err := ac.Close(); err != nil {
			t.Errorf("Failed to close autocomplete: %v", err)
		}
	}()

	postalCodes := []PostalCode{
		{Pincode: "411001", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "380001", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "302001", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
		{Pincode: "226001", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
	}

	for i, pc := range postalCodes {
		searchableText := fmt.Sprintf("%s %s %s %s", pc.Pincode, pc.City, pc.District, pc.State)
		displayText := fmt.Sprintf("%s - %s, %s (%s)", pc.Pincode, pc.City, pc.District, pc.State)
		err := ac.Index(ctx, fmt.Sprintf("test_ngram_%d", i), searchableText, displayText)
		if err != nil {
			t.Errorf("Failed to index %s: %v", pc.Pincode, err)
		}
	}

	tests := []struct {
		name            string
		query           string
		expectedCity    string
		expectedPincode string
	}{
		{
			name:            "NGram match - middle of pincode",
			query:           "1100",
			expectedCity:    "Pune",
			expectedPincode: "411001",
		},
		{
			name:            "NGram match - partial city",
			query:           "meda",
			expectedCity:    "Ahmedabad",
			expectedPincode: "380001",
		},
		{
			name:            "NGram match - partial state",
			query:           "jara",
			expectedCity:    "Ahmedabad",
			expectedPincode: "380001",
		},
		{
			name:            "NGram match - cross word boundary",
			query:           "ar Pra",
			expectedCity:    "Lucknow",
			expectedPincode: "226001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ac.Query(ctx, tt.query, 5)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if len(results) == 0 {
				t.Errorf("Expected at least one result for query: %s", tt.query)
				return
			}

			// Check that display contains expected city and pincode
			if !strings.Contains(results[0].Display, tt.expectedCity) {
				t.Errorf("Expected display to contain city %s, got %s", tt.expectedCity, results[0].Display)
			}
			if !strings.Contains(results[0].Display, tt.expectedPincode) {
				t.Errorf("Expected display to contain pincode %s, got %s", tt.expectedPincode, results[0].Display)
			}
		})
	}
}

func BenchmarkIndianPostalCodeNGramSearch(b *testing.B) {
	ctx := context.Background()

	redisConfig := redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	config := autocomplete.NewConfig(redisConfig)
	config.Options.MatchStrategy = autocomplete.MatchNGram
	config.Options.NGramSize = 3
	config.Options.Namespace = "bench_postal_ngram"

	ac, err := autocomplete.New("redis", config)
	if err != nil {
		b.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer func() {
		if err := ac.Close(); err != nil {
			b.Errorf("Failed to close autocomplete: %v", err)
		}
	}()

	for i := 0; i < 100; i++ {
		pincode := fmt.Sprintf("%06d", 100001+i)
		searchableText := fmt.Sprintf("%s Test City Test District Test State", pincode)
		displayText := fmt.Sprintf("%s - Test City, Test District (Test State)", pincode)
		_ = ac.Index(ctx, fmt.Sprintf("bench_%d", i), searchableText, displayText)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ac.Query(ctx, "100", 10)
	}
}
