package server

import "ask/store"

const VERSION = "v0.2.2"

type VersionResponse struct {
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type CountryListResponse struct {
	Countries []store.Country `json:"countries"`
	Count     int             `json:"count"`
}

type ImportStatusResponse struct {
	Tables []store.ImportStatus `json:"tables"`
}
