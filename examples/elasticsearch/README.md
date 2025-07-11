# Elasticsearch Autocomplete Examples

This directory contains examples demonstrating the Elasticsearch autocomplete provider.

## Examples

- **basic/** - Simple string-based autocomplete with Indian postal codes
- **advanced/** - Advanced example with structured text data and interactive search

## Quick Start with Docker Compose

### Prerequisites

- Docker and Docker Compose installed
- No local Elasticsearch instance running on port 9200

### 1. Start Elasticsearch

```bash
# Start only Elasticsearch
docker-compose up -d elasticsearch

# Wait for Elasticsearch to be healthy
docker-compose ps
```

### 2. Run Examples Locally

With Elasticsearch running in Docker, you can run the examples locally:

```bash
# Basic example
cd basic
go run main.go

# Advanced example (interactive)
cd advanced
go run main.go
```

### 3. Run Examples in Docker

Build and run examples in containers:

```bash
# Run basic example
docker-compose --profile examples run --rm basic-example

# Run advanced example (interactive)
docker-compose --profile examples run --rm advanced-example
```

### 4. Optional: Start Kibana

For visualization and debugging:

```bash
docker-compose --profile with-kibana up -d kibana
```

Access Kibana at http://localhost:5601

## Docker Compose Services

### Elasticsearch
- URL: http://localhost:9200
- Single-node development setup
- Security disabled for easy testing
- 512MB heap size (configurable in .env)
- Data persisted in Docker volume

### Kibana (Optional)
- URL: http://localhost:5601
- Profile: `with-kibana`
- Connected to Elasticsearch

### Example Containers
- Profile: `examples`
- Built from source
- Connected to Elasticsearch network

## Configuration

Edit `.env` file to customize:
- Elasticsearch version
- Memory limits
- Ports
- Security settings

## Common Commands

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f elasticsearch

# Check service health
docker-compose ps

# Stop all services
docker-compose down

# Remove all data
docker-compose down -v

# Build examples
docker-compose --profile examples build
```

## Troubleshooting

### Connection Refused
- Ensure Elasticsearch is healthy: `docker-compose ps`
- Check logs: `docker-compose logs elasticsearch`
- Verify port 9200 is not in use: `lsof -i :9200`

### Out of Memory
- Increase heap size in docker-compose.yml
- Ensure Docker has enough memory allocated

### Examples Can't Connect
- Use `http://localhost:9200` when running locally
- Use `http://elasticsearch:9200` when running in containers

### NGram Tokenizer Error
If you see an error about `max_gram` and `min_gram` difference:
- This has been fixed in the provider code by setting `index.max_ngram_diff: 20`
- If using an existing index, delete it first: `curl -X DELETE http://localhost:9200/postal_codes_basic`
- The index will be recreated with correct settings on next run

## Development Tips

1. **Quick Testing**: Use `RefreshPolicy: "true"` in config for immediate visibility
2. **View Index**: `curl http://localhost:9200/postal_codes_basic/_search?pretty`
3. **Delete Index**: `curl -X DELETE http://localhost:9200/postal_codes_basic`
4. **Monitor Health**: `curl http://localhost:9200/_cluster/health?pretty`
5. **Debug Script**: Run `./debug.sh` to inspect index contents and test queries

## Environment Variables

The examples respect these environment variables:
- `ELASTICSEARCH_URL` - Override default URL (default: http://localhost:9200)
