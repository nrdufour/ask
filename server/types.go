package server

import (
	"runtime/debug"

	"ask/store"
)

// VERSION is the version Go stamps from git when building: the tag on a
// tagged commit ("v0.3.0"), a pseudo-version past the last tag otherwise,
// "+dirty" with uncommitted changes. Releasing is tagging; nothing to bump.
var VERSION = func() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}()

type VersionResponse struct {
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
	// Tiles is the basemap's state: disabled, absent, updating, ready or failed.
	Tiles string `json:"tiles"`
}

type CountryListResponse struct {
	Countries []store.Country `json:"countries"`
	Count     int             `json:"count"`
}

type ImportStatusResponse struct {
	Tables []store.ImportStatus `json:"tables"`
}
