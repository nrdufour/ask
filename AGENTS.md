# AGENTS.md

## Project Overview

Main Goal: Surfacing a database of airport related data coming from the site ourairports.com, and especially by storing their csv file first in a sqlite and exposing it via a REST api for later use by other clients.

**Name:** ask  
**Language:** Go  
**Type:** CLI Application  

## Architecture & Structure

- **Main entry point:** `main.go` or `cmd/`
- **Core logic:**: store and serve the airports.csv db as a REST server
- **Configuration:**: None yet, but will need to be added
- **Tests:**: none yet

## Development Commands

### Build & Run
```bash
# Build the project
go build

# Run the project
go run .

# Run with arguments
go run . [command] [flags]
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/[package-name]
```

### Code Quality
```bash
# Format code
go fmt ./...

# Lint code (if golangci-lint is available)
golangci-lint run

# Vet code
go vet ./...

# Tidy dependencies
go mod tidy
```

## Dependencies & Modules

Key dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management

## Configuration

- Config files location: 
- Environment variables: 
- Default settings: 

## Common Tasks

### Adding a new command
1. Create new command file in `cmd/` directory
2. Implement cobra.Command structure
3. Add command to root command
4. Add tests for the command

### Adding configuration options
1. Define config struct
2. Add viper bindings
3. Update config file examples
4. Document new options

## Testing Strategy

- Unit tests for core logic
- Integration tests for CLI commands
- Test coverage target: 80%+

## Deployment

- Build artifacts: 
- Release process: 
- Distribution: 

## Notes

- Follow Go conventions and idioms
- Use structured logging where applicable
- Ensure proper error handling
- Document public APIs

## Troubleshooting

Common issues and solutions:
  