# Multi-stage build for Go examples
FROM golang:1.23.3-alpine AS builder

# Install git and ca-certificates for fetching dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build basic example
FROM builder AS basic-builder
RUN go build -o /basic-example ./examples/elasticsearch/basic/main.go

# Build advanced example
FROM builder AS advanced-builder
RUN go build -o /advanced-example ./examples/elasticsearch/advanced/main.go

# Final stage for basic example
FROM alpine:latest AS basic
RUN apk add --no-cache ca-certificates
COPY --from=basic-builder /basic-example /basic-example
CMD ["/basic-example"]

# Final stage for advanced example
FROM alpine:latest AS advanced
RUN apk add --no-cache ca-certificates
COPY --from=advanced-builder /advanced-example /advanced-example
CMD ["/advanced-example"]
