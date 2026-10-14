package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

// The basemap is an extract of the Protomaps world build, kept on disk and
// served to the browser with HTTP range requests. Without it the pages fall
// back to OpenStreetMap's tile servers.
var (
	tilesBuilds   = "https://build-metadata.protomaps.dev/builds.json"
	tilesBuildURL = "https://build.protomaps.com/"
)

const (
	// Vector tiles overzoom cleanly, and the airport page draws runways from
	// our own data, so z12 is detailed enough: 18 GB instead of 139.
	tilesMaxZoom = 12
	tilesMaxAge  = 28 * 24 * time.Hour
)

// Tile states, reported by /health.
const (
	tilesDisabled = "disabled" // no --tiles path
	tilesAbsent   = "absent"   // no file yet, no update running
	tilesUpdating = "updating"
	tilesReady    = "ready"
	tilesFailed   = "failed"
)

type tiles struct {
	path string

	mu    sync.Mutex
	state string
}

func newTiles(path string) *tiles {
	t := &tiles{path: path, state: tilesDisabled}
	if path == "" {
		return t
	}
	t.state = tilesAbsent
	if _, err := os.Stat(path); err == nil {
		t.state = tilesReady
	}
	return t
}

func (t *tiles) status() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

func (t *tiles) setStatus(s string) {
	t.mu.Lock()
	t.state = s
	t.mu.Unlock()
}

// stale reports whether the basemap is missing or older than tilesMaxAge.
func (t *tiles) stale() bool {
	info, err := os.Stat(t.path)
	return err != nil || time.Since(info.ModTime()) > tilesMaxAge
}

// refreshIfStale updates the basemap in the background when it is missing or
// old. The deploy workflow polls /health until it is ready. The extract is
// not resumable, so the machine must stay up meanwhile: fly.toml turns
// auto-stop off.
func (t *tiles) refreshIfStale(ctx context.Context) {
	if t.path == "" || !t.stale() {
		return
	}
	go func() {
		t.setStatus(tilesUpdating)
		if err := t.update(ctx); err != nil {
			log.Printf("tiles: update failed: %v", err)
			t.setStatus(tilesFailed)
			return
		}
		t.setStatus(tilesReady)
	}()
}

// update replaces the basemap with an extract of the latest Protomaps build.
// The old file goes first: the volume holds only one copy, and the pages fall
// back to OpenStreetMap meanwhile.
func (t *tiles) update(ctx context.Context) error {
	build, err := latestTilesBuild(ctx)
	if err != nil {
		return err
	}
	log.Printf("tiles: extracting %s up to z%d into %s", build, tilesMaxZoom, t.path)

	if err := os.Remove(t.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	tmp := t.path + ".tmp"
	os.Remove(tmp)

	start := time.Now()
	cmd := exec.CommandContext(ctx, "pmtiles", "extract", tilesBuildURL+build, tmp,
		fmt.Sprintf("--maxzoom=%d", tilesMaxZoom), "--download-threads=8")
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("pmtiles extract: %w", err)
	}
	if err := os.Rename(tmp, t.path); err != nil {
		return err
	}
	log.Printf("tiles: %s ready in %s", build, time.Since(start).Round(time.Second))
	return nil
}

// latestTilesBuild returns the file name of the newest Protomaps build,
// such as "20261014.pmtiles".
func latestTilesBuild(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tilesBuilds, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", tilesBuilds, resp.Status)
	}

	var builds []struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return "", fmt.Errorf("%s: %w", tilesBuilds, err)
	}
	if len(builds) == 0 {
		return "", fmt.Errorf("%s lists no build", tilesBuilds)
	}
	// Keys are dated, YYYYMMDD.pmtiles: the largest is the newest.
	sort.Slice(builds, func(i, j int) bool { return builds[i].Key < builds[j].Key })
	return builds[len(builds)-1].Key, nil
}

// tilesHandler serves the basemap, range requests included, once it is ready.
func (s *Server) tilesHandler(w http.ResponseWriter, r *http.Request) {
	if s.tiles.status() != tilesReady {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, s.tiles.path)
}
