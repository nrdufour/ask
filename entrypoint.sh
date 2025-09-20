#!/bin/sh

# entrypoint.sh - Startup script for the ask application
# This script ensures that the database is initialized before starting the server

set -e  # Exit on any error

echo "Starting Airport Swiss Knife application..."

# Set the port from environment variable or default to 8080
PORT=${PORT:-8080}

echo "Initializing database..."
# Run the init command to set up the database and import data
/app/ask init

echo "Database initialized successfully!"

echo "Starting server on port $PORT..."
# Start the server
exec /app/ask serve --port "$PORT"