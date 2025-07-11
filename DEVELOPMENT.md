# Development Setup

This guide helps you set up your development environment for the CVL-KRA Autocomplete project.

## Prerequisites

- Go 1.23.3 or later
- Redis server (for testing)
- Python 3.x with pip (for pre-commit)
- Git

## Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd autocomplete

# Install all development tools
make install-tools

# Set up pre-commit hooks
make pre-commit-install

# Run all checks
make check
```

## Available Make Commands

### Core Commands

- `make build` - Build the project
- `make test` - Run tests with race detection
- `make coverage` - Generate test coverage report
- `make check` - Run all checks (fmt, vet, lint, test)

### Code Quality

- `make fmt` - Check code formatting (use `make fmt-fix` to auto-fix)
- `make vet` - Run go vet
- `make lint` - Run golangci-lint
- `make tidy` - Clean up go.mod and verify dependencies

### Development Tools

- `make install-tools` - Install all required development tools
- `make pre-commit-install` - Install pre-commit hooks
- `make pre-commit-run` - Manually run pre-commit on all files

### Other Commands

- `make clean` - Remove build artifacts
- `make bench` - Run benchmarks
- `make docs` - Start documentation server
- `make help` - Show all available commands

## Pre-commit Hooks

Pre-commit hooks automatically run before each commit to ensure code quality:

1. **Go checks**: formatting, imports, vet, cyclomatic complexity, linting
2. **General checks**: trailing whitespace, file endings, YAML validation
3. **Security**: secret detection, gosec scanning
4. **Tests**: runs tests and build

### Bypassing Hooks

In rare cases where you need to commit without running hooks:

```bash
git commit --no-verify -m "Emergency fix"
```

## Code Standards

This project follows coding standards:

### No Comments Policy

Code must be self-documenting. Comments are not allowed except for:
- Godoc comments on exported functions/types
- Build tags and code generation directives

### Self-Documenting Code Principles

1. **Clear Naming**: Functions and variables should have descriptive names
2. **Named Constants**: Extract magic numbers to named constants
3. **Small Functions**: Break complex logic into focused functions
4. **Type Safety**: Avoid `interface{}` without clear justification

### Example

Instead of:
```go
// Check if user is adult
if age >= 18 {
    // Process adult user
}
```

Write:
```go
const adultAgeThreshold = 18

func isAdult(age int) bool {
    return age >= adultAgeThreshold
}

if isAdult(age) {
    processAdultUser()
}
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run specific package tests
go test ./providers/redis/...
```

### Test Requirements

- All code must have tests
- Tests must be behavior-driven, not implementation-focused
- Use table-driven tests for multiple scenarios
- Mock external dependencies

## CI/CD Pipeline

GitHub Actions runs on every push and PR:

1. **Lint**: Code formatting and static analysis
2. **Test**: Unit tests with coverage reporting
3. **Build**: Cross-platform builds
4. **Security**: Vulnerability scanning

## Project Structure

```
autocomplete/
+-- providers/          # Storage provider implementations
|   +-- redis/         # Redis provider
+-- examples/          # Example applications
+-- .github/          # GitHub Actions workflows
+-- .golangci.yml     # Linter configuration
+-- .pre-commit-config.yaml  # Pre-commit hooks
+-- Makefile          # Build automation
+-- go.mod            # Go module definition
+-- tools.go          # Development tool dependencies
```

## Common Issues

### Pre-commit Not Found

```bash
# Install pre-commit
pip install --user pre-commit

# Or on macOS
brew install pre-commit
```

### Redis Connection Failed

Ensure Redis is running:
```bash
# Start Redis
redis-server

# Or with Docker
docker run -p 6379:6379 redis:alpine
```

### Linter Errors

Most linting issues can be auto-fixed:
```bash
# Fix formatting
make fmt-fix

# Then run full check
make check
```

## Contributing

1. Create a feature branch
2. Make changes following code standards
3. Ensure all tests pass: `make check`
4. Create PR with clear description

Remember: The pre-commit hooks will enforce standards automatically!
