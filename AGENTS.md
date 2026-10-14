# Airport Swiss Knife

**ask** is a Go CLI that downloads airport and country data from [OurAirports](https://ourairports.com/), stores it in SQLite, and serves it through a web UI and a JSON API.

## Commands

```bash
just check        # what CI runs: gofmt + nixfmt, vet, staticcheck, tests, build
just run          # build and serve on :8080
./ask init        # clone/pull the data and rebuild the database
./ask serve -p 8080
./ask version
```

The dev shell comes from `flake.nix` (direnv loads it). The binary needs cgo for `go-sqlite3`.

## Layout

- `cmd/` — Cobra commands (`init`, `serve`, `version`, `query` — the last is a stub).
- `db/` — schema, indexes and the CSV import. Columns are matched by CSV header name; empty fields become NULL.
- `repository/` — clones or pulls the OurAirports data repo with go-git.
- `server/` — HTTP server on the stdlib `ServeMux`.
  - `airports.go` — all airport queries and row scanning (`getAirportByCode`, `searchAirports`, `getAirportsInRange`).
  - `geo.go` — haversine distance, initial bearing, and the bounding box used to prefilter range queries.
  - `*_handlers.go` — one file per API area; `web_handlers.go` renders the pages.
- `web/` — templates and static assets, embedded into the binary (`web.FS`).
  - `templates/layout.html` defines the shared page; each page defines `title`, `content` and optionally `head` and `scripts`.
  - `static/app.css`, `static/app.js` — the only stylesheet and shared script. Fonts (B612) and Leaflet are vendored under `static/`, no CDN. Map tiles come from tile.openstreetmap.org.

## API

- `GET /api/airport/search?name=&country=&limit=` — name, city, keywords or exact code; at most 100 results by default (`truncated` says if more matched).
- `GET /api/airport/distance?departure=&destination=` — NM, km, statute miles and initial true course.
- `GET /api/airport/reachable?icao=&range=&type=` — airports within range, nearest first.
- `GET /api/airport/time?icao=` — IANA timezone and local time.
- `GET /api/country`, `GET /api/import/status`, `GET /health`, `GET /version`.

4-letter codes are looked up by `icao_code`, then `ident`, then `gps_code`: most airports in the data have no `icao_code`.

## Notes

- The repository directory defaults to `$HOME/.ask` (`--repository`/`-r`): data in `data/`, database in `db/ask.db`. The server opens the database read-only.
- `init` recreates the database from scratch each time.
- The Docker image bakes the database in at build time; the container only serves.
- Version lives in `server/types.go` (`VERSION`); bump it when tagging, along with `IMAGE_VERSION` in the Dockerfile.
- The server runs in about 55 MB, timezone finder (`tzf` v2.1) included.
