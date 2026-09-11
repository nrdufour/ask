default:
    @just --list

# Build the Go binary
build:
    go build -o ask .

# Run the tests.
test:
    go test ./...

# vet and staticcheck.
lint:
    go vet ./...
    staticcheck ./...

# Fail if any tracked Go or Nix file needs formatting.
fmt-check:
    #!/usr/bin/env bash
    unformatted=$(gofmt -l $(git ls-files '*.go'))
    if [ -n "$unformatted" ]; then
        echo "these files need gofmt:"; echo "$unformatted"; exit 1
    fi
    nixfmt --check $(git ls-files '*.nix')

# What CI runs, so a red build is reproducible in one command.
# The binary needs cgo for go-sqlite3, so no CGO_ENABLED=0 here.
check: fmt-check lint test build

# Run the server locally
run: build
    ./ask serve

# Deploy to Fly.io
deploy:
    flyctl deploy

# Show Fly.io app status
status:
    flyctl status

# Tail Fly.io logs
logs:
    flyctl logs

# Remove build output.
clean:
    rm -f ask

