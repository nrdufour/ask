package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLatestTilesBuild(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Not in date order, as nothing promises it.
		w.Write([]byte(`[{"key":"20261012.pmtiles"},{"key":"20261014.pmtiles"},{"key":"20261013.pmtiles"}]`))
	}))
	defer ts.Close()
	defer func(old string) { tilesBuilds = old }(tilesBuilds)
	tilesBuilds = ts.URL

	got, err := latestTilesBuild(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "20261014.pmtiles" {
		t.Errorf("latest build = %s, want 20261014.pmtiles", got)
	}
}

func TestTilesState(t *testing.T) {
	if s := newTiles("").status(); s != tilesDisabled {
		t.Errorf("no path: %s, want %s", s, tilesDisabled)
	}

	path := filepath.Join(t.TempDir(), "basemap.pmtiles")
	tl := newTiles(path)
	if tl.status() != tilesAbsent || !tl.stale() {
		t.Errorf("missing file: %s, stale %v; want %s, stale", tl.status(), tl.stale(), tilesAbsent)
	}

	if err := os.WriteFile(path, []byte("PMTiles"), 0o644); err != nil {
		t.Fatal(err)
	}
	tl = newTiles(path)
	if tl.status() != tilesReady || tl.stale() {
		t.Errorf("fresh file: %s, stale %v; want %s, not stale", tl.status(), tl.stale(), tilesReady)
	}

	old := time.Now().Add(-tilesMaxAge - time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if !tl.stale() {
		t.Error("a file older than tilesMaxAge should be stale")
	}
	// Stale tiles keep serving until the update replaces them.
	if tl.status() != tilesReady {
		t.Errorf("stale file: %s, want %s", tl.status(), tilesReady)
	}
}
