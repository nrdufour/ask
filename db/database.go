/*
Copyright © 2024 Nicolas Dufour
*/
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

// Path returns the location of the SQLite database inside the repository.
func Path() string {
	return filepath.Join(viper.GetString("repository"), viper.GetString("db"), "ask.db")
}

// InitializeDatabase recreates the database and imports airport and country data
func InitializeDatabase() error {
	dbPath := Path()
	if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	// Remove existing database file to ensure clean recreation
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing database: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := CreateSchema(db); err != nil {
		return err
	}
	fmt.Printf("Database created at: %s\n", dbPath)

	commitHash, commitDate, err := getGitCommitInfo()
	if err != nil {
		return fmt.Errorf("failed to get git commit info: %w", err)
	}

	for _, table := range tableOrder {
		if err := importTable(db, table, commitHash, commitDate); err != nil {
			return fmt.Errorf("failed to import %s data: %w", table, err)
		}
	}

	if err := CreateIndexes(db); err != nil {
		return err
	}

	fmt.Println("Database initialized successfully!")
	return nil
}

// tableOrder lists the imported tables; csvFiles names the OurAirports file
// each one comes from.
var (
	tableOrder = []string{"airports", "countries", "regions", "runways", "frequencies", "navaids"}
	csvFiles   = map[string]string{
		"airports":    "airports.csv",
		"countries":   "countries.csv",
		"regions":     "regions.csv",
		"runways":     "runways.csv",
		"frequencies": "airport-frequencies.csv",
		"navaids":     "navaids.csv",
	}
)

// columns lists the columns of each data table, in the order of the
// ourairports.com CSV files.
var columns = map[string][]string{
	"airports": {
		"id", "ident", "type", "name", "latitude_deg", "longitude_deg", "elevation_ft",
		"continent", "iso_country", "iso_region", "municipality", "scheduled_service",
		"icao_code", "iata_code", "gps_code", "local_code", "home_link", "wikipedia_link", "keywords",
	},
	"countries": {"id", "code", "name", "continent", "wikipedia_link", "keywords"},
	"regions":   {"id", "code", "local_code", "name", "continent", "iso_country", "wikipedia_link", "keywords"},
	"runways": {
		"id", "airport_ref", "airport_ident", "length_ft", "width_ft", "surface", "lighted", "closed",
		"le_ident", "le_latitude_deg", "le_longitude_deg", "le_elevation_ft", "le_heading_degT", "le_displaced_threshold_ft",
		"he_ident", "he_latitude_deg", "he_longitude_deg", "he_elevation_ft", "he_heading_degT", "he_displaced_threshold_ft",
	},
	"frequencies": {"id", "airport_ref", "airport_ident", "type", "description", "frequency_mhz"},
	"navaids": {
		"id", "filename", "ident", "name", "type", "frequency_khz", "latitude_deg", "longitude_deg", "elevation_ft",
		"iso_country", "dme_frequency_khz", "dme_channel", "dme_latitude_deg", "dme_longitude_deg", "dme_elevation_ft",
		"slaved_variation_deg", "magnetic_variation_deg", "usageType", "power", "associated_airport",
	},
}

// CreateSchema creates the airports, countries and import_status tables
func CreateSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS airports (
		id INTEGER,
		ident TEXT,
		type TEXT,
		name TEXT,
		latitude_deg REAL,
		longitude_deg REAL,
		elevation_ft INTEGER,
		continent TEXT,
		iso_country TEXT,
		iso_region TEXT,
		municipality TEXT,
		scheduled_service TEXT,
		icao_code TEXT,
		iata_code TEXT,
		gps_code TEXT,
		local_code TEXT,
		home_link TEXT,
		wikipedia_link TEXT,
		keywords TEXT
	);
	CREATE TABLE IF NOT EXISTS countries (
		id INTEGER,
		code TEXT,
		name TEXT,
		continent TEXT,
		wikipedia_link TEXT,
		keywords TEXT
	);
	CREATE TABLE IF NOT EXISTS regions (
		id INTEGER,
		code TEXT,
		local_code TEXT,
		name TEXT,
		continent TEXT,
		iso_country TEXT,
		wikipedia_link TEXT,
		keywords TEXT
	);
	CREATE TABLE IF NOT EXISTS runways (
		id INTEGER,
		airport_ref INTEGER,
		airport_ident TEXT,
		length_ft INTEGER,
		width_ft INTEGER,
		surface TEXT,
		lighted INTEGER,
		closed INTEGER,
		le_ident TEXT,
		le_latitude_deg REAL,
		le_longitude_deg REAL,
		le_elevation_ft INTEGER,
		le_heading_degT REAL,
		le_displaced_threshold_ft INTEGER,
		he_ident TEXT,
		he_latitude_deg REAL,
		he_longitude_deg REAL,
		he_elevation_ft INTEGER,
		he_heading_degT REAL,
		he_displaced_threshold_ft INTEGER
	);
	CREATE TABLE IF NOT EXISTS frequencies (
		id INTEGER,
		airport_ref INTEGER,
		airport_ident TEXT,
		type TEXT,
		description TEXT,
		frequency_mhz REAL
	);
	CREATE TABLE IF NOT EXISTS navaids (
		id INTEGER,
		filename TEXT,
		ident TEXT,
		name TEXT,
		type TEXT,
		frequency_khz INTEGER,
		latitude_deg REAL,
		longitude_deg REAL,
		elevation_ft INTEGER,
		iso_country TEXT,
		dme_frequency_khz INTEGER,
		dme_channel TEXT,
		dme_latitude_deg REAL,
		dme_longitude_deg REAL,
		dme_elevation_ft INTEGER,
		slaved_variation_deg REAL,
		magnetic_variation_deg REAL,
		usageType TEXT,
		power TEXT,
		associated_airport TEXT
	);
	CREATE TABLE IF NOT EXISTS import_status (
		table_name TEXT PRIMARY KEY,
		last_import_date TEXT,
		git_commit_hash TEXT,
		git_commit_date TEXT,
		record_count INTEGER
	);`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// CreateIndexes indexes the columns used for airport lookups. Run it after
// the import: building an index once is faster than maintaining it per insert.
func CreateIndexes(db *sql.DB) error {
	indexes := `
	CREATE INDEX IF NOT EXISTS airports_icao_code ON airports(icao_code);
	CREATE INDEX IF NOT EXISTS airports_iata_code ON airports(iata_code);
	CREATE INDEX IF NOT EXISTS airports_ident ON airports(ident);
	CREATE INDEX IF NOT EXISTS airports_gps_code ON airports(gps_code);
	CREATE INDEX IF NOT EXISTS airports_latitude ON airports(latitude_deg);
	CREATE INDEX IF NOT EXISTS regions_code ON regions(code);
	CREATE INDEX IF NOT EXISTS runways_airport ON runways(airport_ref);
	CREATE INDEX IF NOT EXISTS frequencies_airport ON frequencies(airport_ref);
	CREATE INDEX IF NOT EXISTS navaids_latitude ON navaids(latitude_deg);`

	if _, err := db.Exec(indexes); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}
