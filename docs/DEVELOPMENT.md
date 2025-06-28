# Development Guide

## Table of Contents
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Code Style](#code-style)
- [API Documentation](#api-documentation)
- [Debugging](#debugging)
- [Performance Profiling](#performance-profiling)
- [Dependency Management](#dependency-management)
- [Release Process](#release-process)
- [Code Review Guidelines](#code-review-guidelines)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools

- Go 1.21 or later
- Git
- Make
- Docker & Docker Compose (for containerized development)
- golangci-lint (for linting)
- gomock (for generating mocks)
- air (for live reloading)
- protoc (if working with protocol buffers)

### Optional Tools

- Delve or Delve DAP (for debugging)
- Graphviz (for visualization)
- k6 (for load testing)
- Postman (for API testing)

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/copyleftdev/payer-status-io.git
cd payer-status-io
```

### Install Dependencies

```bash
# Install Go dependencies
make deps

# Install pre-commit hooks
make install-hooks
```

### Build the Application

```bash
# Build the binary
make build

# Build for all platforms
make release
```

### Run the Application

```bash
# Run in development mode (with live reload)
make dev

# Run tests
make test

# Run linters
make lint
```

## Project Structure

```
.
├── cmd/                  # Main application entry points
│   └── server/           # WebSocket server
│       ├── main.go       # Main application entry point
│       └── config.go     # Configuration loading
├── internal/             # Private application code
│   ├── config/           # Configuration structures and validation
│   ├── hub/              # WebSocket hub implementation
│   ├── prober/           # HTTP prober implementation
│   ├── scheduler/        # Probe scheduling logic
│   └── metrics/          # Metrics collection and reporting
├── pkg/                  # Public libraries
│   └── api/              # Public API types and clients
├── web/                  # Web assets
│   ├── static/           # Static files
│   └── templates/        # HTML templates
├── test/                 # Test utilities and fixtures
├── scripts/              # Build and deployment scripts
├── docs/                 # Documentation
├── .github/              # GitHub configuration
│   └── workflows/        # GitHub Actions workflows
├── .golangci.yml         # Linter configuration
├── Makefile              # Build automation
├── go.mod               # Go module definition
└── go.sum               # Go module checksums
```

## Development Workflow

### Branch Strategy

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit with a descriptive message:
   ```bash
   git commit -m "feat: add new feature"
   ```

3. Push your branch and create a pull request:
   ```bash
   git push -u origin feature/your-feature-name
   ```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Example:
```
feat(api): add support for JWT authentication

- Add JWT middleware
- Update authentication service
- Add tests for JWT validation

Closes #123
```

### Code Review Process

1. Create a draft PR for early feedback
2. Request reviews from at least one maintainer
3. Address all review comments
4. Ensure all tests pass
5. Squash and merge when approved

## Testing

### Unit Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run tests for a specific package
make test PKG=./internal/hub
```

### Integration Tests

```bash
# Start dependencies (Docker required)
mock-services up

# Run integration tests
make test-integration

# Clean up
mock-services down
```

### E2E Tests

```bash
# Start the application in test mode
make test-e2e-setup

# Run end-to-end tests
make test-e2e

# Stop test environment
make test-e2e-teardown
```

### Load Testing

```bash
# Install k6 if not already installed
brew install k6

# Run load test
k6 run scripts/loadtest/script.js
```

## Code Style

### Formatting

```bash
# Format code
make fmt

# Check formatting
make fmt-check
```

### Linting

```bash
# Run linters
make lint

# Fix linting issues
make lint-fix
```

### Code Generation

```bash
# Generate mocks
make generate
```

## API Documentation

### Generate API Documentation

```bash
# Generate OpenAPI/Swagger docs
make docs

# Serve API documentation
make docs-serve
```

### Update API Client Libraries

```bash
# Generate Go client
make generate-client-go

# Generate TypeScript client
make generate-client-ts
```

## Debugging

### Using Delve

```bash
# Start the application in debug mode
dlv debug --listen=:2345 --headless=true --api-version=2 --accept-multiclient ./cmd/server
```

### Debugging in VS Code

1. Install the Go extension
2. Add this to your `launch.json`:
   ```json
   {
     "name": "Debug Server",
     "type": "go",
     "request": "launch",
     "mode": "debug",
     "program": "cmd/server/main.go",
     "args": ["--config", "config/local.yaml"]
   }
   ```

### Profiling

```bash
# Generate CPU profile
make profile-cpu

# Generate memory profile
make profile-mem

# Generate trace
make profile-trace
```

## Performance Profiling

### CPU Profiling

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // ... rest of your code
}
```

Analyze with:

```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/profile?seconds=30
```

### Memory Profiling

```bash
# Get heap profile
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap

# Get memory allocation profile
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/allocs
```

### Tracing

```go
import "runtime/trace"

func handler(w http.ResponseWriter, r *http.Request) {
    f, _ := os.Create("trace.out")
    defer f.Close()
    
    trace.Start(f)
    defer trace.Stop()
    
    // Your code here
}
```

View the trace:

```bash
go tool trace trace.out
```

## Dependency Management

### Adding Dependencies

```bash
# Add a new dependency
go get github.com/example/package

# Update go.mod and go.sum
go mod tidy
```

### Updating Dependencies

```bash
# Update a specific dependency
go get -u github.com/example/package

# Update all dependencies
go get -u ./...

# Clean up unused dependencies
go mod tidy
```

### Vendoring

```bash
# Create/update vendor directory
go mod vendor

# Verify dependencies
go mod verify
```

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/). Update the version in:

- `VERSION` file
- `internal/version/version.go`
- Any relevant documentation

### Creating a Release

1. Create a release branch:
   ```bash
   git checkout -b release/vX.Y.Z
   ```

2. Update changelog:
   ```bash
   make changelog
   ```

3. Commit changes:
   ```bash
   git add .
   git commit -m "chore: release vX.Y.Z"
   git tag -a vX.Y.Z -m "Version X.Y.Z"
   ```

4. Push changes:
   ```bash
   git push origin vX.Y.Z
   git push origin release/vX.Y.Z
   ```

5. Create a GitHub release
   - Go to Releases > Draft a new release
   - Tag version: vX.Y.Z
   - Release title: vX.Y.Z
   - Description: Copy from CHANGELOG.md
   - Attach binaries from `dist/`
   - Publish release

## Code Review Guidelines

### For Reviewers

- Focus on code correctness, security, and maintainability
- Check for proper error handling and logging
- Ensure consistency with the project's coding style
- Verify test coverage
- Look for potential performance issues
- Check for proper documentation

### For Authors

- Keep PRs focused and small
- Write clear commit messages
- Include tests for new features
- Update documentation as needed
- Address all review comments
- Rebase on main before final review

## Troubleshooting

### Common Issues

#### Build Failures

```bash
# Clean and rebuild
make clean
make
```

#### Test Failures

```bash
# Run with verbose output
go test -v ./...

# Run a specific test
go test -run TestName
```

#### Dependency Issues

```bash
# Clear module cache
go clean -modcache

# Update dependencies
go get -u

go mod tidy
```

### Getting Help

1. Check the documentation
2. Search existing issues
3. Ask in the #dev channel
4. Open a new issue if needed

## Contributing

### Reporting Issues

When reporting issues, please include:

1. Version of the application
2. Steps to reproduce
3. Expected behavior
4. Actual behavior
5. Logs or error messages

### Feature Requests

For feature requests:

1. Describe the problem you're trying to solve
2. Explain why this feature is needed
3. Propose a solution
4. Include any relevant examples or references

### Pull Requests

Before submitting a PR:

1. Ensure all tests pass
2. Update documentation
3. Follow the code style
4. Keep commits atomic
5. Write meaningful commit messages
6. Reference related issues

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
