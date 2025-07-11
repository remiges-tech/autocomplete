package main

import (
	"context"
	"fmt"
	"log"

	"github.com/remiges/cvl-kra/autocomplete"
	"github.com/remiges/cvl-kra/autocomplete/providers/redis"
	_ "github.com/remiges/cvl-kra/autocomplete/providers/redis"
)

const (
	defaultNGramSize = 3
	maxQueryResults  = 10

	exampleTextLength     = 20
	prefixEntriesCount    = 20
	nGramEntriesCount     = 18
	substringEntriesCount = 210
)

type Product struct {
	id   string
	text string
}

type StrategyDemo struct {
	name        string
	strategy    autocomplete.MatchStrategy
	description string
	queries     []string
}

func main() {
	ctx := context.Background()
	redisConfig := createRedisConfig()
	products := getSampleProducts()
	strategies := getMatchingStrategies()

	demonstrateStrategies(ctx, redisConfig, products, strategies)
	displayStorageComparison()
}

func createRedisConfig() redis.Config {
	return redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}

func getSampleProducts() []Product {
	return []Product{
		{"1", "Apple iPhone 14 Pro"},
		{"2", "Samsung Galaxy Phone"},
		{"3", "Google Pixel Phone"},
		{"4", "MacBook Pro Laptop"},
		{"5", "Surface Pro Tablet"},
	}
}

func getMatchingStrategies() []StrategyDemo {
	return []StrategyDemo{
		{
			name:        "Prefix Matching",
			strategy:    autocomplete.MatchPrefix,
			description: "Traditional autocomplete - matches from the beginning only",
			queries:     []string{"app", "mac", "phone", "pro"},
		},
		{
			name:        "N-Gram Matching",
			strategy:    autocomplete.MatchNGram,
			description: "Fixed 3-character sequences",
			queries:     []string{"pho", "pro", "app", "gal", "mac"},
		},
		{
			name:        "Substring Matching",
			strategy:    autocomplete.MatchSubstring,
			description: "Matches any part of the text",
			queries:     []string{"phone", "pro", "book", "galaxy", "14"},
		},
	}
}

func demonstrateStrategies(ctx context.Context, redisConfig redis.Config, products []Product, strategies []StrategyDemo) {
	for _, strategy := range strategies {
		demonstrateStrategy(ctx, redisConfig, products, strategy)
	}
}

func demonstrateStrategy(ctx context.Context, redisConfig redis.Config, products []Product, strategy StrategyDemo) {
	printStrategyHeader(strategy)

	ac := createAutocompleteInstance(redisConfig, strategy)
	defer ac.Close()

	indexProducts(ctx, ac, products)
	runQueries(ctx, ac, strategy.queries)
}

func printStrategyHeader(strategy StrategyDemo) {
	fmt.Printf("\n========== %s ==========\n", strategy.name)
	fmt.Printf("Description: %s\n\n", strategy.description)
}

func createAutocompleteInstance(redisConfig redis.Config, strategy StrategyDemo) autocomplete.AutoComplete {
	config := autocomplete.NewConfig(redisConfig)
	config.Options.MatchStrategy = strategy.strategy
	config.Options.NGramSize = defaultNGramSize
	config.Options.Namespace = createNamespace(strategy.name)

	ac, err := autocomplete.New("redis", config)
	if err != nil {
		log.Fatalf("Failed to create autocomplete: %v", err)
	}

	return ac
}

func createNamespace(strategyName string) string {
	return fmt.Sprintf("demo_%s", strategyName)
}

func indexProducts(ctx context.Context, ac autocomplete.AutoComplete, products []Product) {
	fmt.Println("Indexing products...")
	for _, product := range products {
		err := ac.Index(ctx, product.id, product.text, product.text)
		if err != nil {
			log.Printf("Failed to index %s: %v", product.id, err)
		}
	}
}

func runQueries(ctx context.Context, ac autocomplete.AutoComplete, queries []string) {
	fmt.Println("\nQuery results:")
	for _, query := range queries {
		runSingleQuery(ctx, ac, query)
	}
}

func runSingleQuery(ctx context.Context, ac autocomplete.AutoComplete, query string) {
	results, err := ac.Query(ctx, query, maxQueryResults)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Printf("  '%s' -> ", query)
	printQueryResults(results)
}

func printQueryResults(results []autocomplete.Result) {
	if len(results) == 0 {
		fmt.Println("No matches")
		return
	}

	for i, result := range results {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(result.Display)
	}
	fmt.Println()
}

func displayStorageComparison() {
	fmt.Println("\n========== Storage Comparison ==========")
	fmt.Printf("For text 'Apple iPhone 14 Pro' (%d characters):\n", exampleTextLength)
	fmt.Printf("- Prefix:       %d entries (prefixes only)\n", prefixEntriesCount)
	fmt.Printf("- N-Gram (n=%d): %d entries (%d-char sequences)\n", defaultNGramSize, nGramEntriesCount, defaultNGramSize)
	fmt.Printf("- Substring:    %d entries (all substrings)\n", substringEntriesCount)
	fmt.Println("\nChoose based on your needs:")
	fmt.Println("- Prefix: Traditional autocomplete, lowest storage")
	fmt.Println("- N-Gram: Good for typo tolerance, balanced storage")
	fmt.Println("- Substring: Maximum flexibility, highest storage")
}
