package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/remiges/cvl-kra/autocomplete"
	"github.com/remiges/cvl-kra/autocomplete/providers/redis"
	_ "github.com/remiges/cvl-kra/autocomplete/providers/redis"
)

const (
	defaultSearchLimit        = 5
	interactiveSearchLimit    = 10
	demoDelayDuration         = 2 * time.Second
	separatorLineLength       = 70
	totalPostalCodesInDataset = 80
)

// PostalCode represents an Indian postal code with location information
type PostalCode struct {
	Pincode  string `json:"pincode"`
	City     string `json:"city"`
	District string `json:"district"`
	State    string `json:"state"`
}

func main() {
	ctx := context.Background()

	fmt.Println("Indian Postal Code Autocomplete Example")
	fmt.Println("======================================")
	fmt.Println()

	ac := createAutocomplete()
	defer func() {
		if err := ac.Close(); err != nil {
			log.Printf("Failed to close autocomplete: %v", err)
		}
	}()
	indexSampleData(ctx, ac)
	searchExample(ctx, ac, "bangalore")
	searchExample(ctx, ac, "400")
	searchExample(ctx, ac, "pun")

	fmt.Println("\n\nWant to see more? Running full demo...")
	time.Sleep(demoDelayDuration)

	runFullDemo(ctx, ac)
}

func createAutocomplete() autocomplete.AutoComplete {
	redisConfig := createRedisConfiguration()
	config := createAutocompleteConfiguration(redisConfig)

	ac, err := autocomplete.New("redis", config)
	if err != nil {
		log.Fatalf("Failed to create autocomplete: %v", err)
	}

	fmt.Println("[OK] Step 1: Autocomplete instance created")
	return ac
}

func createRedisConfiguration() redis.Config {
	return redis.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
}

func createAutocompleteConfiguration(redisConfig redis.Config) autocomplete.Config {
	config := autocomplete.NewConfig(redisConfig)
	config.Options.MatchStrategy = autocomplete.MatchSubstring
	config.Options.Namespace = "indian_postal_codes"
	return config
}

func indexSampleData(ctx context.Context, ac autocomplete.AutoComplete) {
	sampleCodes := getSamplePostalCodes()

	for _, pc := range sampleCodes {
		id := pc.Pincode
		displayText := createDisplayText(pc)

		// Index each field separately with the same ID
		fields := []string{pc.Pincode, pc.City, pc.District, pc.State}
		for _, field := range fields {
			err := ac.Index(ctx, id, field, displayText)
			if err != nil {
				log.Printf("Failed to index field '%s' for %s: %v", field, pc.Pincode, err)
			}
		}
	}

	fmt.Printf("[OK] Step 2: Indexed %d sample postal codes\n", len(sampleCodes))
}

func getSamplePostalCodes() []PostalCode {
	return []PostalCode{
		{Pincode: "560001", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "400001", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "411001", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "110001", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "600001", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
	}
}

func createDisplayText(pc PostalCode) string {
	return fmt.Sprintf("%s - %s, %s (%s)", pc.Pincode, pc.City, pc.District, pc.State)
}

func searchExample(ctx context.Context, ac autocomplete.AutoComplete, query string) {
	results, err := ac.Query(ctx, query, defaultSearchLimit)
	if err != nil {
		log.Printf("Search error: %v", err)
		return
	}

	fmt.Printf("\n[OK] Step 3: Search for '%s' - Found %d results:\n", query, len(results))
	printSearchResults(results)
}

func printSearchResults(results []autocomplete.Result) {
	for _, result := range results {
		fmt.Printf("   %s\n", result.Display)
	}
}

func runFullDemo(ctx context.Context, ac autocomplete.AutoComplete) {
	indexFullDataset(ctx, ac)
	demonstrateSearches(ctx, ac)
	showStatistics()
	runInteractiveMode(ctx, ac)
}

func indexFullDataset(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\nIndexing complete dataset...")

	if err := ac.DeleteAll(ctx); err != nil {
		log.Printf("Warning: failed to clear existing data: %v", err)
	}

	postalCodes := getFullPostalCodeDataset()
	startTime := time.Now()

	for _, pc := range postalCodes {
		id := pc.Pincode
		displayText := createDisplayText(pc)

		// Index each field separately with the same ID
		fields := []string{pc.Pincode, pc.City, pc.District, pc.State}
		for _, field := range fields {
			err := ac.Index(ctx, id, field, displayText)
			if err != nil {
				log.Printf("Failed to index field '%s' for %s: %v", field, pc.Pincode, err)
			}
		}
	}

	fmt.Printf("Indexed %d postal codes in %v\n", len(postalCodes), time.Since(startTime))
}

func demonstrateSearches(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\n========== Example Searches ==========")

	queries := []struct {
		query string
		desc  string
	}{
		{"110", "Delhi postal codes starting with 110"},
		{"400", "Mumbai postal codes starting with 400"},
		{"bangalore", "Search by city name"},
		{"maharashtra", "Search by state name"},
		{"central delhi", "Search by district name"},
		{"ahm", "Partial city match (Ahmedabad)"},
		{"pun", "Partial city match (Pune)"},
		{"002", "PIN codes containing 002"},
	}

	for _, q := range queries {
		fmt.Printf("\nSearching: %s (%s)\n", q.query, q.desc)
		fmt.Println(strings.Repeat("-", separatorLineLength))

		start := time.Now()
		results, err := ac.Query(ctx, q.query, defaultSearchLimit)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("Search error: %v", err)
			continue
		}

		fmt.Printf("Found %d results in %v:\n", len(results), elapsed)

		for i, result := range results {
			fmt.Printf("%d. %s\n", i+1, result.Display)
		}

		if len(results) == 0 {
			fmt.Println("No results found")
		}
	}
}

func showStatistics() {
	fmt.Printf("\n========== Statistics ==========\n")
	fmt.Printf("Total postal codes indexed: %d\n", totalPostalCodesInDataset)
	fmt.Printf("Storage provider: Redis\n")
	fmt.Printf("Search strategy: Substring\n")
	fmt.Printf("\nSubstring matching finds exact partial matches anywhere in the text.\n")
	fmt.Printf("For example: 'pun' matches 'Pune', '001' matches any text containing '001' as a substring\n")
}

func runInteractiveMode(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\n========== Interactive Search ==========")
	fmt.Println("Enter search queries (type 'quit' to exit):")
	fmt.Println("Try: city names, PIN codes, state names, or partial matches")
	fmt.Println()

	for {
		fmt.Print("Search: ")
		var query string
		if _, err := fmt.Scanln(&query); err != nil {
			log.Printf("Error reading input: %v", err)
			continue
		}

		if isExitCommand(query) {
			fmt.Println("Goodbye!")
			break
		}

		if query == "" {
			continue
		}

		start := time.Now()
		results, err := ac.Query(ctx, query, interactiveSearchLimit)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("Search error: %v", err)
			continue
		}

		fmt.Printf("\nFound %d results in %v:\n", len(results), elapsed)

		if len(results) == 0 {
			fmt.Println("No matches found. Try a different query.")
		} else {
			displayResults(results)
		}
		fmt.Println()
	}
}

func isExitCommand(command string) bool {
	return command == "quit" || command == "exit"
}

func displayResults(results []autocomplete.Result) {
	fmt.Println("\n--- Results ---")

	for _, result := range results {
		fmt.Println(result.Display)
	}

	if hasResults(results) {
		displayFirstResultDetails(results[0])
	}
}

func hasResults(results []autocomplete.Result) bool {
	return len(results) > 0
}

func displayFirstResultDetails(result autocomplete.Result) {
	fmt.Printf("\n--- First result details ---\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Display: %s\n", result.Display)
	fmt.Printf("Score: %.2f\n", result.Score)
}

func getFullPostalCodeDataset() []PostalCode {
	return []PostalCode{
		{Pincode: "110001", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110002", City: "New Delhi", District: "North Delhi", State: "Delhi"},
		{Pincode: "110003", City: "New Delhi", District: "North Delhi", State: "Delhi"},
		{Pincode: "110005", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110006", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110007", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110008", City: "New Delhi", District: "Central Delhi", State: "Delhi"},
		{Pincode: "110009", City: "New Delhi", District: "North Delhi", State: "Delhi"},
		{Pincode: "110011", City: "New Delhi", District: "New Delhi", State: "Delhi"},
		{Pincode: "110012", City: "New Delhi", District: "South Delhi", State: "Delhi"},
		{Pincode: "400001", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400002", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400003", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400004", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400005", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400006", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400007", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400008", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400009", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "400010", City: "Mumbai", District: "Mumbai City", State: "Maharashtra"},
		{Pincode: "560001", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560002", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560003", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560004", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560005", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560006", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560007", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560008", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560009", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "560010", City: "Bangalore", District: "Bangalore Urban", State: "Karnataka"},
		{Pincode: "600001", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600002", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600003", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600004", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600005", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600006", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600007", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600008", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600009", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "600010", City: "Chennai", District: "Chennai", State: "Tamil Nadu"},
		{Pincode: "700001", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700002", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700003", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700004", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700005", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700006", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700007", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700008", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700009", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "700010", City: "Kolkata", District: "Kolkata", State: "West Bengal"},
		{Pincode: "500001", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500002", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500003", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500004", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500005", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500006", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500007", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500008", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500009", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "500010", City: "Hyderabad", District: "Hyderabad", State: "Telangana"},
		{Pincode: "380001", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "380002", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "380003", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "380004", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "380005", City: "Ahmedabad", District: "Ahmedabad", State: "Gujarat"},
		{Pincode: "411001", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "411002", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "411003", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "411004", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "411005", City: "Pune", District: "Pune", State: "Maharashtra"},
		{Pincode: "226001", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
		{Pincode: "226002", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
		{Pincode: "226003", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
		{Pincode: "226004", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
		{Pincode: "226005", City: "Lucknow", District: "Lucknow", State: "Uttar Pradesh"},
		{Pincode: "302001", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
		{Pincode: "302002", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
		{Pincode: "302003", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
		{Pincode: "302004", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
		{Pincode: "302005", City: "Jaipur", District: "Jaipur", State: "Rajasthan"},
	}
}
