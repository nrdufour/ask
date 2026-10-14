#!/bin/sh
# The database is baked into the image at build time; just serve it.
# ASK_TILES, when set, is where the self-hosted basemap lives (a volume).

if [ "$(id -u)" = 0 ]; then
    # Volumes mount owned by root: give the tiles directory to appuser, then
    # drop privileges for good.
    if [ -n "$ASK_TILES" ]; then
        mkdir -p "$(dirname "$ASK_TILES")"
        chown appuser:appgroup "$(dirname "$ASK_TILES")"
    fi
    exec su-exec appuser "$0" "$@"
fi

exec /app/ask serve --port "${PORT:-8080}" ${ASK_TILES:+--tiles "$ASK_TILES"}
