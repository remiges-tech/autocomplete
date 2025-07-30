// Package main demonstrates advanced usage of the Elasticsearch autocomplete provider
// with structured data for Indian postal codes using carefully formatted text.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/remiges-tech/autocomplete"
	"github.com/remiges-tech/autocomplete/providers/elasticsearch"
	_ "github.com/remiges-tech/autocomplete/providers/elasticsearch" // Register provider
)

type postalCodeEntry struct {
	id      string
	text    string
	display string
}

func main() {
	// Get Elasticsearch URL from environment or use default
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	// Create Elasticsearch configuration
	esConfig := elasticsearch.Config{
		URLs:          []string{esURL},
		Index:         "indian_postal_codes_advanced",
		RefreshPolicy: "wait_for", // Wait for indexing to complete
	}

	// Create autocomplete configuration
	config := autocomplete.NewConfig(esConfig)
	config.Options.MatchStrategy = autocomplete.MatchSubstring
	config.Options.Namespace = "postal_codes_structured"

	// Create autocomplete instance
	ac, err := autocomplete.New("elasticsearch", config)
	if err != nil {
		log.Fatalf("Failed to create autocomplete: %v", err)
	}
	defer ac.Close()

	ctx := context.Background()

	// Clear existing data
	fmt.Println("Clearing existing data...")
	if err := ac.DeleteAll(ctx); err != nil {
		log.Printf("Warning: failed to clear existing data: %v", err)
	}

	// Index postal codes with structured text
	fmt.Println("\nIndexing Indian postal codes with structured text...")

	postalCodes := getPostalCodeDataset()
	startTime := time.Now()

	for _, pc := range postalCodes {
		if err := ac.Index(ctx, pc.id, pc.text, pc.display); err != nil {
			log.Printf("Failed to index %s: %v", pc.id, err)
		}
	}

	fmt.Printf("Indexed %d postal codes in %v\n", len(postalCodes), time.Since(startTime))

	// Demonstrate various search scenarios
	fmt.Println("\n========== Search Examples ==========")

	// Search by pincode
	fmt.Println("\n1. Search by pincode prefix '110':")
	searchAndDisplay(ctx, ac, "110", 5)

	// Search by city name
	fmt.Println("\n2. Search by city 'Mumbai':")
	searchAndDisplay(ctx, ac, "Mumbai", 5)

	// Search by state
	fmt.Println("\n3. Search by state 'Karnataka':")
	searchAndDisplay(ctx, ac, "Karnataka", 5)

	// Search by district
	fmt.Println("\n4. Search by district 'Pune':")
	searchAndDisplay(ctx, ac, "Pune", 5)

	// Search by landmark or area
	fmt.Println("\n5. Search by landmark 'Gateway':")
	searchAndDisplay(ctx, ac, "Gateway", 5)

	// Search by region
	fmt.Println("\n6. Search by region 'Western':")
	searchAndDisplay(ctx, ac, "Western", 3)

	// Multi-word search
	fmt.Println("\n7. Multi-word search 'New Delhi':")
	searchAndDisplay(ctx, ac, "New Delhi", 5)

	// Partial word search
	fmt.Println("\n8. Partial search 'Bang' (for Bangalore):")
	searchAndDisplay(ctx, ac, "Bang", 5)

	// Case insensitive search
	fmt.Println("\n9. Case insensitive search 'chennai':")
	searchAndDisplay(ctx, ac, "chennai", 5)

	// Interactive search demo
	fmt.Println("\n========== Interactive Search ==========")
	fmt.Println("Try searching for postal codes. Type 'exit' to quit.")
	interactiveSearch(ctx, ac)
}

func getPostalCodeDataset() []postalCodeEntry {
	// Create structured text that includes all searchable information
	return []postalCodeEntry{
		// Delhi/NCR Region
		{
			id:      "110001",
			text:    "110001 New Delhi Connaught Place Delhi NCR North India Central Business District CBD Rajiv Chowk Metro",
			display: "110001 - Connaught Place, New Delhi - Delhi (NCR)",
		},
		{
			id:      "110002",
			text:    "110002 New Delhi Indraprastha Estate Delhi NCR North India IP Estate Supreme Court",
			display: "110002 - Indraprastha Estate, New Delhi - Delhi (NCR)",
		},
		{
			id:      "110003",
			text:    "110003 New Delhi Pandara Road Delhi NCR North India Embassy Area Diplomatic Enclave",
			display: "110003 - Pandara Road, New Delhi - Delhi (NCR)",
		},
		{
			id:      "201301",
			text:    "201301 Noida Sector 1 Gautam Buddha Nagar Uttar Pradesh NCR Tech Hub IT Park",
			display: "201301 - Noida Sector 1 - Gautam Buddha Nagar, UP (NCR)",
		},
		{
			id:      "122001",
			text:    "122001 Gurgaon Gurugram Haryana NCR Cyber City Millennium City DLF",
			display: "122001 - Gurgaon - Haryana (NCR)",
		},

		// Mumbai Region
		{
			id:      "400001",
			text:    "400001 Mumbai Fort General Post Office Maharashtra Western India Gateway of India Bombay Stock Exchange BSE",
			display: "400001 - Fort, Mumbai - Maharashtra (Western)",
		},
		{
			id:      "400002",
			text:    "400002 Mumbai Kalbadevi Maharashtra Western India Crawford Market Zaveri Bazaar Jewelry Market",
			display: "400002 - Kalbadevi, Mumbai - Maharashtra (Western)",
		},
		{
			id:      "400003",
			text:    "400003 Mumbai Mandvi Maharashtra Western India Cotton Exchange Textile Market Port Area",
			display: "400003 - Mandvi, Mumbai - Maharashtra (Western)",
		},
		{
			id:      "400004",
			text:    "400004 Mumbai Girgaon Charni Road Maharashtra Western India Chowpatty Beach Marine Drive",
			display: "400004 - Girgaon, Mumbai - Maharashtra (Western)",
		},
		{
			id:      "400005",
			text:    "400005 Mumbai Colaba Maharashtra Western India Gateway of India Taj Hotel Navy Nagar",
			display: "400005 - Colaba, Mumbai - Maharashtra (Western)",
		},

		// Bangalore Region
		{
			id:      "560001",
			text:    "560001 Bangalore Bengaluru General Post Office Karnataka South India Silicon Valley IT Capital MG Road",
			display: "560001 - GPO, Bangalore - Karnataka (South)",
		},
		{
			id:      "560002",
			text:    "560002 Bangalore Bengaluru Malleswaram Karnataka South India Residential Area Traditional Market",
			display: "560002 - Malleswaram, Bangalore - Karnataka (South)",
		},
		{
			id:      "560003",
			text:    "560003 Bangalore Bengaluru Rajajinagar Karnataka South India Industrial Area West Bangalore",
			display: "560003 - Rajajinagar, Bangalore - Karnataka (South)",
		},

		// Chennai Region
		{
			id:      "600001",
			text:    "600001 Chennai Madras General Post Office Tamil Nadu South India Marina Beach Central Station",
			display: "600001 - GPO, Chennai - Tamil Nadu (South)",
		},
		{
			id:      "600002",
			text:    "600002 Chennai Madras Parrys Corner Tamil Nadu South India Commercial Hub George Town",
			display: "600002 - Parrys, Chennai - Tamil Nadu (South)",
		},

		// Kolkata Region
		{
			id:      "700001",
			text:    "700001 Kolkata Calcutta General Post Office West Bengal Eastern India Dalhousie BBD Bagh",
			display: "700001 - GPO, Kolkata - West Bengal (East)",
		},
		{
			id:      "700002",
			text:    "700002 Kolkata Calcutta Burtolla West Bengal Eastern India North Kolkata Old City",
			display: "700002 - Burtolla, Kolkata - West Bengal (East)",
		},

		// Pune Region
		{
			id:      "411001",
			text:    "411001 Pune Maharashtra Western India Camp Area Cantonment MG Road East Pune",
			display: "411001 - Pune Camp - Maharashtra (Western)",
		},
		{
			id:      "411002",
			text:    "411002 Pune Maharashtra Western India City Central Laxmi Road Tulsi Baug",
			display: "411002 - Pune City - Maharashtra (Western)",
		},

		// Hyderabad Region
		{
			id:      "500001",
			text:    "500001 Hyderabad Telangana South India General Post Office Abids Commercial Center",
			display: "500001 - GPO, Hyderabad - Telangana (South)",
		},
		{
			id:      "500002",
			text:    "500002 Hyderabad Telangana South India Malakpet Old City Area Residential",
			display: "500002 - Malakpet, Hyderabad - Telangana (South)",
		},

		// Ahmedabad Region
		{
			id:      "380001",
			text:    "380001 Ahmedabad Gujarat Western India Gandhi Road Ashram Road Commercial Hub",
			display: "380001 - Gandhi Road, Ahmedabad - Gujarat (Western)",
		},

		// Jaipur Region
		{
			id:      "302001",
			text:    "302001 Jaipur Rajasthan North India Pink City Amer Road Palace Area Tourist Hub",
			display: "302001 - Amer Road, Jaipur - Rajasthan (North)",
		},

		// Kerala Region
		{
			id:      "682001",
			text:    "682001 Kochi Cochin Ernakulam Kerala South India Fort Kochi Colonial Area Spice Market",
			display: "682001 - Fort Kochi - Kerala (South)",
		},
		{
			id:      "695001",
			text:    "695001 Thiruvananthapuram Trivandrum Kerala South India General Post Office Capital City",
			display: "695001 - GPO, Thiruvananthapuram - Kerala (South)",
		},
	}
}

func searchAndDisplay(ctx context.Context, ac autocomplete.AutoComplete, query string, limit int) {
	results, err := ac.Query(ctx, query, limit)
	if err != nil {
		log.Printf("Search error: %v", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("  No results found")
		return
	}

	for i, result := range results {
		fmt.Printf("  %d. %s (Score: %.2f)\n", i+1, result.Display, result.Score)
	}
}

func interactiveSearch(ctx context.Context, ac autocomplete.AutoComplete) {
	fmt.Println("\nEnter search terms (or 'exit' to quit):")

	for {
		fmt.Print("\n> ")
		var query string
		fmt.Scanln(&query)

		if strings.ToLower(query) == "exit" {
			break
		}

		if query == "" {
			continue
		}

		// Perform search
		startTime := time.Now()
		results, err := ac.Query(ctx, query, 10)
		searchTime := time.Since(startTime)

		if err != nil {
			fmt.Printf("Search error: %v\n", err)
			continue
		}

		// Display results
		fmt.Printf("\nFound %d results in %v:\n", len(results), searchTime)

		if len(results) == 0 {
			fmt.Println("No matches found. Try different search terms.")
			continue
		}

		for i, result := range results {
			fmt.Printf("%2d. %s\n", i+1, result.Display)

			// Show what matched (first 100 chars of text)
			if len(results) <= 5 {
				fmt.Printf("    Relevance: %.2f\n", result.Score)
			}
		}

		// Show search tips
		if len(results) < 3 {
			fmt.Println("\nTip: Try searching for city names, pincodes, landmarks, or regions.")
		}
	}

	fmt.Println("\nThank you for using the postal code search!")
}
