// Package main demonstrates basic usage of the Elasticsearch autocomplete provider
// with Indian postal code data.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/remiges/cvl-kra/autocomplete"
	"github.com/remiges/cvl-kra/autocomplete/providers/elasticsearch"
	_ "github.com/remiges/cvl-kra/autocomplete/providers/elasticsearch" // Register provider
)

// PostalCode represents an Indian postal code with location information
type PostalCode struct {
	Pincode  string
	City     string
	District string
	State    string
}

func main() {
	ctx := context.Background()
	ac := setupAutocomplete()
	defer ac.Close()

	indexPostalCodes(ctx, ac)
	runSearchExamples(ctx, ac)
	runUpdateExample(ctx, ac)
	runDeleteExample(ctx, ac)
	runPerformanceTest(ctx, ac)
}

func setupAutocomplete() autocomplete.AutoComplete {
	// Get Elasticsearch URL from environment or use default
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	// Create Elasticsearch configuration
	esConfig := elasticsearch.Config{
		URLs:          []string{esURL},
		Index:         "postal_codes_basic",
		RefreshPolicy: "true", // Immediate visibility for demo
	}

	// Create autocomplete configuration
	config := autocomplete.NewConfig(esConfig)
	config.Options.MatchStrategy = autocomplete.MatchSubstring
	config.Options.Namespace = "postal_codes"

	// Create autocomplete instance
	ac, err := autocomplete.New("elasticsearch", config)
	if err != nil {
		log.Fatalf("Failed to create autocomplete: %v", err)
	}

	return ac
}

func indexPostalCodes(ctx context.Context, ac autocomplete.AutoComplete) {
	// Clear existing data
	fmt.Println("Clearing existing data...")
	if err := ac.DeleteAll(ctx); err != nil {
		log.Printf("Warning: failed to clear existing data: %v", err)
	}

	// Wait a moment for deletion to propagate
	time.Sleep(1 * time.Second)

	// Index sample postal codes
	fmt.Println("\nIndexing sample postal codes...")

	// Simple postal code entries
	postalCodes := []PostalCode{
		{Pincode: "110001", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110002", City: "New Delhi", District: "North Delhi", State: "Delhi"},
		{Pincode: "400001", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400002", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "560001", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "600001", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "700001", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "500001", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "380001", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "411001", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "226001", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
		{Pincode: "302001", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
	}

	// Index each postal code - combining all fields for searchability
	startTime := time.Now()
	indexed := 0
	for _, pc := range postalCodes {
		id := pc.Pincode
		// Combine all fields for searchability
		text := fmt.Sprintf("%s %s %s %s", pc.Pincode, pc.City, pc.District, pc.State)
		display := fmt.Sprintf("%s - %s, %s (%s)", pc.Pincode, pc.City, pc.District, pc.State)

		if err := ac.Index(ctx, id, text, display); err != nil {
			log.Printf("Failed to index %s: %v", id, err)
		} else {
			indexed++
		}
	}
	fmt.Printf("Successfully indexed %d/%d postal codes in %v\n", indexed, len(postalCodes), time.Since(startTime))

	// Wait a moment for indexing to complete
	fmt.Println("Waiting for Elasticsearch to refresh...")
	time.Sleep(2 * time.Second)
}

func runSearchExamples(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\n========== Search Examples ==========")

	// Search by PIN code prefix
	fmt.Println("\n1. Search by PIN code prefix:")
	searchQueries := []string{"110", "400", "560"}
	for _, query := range searchQueries {
		searchAndPrint(ctx, ac, query, 5)
	}

	// Search by city name
	fmt.Println("\n\n2. Search by city name:")
	cityQueries := []string{"delhi", "mumbai", "bangalore", "chennai"}
	for _, query := range cityQueries {
		searchAndPrint(ctx, ac, query, 3)
	}

	// Search by state
	fmt.Println("\n\n3. Search by state:")
	searchAndPrint(ctx, ac, "maharashtra", 10)

	// Partial search
	fmt.Println("\n\n4. Partial text search:")
	partialQueries := []string{"bang", "chen", "kol"}
	for _, query := range partialQueries {
		searchAndPrint(ctx, ac, query, 3)
	}
}

func searchAndPrint(ctx context.Context, ac autocomplete.AutoComplete, query string, limit int) {
	fmt.Printf("\nSearching for '%s':\n", query)
	results, err := ac.Query(ctx, query, limit)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("  No results found")
	} else {
		for _, result := range results {
			fmt.Printf("  - %s\n", result.Display)
		}
	}
}

func runUpdateExample(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\n\n========== Update Example ==========")
	fmt.Println("Updating Delhi postal code with more details...")
	updatedText := "110001 New Delhi Connaught Place Central Delhi Delhi NCR National Capital"
	updatedDisplay := "110001 - New Delhi (Connaught Place, Central Delhi) - National Capital Region"

	if err := ac.Index(ctx, "110001", updatedText, updatedDisplay); err != nil {
		log.Printf("Failed to update: %v", err)
	}

	// Query to see the update
	fmt.Println("\nSearching for 'connaught' after update:")
	searchAndPrint(ctx, ac, "connaught", 5)
}

func runDeleteExample(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\n\n========== Delete Example ==========")
	fmt.Println("Deleting Lucknow postal code...")
	if err := ac.Delete(ctx, "226001"); err != nil {
		log.Printf("Failed to delete: %v", err)
	}

	// Query to confirm deletion
	fmt.Println("\nSearching for 'lucknow' after deletion:")
	searchAndPrint(ctx, ac, "lucknow", 5)
}

func runPerformanceTest(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\n\n========== Performance Test ==========")
	fmt.Println("Testing search performance...")

	testQueries := []string{"110", "delhi", "mumbai", "400", "bangalore", "maharashtra"}
	for _, query := range testQueries {
		start := time.Now()
		_, err := ac.Query(ctx, query, 10)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("Query '%s' failed: %v", query, err)
		} else {
			fmt.Printf("Query '%s' completed in %v\n", query, elapsed)
		}
	}
}
