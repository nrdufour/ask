#!/bin/sh
# The database is baked into the image at build time; just serve it.
exec /app/ask serve --port "${PORT:-8080}"
