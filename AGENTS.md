# Airport Swiss Knife

**ask** is a Go CLI that downloads airport and country data from [OurAirports](https://ourairports.com/), stores it in SQLite, and serves it through a web UI and a JSON API.

## Commands

```bash
just check        # what CI runs: gofmt + nixfmt, vet, staticcheck, tests, build
just run          # build and serve on :8080
./ask init        # clone/pull the data and rebuild the database
./ask serve -p 8080
./ask search heathrow [--country GB] [--limit 20]
./ask distance LFPG KJFK
./ask range LFPG 300 [--type large_airport,heliport | --all-types]
./ask time LFPG KJFK
./ask info LFPG          # runways, frequencies, navaids
./ask version
```

Every query command takes `--json` and then prints exactly what the matching API endpoint returns.

The dev shell comes from `flake.nix` (direnv loads it). The binary needs cgo for `go-sqlite3`.

## Layout

- `cmd/` — Cobra commands. `output.go` holds what the query commands share (opening the store, tables, `--json`).
- `db/` — schema, indexes and the CSV import of airports, countries, regions, runways, frequencies (`airport-frequencies.csv`) and navaids. Columns are matched by CSV header name; empty fields become NULL. Runways and frequencies join airports on `airport_ref` = `airports.id`.
- `repository/` — clones or pulls the OurAirports data repo with go-git.
- `store/` — every airport question, for both the server and the CLI: `ByCode`, `Search`, `Route`, `InRange`, `LocalTime`, `Detail`, `Countries`, `ImportStatus`. `Detail` and `LocalTime` also take OurAirports idents (`FR-0496`) so airports without an ICAO code work too.
  - `detail.go` — runways (headings from the data, else computed from the thresholds, else read from the designator and marked magnetic), frequencies, navaids within 30 NM. It validates its inputs (`*InputError`, shown to users as-is) and wraps `ErrNotFound`; its result types are the API's JSON.
  - `geo.go` — haversine distance, initial bearing, and the bounding box used to prefilter range queries.
  - `testdata/` — the CSV fixtures both `store` and `server` tests load with `db.ImportDir`.
- `server/` — HTTP server on the stdlib `ServeMux`. `api_handlers.go` parses parameters and maps store errors to 400/404/500; `web_handlers.go` renders the pages.
- `web/` — templates and static assets, embedded into the binary (`web.FS`).
  - `templates/layout.html` defines the shared page; each page defines `title`, `content` and optionally `head` and `scripts`.
  - `static/app.css`, `static/app.js` — the only stylesheet and shared script. Fonts (B612), Leaflet and protomaps-leaflet are vendored under `static/`, no CDN.
- Map tiles: `ask serve --tiles <path>` serves a Protomaps PMTiles extract (z0–12, ~18 GB) at `/tiles/basemap.pmtiles`, rendered with protomaps-leaflet. `server/tiles.go` re-extracts it with the `pmtiles` CLI when missing or older than 28 days; `/health` reports its state. Without it the pages use tile.openstreetmap.org. On Fly it lives on the `tiles` volume at `/data`.

## API

- `GET /api/airport/search?name=&country=&limit=` — name, city, keywords or exact code; at most 100 results by default (`truncated` says if more matched).
- `GET /api/airport/distance?departure=&destination=` — NM, km, statute miles and initial true course.
- `GET /api/airport/reachable?icao=&range=&type=` — airports within range, nearest first.
- `GET /api/airport/time?icao=` — IANA timezone and local time.
- `GET /api/airport/{code}` — the airport with runways, frequencies and navaids within 30 NM. The page is `/airport/{code}`; `?wind=270/15` fills in its wind-component calculator.
- `GET /api/country`, `GET /api/import/status`, `GET /health`, `GET /version`.

4-letter codes are looked up by `icao_code`, then `ident`, then `gps_code`: most airports in the data have no `icao_code`.

## Notes

- The repository directory defaults to `$HOME/.ask` (`--repository`/`-r`): data in `data/`, database in `db/ask.db`. The server opens the database read-only.
- `init` recreates the database from scratch each time.
- The Docker image bakes the database in at build time; the container only serves. Its entrypoint starts as root only to hand the tiles volume to `appuser`.
- `.forgejo/workflows/deploy.yaml` deploys to Fly.io on every push to main and weekly (to refresh the data), using the `FLY_API_TOKEN` repo secret. `just deploy` still works by hand.
- The version comes from git: Go stamps the tag into the binary (`v0.3.0` on a tagged commit, a pseudo-version past it otherwise), so releasing is `git tag vX.Y.Z && git push --tags`. Builds need the tags, hence `fetch-depth: 0` in the workflows.
- The server runs in about 55 MB, timezone finder (`tzf` v2.1) included.
