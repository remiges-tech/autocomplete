package main

import (
	"context"
	"fmt"
	"log"

	"github.com/remiges-tech/autocomplete"
	"github.com/remiges-tech/autocomplete/providers/redis"
	_ "github.com/remiges-tech/autocomplete/providers/redis"
)

func main() {
	redisConfig := redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	config := autocomplete.NewConfig(redisConfig)
	config.Options.DefaultLimit = 5
	config.Options.MinPrefixLength = 2
	config.Options.MatchStrategy = autocomplete.MatchSubstring

	ac, err := autocomplete.New("redis", config)
	if err != nil {
		log.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer ac.Close()

	ctx := context.Background()
	sampleData := []struct {
		id     string
		text   string
		source map[string]interface{}
	}{
		{"1", "Apple iPhone 14 Pro", map[string]interface{}{"category": "electronics", "price": 999}},
		{"2", "Apple MacBook Pro", map[string]interface{}{"category": "computers", "price": 2499}},
		{"3", "Apple Watch Series 8", map[string]interface{}{"category": "wearables", "price": 399}},
		{"4", "Samsung Galaxy S23", map[string]interface{}{"category": "electronics", "price": 899}},
		{"5", "Sony PlayStation 5", map[string]interface{}{"category": "gaming", "price": 499}},
		{"6", "Microsoft Surface Pro", map[string]interface{}{"category": "computers", "price": 1299}},
		{"7", "Amazon Echo Dot", map[string]interface{}{"category": "smart-home", "price": 49}},
		{"8", "Google Pixel 7 Pro", map[string]interface{}{"category": "electronics", "price": 899}},
	}

	fmt.Println("Indexing sample data...")
	for _, data := range sampleData {
		price := ""
		if p, ok := data.source["price"].(int); ok {
			price = fmt.Sprintf(" - $%d", p)
		}
		display := data.text + price
		err := ac.Index(ctx, data.id, data.text, display)
		if err != nil {
			log.Printf("Failed to index %s: %v", data.id, err)
		}
	}
	queries := []struct {
		term string
		desc string
	}{
		{"app", "Starting with 'app'"},
		{"pro", "Contains 'pro' anywhere"},
		{"watch", "Contains 'watch'"},
		{"phone", "Contains 'phone'"},
		{"surface", "Contains 'surface'"},
		{"5", "Contains '5'"},
	}

	for _, q := range queries {
		fmt.Printf("\nSearching for '%s' (%s):\n", q.term, q.desc)
		results, err := ac.Query(ctx, q.term, 5)
		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		if len(results) == 0 {
			fmt.Println("  No results found")
		} else {
			for _, result := range results {
				fmt.Printf("  - %s (ID: %s)\n", result.Display, result.ID)
			}
		}
	}
	fmt.Println("\nDeleting 'Apple iPhone 14 Pro' (ID: 1)...")
	err = ac.Delete(ctx, "1")
	if err != nil {
		log.Printf("Failed to delete: %v", err)
	}
	fmt.Println("\nSearching for 'apple' after deletion:")
	results, err := ac.Query(ctx, "apple", 5)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		for _, result := range results {
			fmt.Printf("  - %s (ID: %s)\n", result.Display, result.ID)
		}
	}
}
