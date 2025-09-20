package server

const VERSION = "v0.1"

type VersionResponse struct {
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type Airport struct {
	ID               int     `json:"id"`
	Ident            string  `json:"ident"`
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	LatitudeDeg      float64 `json:"latitude_deg"`
	LongitudeDeg     float64 `json:"longitude_deg"`
	ElevationFt      int     `json:"elevation_ft"`
	Continent        string  `json:"continent"`
	IsoCountry       string  `json:"iso_country"`
	IsoRegion        string  `json:"iso_region"`
	Municipality     string  `json:"municipality"`
	ScheduledService string  `json:"scheduled_service"`
	IcaoCode         string  `json:"icao_code"`
	IataCode         string  `json:"iata_code"`
	GpsCode          string  `json:"gps_code"`
	LocalCode        string  `json:"local_code"`
	HomeLink         string  `json:"home_link"`
	WikipediaLink    string  `json:"wikipedia_link"`
	Keywords         string  `json:"keywords"`
}

type Country struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Continent     string `json:"continent"`
	WikipediaLink string `json:"wikipedia_link"`
	Keywords      string `json:"keywords"`
}

type SearchResponse struct {
	Airports []Airport `json:"airports"`
	Count    int       `json:"count"`
}

type CountryListResponse struct {
	Countries []Country `json:"countries"`
	Count     int       `json:"count"`
}
