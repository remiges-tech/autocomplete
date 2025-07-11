# Indian Postal Code Autocomplete Example

This example demonstrates how to use the autocomplete library for Indian postal codes (PIN codes) with location source data including state, city, and district information.

## Features

- **Indian PIN codes**: 6-digit postal codes from major Indian cities
- **Rich source data**: Each postal code includes city, district, and state information
- **NGram matching**: Uses 3-character sequences for flexible partial matching
- **Two implementations**: Basic and advanced Redis-based examples

## Data Structure

```go
type PostalCode struct {
    Pincode  string `json:"pincode"`
    City     string `json:"city"`
    District string `json:"district"`
    State    string `json:"state"`
}
```

## Running the Example

### Option 1: Using Docker Compose (Recommended)

```bash
# Start Redis
docker-compose up -d

# Run the example
go run main.go

# When done, stop Redis
docker-compose down
```

### Option 2: Using Docker Run

```bash
# Start Redis
docker run -d -p 6379:6379 --name autocomplete-redis redis:7-alpine

# Run the example
go run main.go

# When done, stop and remove Redis
docker stop autocomplete-redis
docker rm autocomplete-redis
```

### Redis Persistence

The Docker Compose setup includes:
- **Volume**: Data survives container restarts
- **AOF (Append Only File)**: Redis saves all write operations for data persistence
- **Health checks**: Ensures Redis is ready before connecting
- **Restart policy**: Redis restarts if it crashes

To clear all data and start fresh:
```bash
docker-compose down -v  # -v removes the volume
docker-compose up -d
```

## Features Demonstrated

This example demonstrates:
- **NGram-based search** with 3-character sequences
- **80 postal codes** from 10 major Indian cities
- **Multiple search types**: PIN codes, cities, districts, states
- **Partial matching** anywhere in the text
- **Example searches** with descriptions
- **Performance metrics** (indexing time, search time)
- **Statistics** about the indexed data
- **Interactive search mode** with both tabular and JSON output formats
- **Source data storage** with complete location information

## Example Searches with NGram

1. **PIN code partial**: `"110"` -> All postal codes containing "110"
2. **City partial**: `"umb"` -> Finds "Mumbai" postal codes
3. **State partial**: `"arnat"` -> Finds "Karnataka" postal codes
4. **District partial**: `"tral"` -> Finds "Central Delhi" postal codes
5. **Cross-word match**: `"ai Ch"` -> Finds "Mumbai City" or "Chennai"
6. **Middle match**: `"enn"` -> Finds "Chennai" postal codes

## Sample Data

The example includes postal codes from major Indian cities:
- **Delhi** (110xxx): Various districts of Delhi NCR
- **Mumbai** (400xxx): Mumbai City district, Maharashtra
- **Bangalore** (560xxx): Bangalore Urban district, Karnataka
- **Chennai** (600xxx): Chennai district, Tamil Nadu
- **Kolkata** (700xxx): Kolkata district, West Bengal
- **Hyderabad** (500xxx): Hyderabad district, Telangana
- **Ahmedabad** (380xxx): Ahmedabad district, Gujarat
- **Pune** (411xxx): Pune district, Maharashtra
- **Lucknow** (226xxx): Lucknow district, Uttar Pradesh
- **Jaipur** (302xxx): Jaipur district, Rajasthan

## Customization

You can extend this example by:
1. Adding more postal codes from the Indian Postal Index Number (PIN) directory
2. Including additional source data like area names, delivery post office names
3. Implementing location-based filtering (nearby postal codes)
4. Adding validation for 6-digit PIN code format
5. Integrating with maps for visual representation
