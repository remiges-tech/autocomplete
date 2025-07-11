# Tool versions (update these to change tool versions)
GOLANGCI_VERSION=v1.61.0

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=gofmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build parameters
BINARY_NAME=autocomplete
COVERAGE_FILE=coverage.out

# Colors for output
GREEN=\033[0;32m
RED=\033[0;31m
YELLOW=\033[0;33m
NC=\033[0m # No Color

.PHONY: all test build clean fmt lint check install-tools coverage help pre-commit-install pre-commit-run

# Default target
all: check build

# Help target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the project"
	@echo "  make test          - Run tests"
	@echo "  make test-integration - Run integration tests (requires Docker)"
	@echo "  make test-all      - Run all tests including integration"
	@echo "  make coverage      - Run tests with coverage report"
	@echo "  make coverage-integration - Run tests with coverage (including integration)"
	@echo "  make coverage-check - Check if coverage meets minimum threshold"
	@echo "  make fmt           - Format code using gofmt"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make vet           - Run go vet"
	@echo "  make check         - Run fmt, vet, lint, and test"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install-tools - Install required development tools"
	@echo "  make tidy          - Run go mod tidy"
	@echo "  make doc           - View package documentation in terminal"
	@echo "  make doc-all       - View all docs including unexported symbols"
	@echo "  make doc-pkg PKG=name - View documentation for specific package"
	@echo "  make doc-serve     - Start documentation web server"
	@echo "  make pre-commit-install - Install pre-commit hooks"
	@echo "  make pre-commit-run - Run pre-commit on all files"

# Build the project
build:
	@echo "$(GREEN)Building...$(NC)"
	$(GOBUILD) -v ./...

# Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race ./...

# Run integration tests
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	$(GOTEST) -v -race -tags=integration ./...

# Run all tests including integration
test-all: test test-integration

# Run tests with coverage
coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "$(GREEN)Coverage report:$(NC)"
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "$(YELLOW)To view HTML coverage report, run: go tool cover -html=$(COVERAGE_FILE)$(NC)"

# Run tests with coverage including integration tests
coverage-integration:
	@echo "$(GREEN)Running tests with coverage (including integration)...$(NC)"
	$(GOTEST) -v -race -tags=integration -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "$(GREEN)Coverage report:$(NC)"
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "$(YELLOW)To view HTML coverage report, run: go tool cover -html=$(COVERAGE_FILE)$(NC)"

# Check test coverage meets minimum threshold
COVERAGE_THRESHOLD=80
# Packages to exclude from coverage requirements (space-separated patterns)
COVERAGE_EXCLUDE_PATTERNS=/examples/ /cmd/ /tools/ /scripts/ /mocks/ /testdata/
coverage-check:
	@echo "$(GREEN)Checking test coverage (minimum $(COVERAGE_THRESHOLD)%)...$(NC)"
	@echo "$(GREEN)Coverage by package:$(NC)"
	@rm -f /tmp/coverage_failed; \
	$(GOCMD) test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./... 2>&1 | grep "coverage:" | while read -r line; do \
		pkg=$$(echo "$$line" | awk '{print $$2}'); \
		coverage=$$(echo "$$line" | awk '{print $$5}' | sed 's/%//' | cut -d. -f1); \
		excluded=0; \
		for pattern in $(COVERAGE_EXCLUDE_PATTERNS); do \
			if echo "$$pkg" | grep -q "$$pattern"; then \
				excluded=1; \
				break; \
			fi; \
		done; \
		if [ "$$excluded" -eq 1 ]; then \
			printf "  %-60s %s%% %s\n" "$$pkg" "$$coverage" "$(YELLOW)(excluded)$(NC)"; \
		elif [ -z "$$coverage" ] || [ "$$coverage" = "statements" ] || [ "$$pkg" = "?" ]; then \
			continue; \
		elif [ "$$coverage" -lt "$(COVERAGE_THRESHOLD)" ]; then \
			printf "  %-60s $(RED)%s%%$(NC) < $(COVERAGE_THRESHOLD)%%\n" "$$pkg" "$$coverage"; \
			touch /tmp/coverage_failed; \
		else \
			printf "  %-60s $(GREEN)%s%%$(NC)\n" "$$pkg" "$$coverage"; \
		fi; \
	done; \
	if [ -f /tmp/coverage_failed ]; then \
		rm -f /tmp/coverage_failed; \
		echo "$(RED)Coverage check failed! Some packages are below $(COVERAGE_THRESHOLD)%$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)All packages meet minimum coverage threshold!$(NC)"; \
	fi

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	@output=$$($(GOFMT) -l .); \
	if [ -n "$$output" ]; then \
		echo "$(RED)The following files need formatting:$(NC)"; \
		echo "$$output"; \
		echo "$(YELLOW)Run 'make fmt-fix' to fix them$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)All files are properly formatted$(NC)"; \
	fi

# Fix formatting
fmt-fix:
	@echo "$(GREEN)Fixing code formatting...$(NC)"
	$(GOFMT) -w .

# Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) ./...

# Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "$(RED)golangci-lint not installed. Run 'make install-tools' to install it$(NC)"; \
		exit 1; \
	fi

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(GREEN)All checks passed!$(NC)"

# Clean build artifacts
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)

# Run go mod tidy
tidy:
	@echo "$(GREEN)Running go mod tidy...$(NC)"
	$(GOMOD) tidy
	$(GOMOD) verify

# Install development tools
install-tools:
	@echo "$(GREEN)Installing development tools...$(NC)"
	# Install golangci-lint
	@if ! command -v $(GOLINT) >/dev/null 2>&1; then \
		echo "Installing golangci-lint $(GOLANGCI_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_VERSION); \
	else \
		echo "golangci-lint already installed"; \
	fi
	# Install pre-commit
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing pre-commit...$(NC)"; \
		pip install --user pre-commit || echo "$(RED)Failed to install pre-commit. Please install Python and pip first.$(NC)"; \
	else \
		echo "pre-commit already installed"; \
	fi
	# Download dependencies
	$(GOGET) -u ./...
	$(GOMOD) download
	@echo "$(GREEN)All tools installed!$(NC)"

# Quick check - format and test only
quick: fmt-fix test
	@echo "$(GREEN)Quick check passed!$(NC)"

# Run example
run-example:
	@echo "$(GREEN)Running basic example...$(NC)"
	$(GOCMD) run examples/basic/main.go

# Run benchmarks
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./...

# Check for security vulnerabilities
security:
	@echo "$(GREEN)Checking for vulnerabilities...$(NC)"
	$(GOCMD) list -json -deps ./... | nancy sleuth

# View package documentation in terminal
doc:
	@echo "$(GREEN)Showing documentation for current package...$(NC)"
	@$(GOCMD) doc -all .

# View documentation for specific package
doc-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "$(RED)Please specify a package: make doc-pkg PKG=providers$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Showing documentation for package $(PKG)...$(NC)"
	@$(GOCMD) doc -all ./$(PKG)

# View all documentation including unexported
doc-all:
	@echo "$(GREEN)Showing all documentation (including unexported)...$(NC)"
	@$(GOCMD) doc -all -u .

# Start documentation server (modern approach using pkgsite)
doc-serve:
	@echo "$(GREEN)Starting documentation server...$(NC)"
	@if command -v pkgsite >/dev/null 2>&1; then \
		pkgsite -http=:6060 & \
		echo "$(GREEN)Documentation server started at http://localhost:6060$(NC)"; \
		echo "$(YELLOW)Browse to http://localhost:6060/github.com/remiges/cvl-kra/autocomplete$(NC)"; \
	else \
		echo "$(YELLOW)pkgsite not found. Install with: go install golang.org/x/pkgsite/cmd/pkgsite@latest$(NC)"; \
		echo "$(YELLOW)Falling back to godoc...$(NC)"; \
		godoc -http=:6060 & \
		echo "$(GREEN)Documentation server started at http://localhost:6060$(NC)"; \
	fi

# Generate documentation (legacy - kept for compatibility)
docs: doc-serve

# Install pre-commit hooks
pre-commit-install:
	@echo "$(GREEN)Installing pre-commit hooks...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "$(GREEN)Pre-commit hooks installed successfully!$(NC)"; \
	else \
		echo "$(RED)pre-commit not found. Please install it first with: pip install --user pre-commit$(NC)"; \
		exit 1; \
	fi

# Run pre-commit on all files
pre-commit-run:
	@echo "$(GREEN)Running pre-commit on all files...$(NC)"
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "$(RED)pre-commit not found. Please install it first with: pip install --user pre-commit$(NC)"; \
		exit 1; \
	fi
