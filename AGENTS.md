# Airpot Swiss Knife

## Project Overview

**ask** (Airport Swiss Knife) is a Go CLI application that provides access to airport and country data from ourairports.com. It downloads, stores, and serves airport data via both CLI commands and a web server with REST API.

## Development Commands

### Build & Run
```bash
# Build the project
go build
# or use the Makefile
make build

# Run directly
go run .

# Run specific commands
./ask init     # Download data and setup local database
./ask serve    # Start HTTP server (default port 8080)
./ask query    # Query airport data
./ask version  # Show version
```

### Code Quality & Testing
```bash
# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Tidy dependencies
go mod tidy

# Run tests (if any exist)
go test ./...
```

## Architecture

The application follows a modular structure:

- **`cmd/`** - CLI commands using Cobra framework
  - `root.go` - Main command setup with global flags and viper config
  - `init.go` - Data download and database initialization
  - `serve.go` - HTTP server startup with graceful shutdown
  - `query.go` - Data querying functionality
  - `version.go` - Version information

- **`server/`** - HTTP server implementation
  - `server.go` - Main server setup with Gorilla Mux router
  - `*_handlers.go` - HTTP handlers for different endpoints
  - `types.go` - Data structures and response types
  - `validators.go` - Input validation logic
  - `distance.go` - Airport distance calculation utilities

- **`db/`** - Database operations
  - `database.go` - SQLite database creation and schema
  - `import.go` - CSV data import from ourairports.com

- **`repository/`** - Data management
  - `setup.go` - Git repository management for data updates

- **`templates/`** - HTML templates for web interface
  - Provides web UI for airport search and distance calculations

## Configuration

The application uses Viper for configuration management:

- **Config file**: `$HOME/.ask.yaml` (optional)
- **Default repository**: `$HOME/.ask/`
- **Database location**: `$HOME/.ask/db/ask.db`
- **Data location**: `$HOME/.ask/data/` (ourairports.com git repo)

Global flags:
- `--repository, -r` - Override default repository directory
- `--config` - Specify custom config file

## Data Flow

1. **Initialization** (`./ask init`):
   - Creates repository directory structure
   - Clones/updates ourairports.com data via Git
   - Creates SQLite database with airports and countries tables
   - Imports CSV data with import status tracking

2. **Server Mode** (`./ask serve`):
   - Starts HTTP server with both API and web endpoints
   - API endpoints: `/api/airport/search`, `/api/airport/distance`, `/api/country`
   - Web interface: `/`, `/airports`, `/distance`
   - Graceful shutdown with SIGINT/SIGTERM handling

## Database Schema

- **airports** - Main airport data with coordinates, codes, and metadata
- **countries** - Country information with codes and names
- **import_status** - Tracks data import metadata including git commit info

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/gorilla/mux` - HTTP router
- `github.com/mattn/go-sqlite3` - SQLite database driver
- `github.com/go-git/go-git/v5` - Git operations for data updates

## Development Notes

- SQLite database is recreated on each `init` to ensure clean state
- Server includes request logging middleware
- Distance calculations use geographical coordinates
- Data is automatically updated from ourairports.com git repository
- Web interface provides user-friendly access to API functionality