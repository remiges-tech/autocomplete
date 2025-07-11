package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/remiges/cvl-kra/autocomplete"
	"github.com/remiges/cvl-kra/autocomplete/providers/elasticsearch"
	_ "github.com/remiges/cvl-kra/autocomplete/providers/elasticsearch"
)

func main() {
	ctx := context.Background()

	// Get Elasticsearch URL from environment or use default
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	// Create Elasticsearch configuration
	esConfig := elasticsearch.Config{
		URLs:          []string{esURL},
		Index:         "test_autocomplete",
		RefreshPolicy: "true", // Immediate visibility
	}

	// Create autocomplete configuration
	config := autocomplete.NewConfig(esConfig)
	config.Options.Namespace = "test"

	// Create autocomplete instance
	ac, err := autocomplete.New("elasticsearch", config)
	if err != nil {
		log.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer ac.Close()

	// Clear any existing data
	fmt.Println("Clearing existing data...")
	if err := ac.DeleteAll(ctx); err != nil {
		fmt.Printf("Warning: failed to clear data: %v\n", err)
	}

	// Index a simple entry
	fmt.Println("\nIndexing test entry...")
	err = ac.Index(ctx, "1", "hello world", "Hello World Display")
	if err != nil {
		log.Fatalf("Failed to index: %v", err)
	}
	fmt.Println("Indexed successfully!")

	// Search for it
	fmt.Println("\nSearching for 'hello'...")
	results, err := ac.Query(ctx, "hello", 10)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("Found %d results:\n", len(results))
	for _, r := range results {
		fmt.Printf("  - ID: %s, Display: %s, Score: %f\n", r.ID, r.Display, r.Score)
	}

	// Search for partial match
	fmt.Println("\nSearching for 'wor'...")
	results, err = ac.Query(ctx, "wor", 10)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	fmt.Printf("Found %d results:\n", len(results))
	for _, r := range results {
		fmt.Printf("  - ID: %s, Display: %s, Score: %f\n", r.ID, r.Display, r.Score)
	}

	// Verify the data in Elasticsearch directly
	fmt.Println("\n\nTo debug, run:")
	fmt.Printf("curl -s '%s/test_autocomplete/_search?pretty' | jq '.hits.hits'\n", esURL)
}
