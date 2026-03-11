default:
    @just --list

# Build the Go binary
build:
    go build -o ask .

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
