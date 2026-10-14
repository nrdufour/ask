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

	for _, table := range []string{"airports", "countries"} {
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

// columns lists the columns of each data table, in the order of the
// ourairports.com CSV files.
var columns = map[string][]string{
	"airports": {
		"id", "ident", "type", "name", "latitude_deg", "longitude_deg", "elevation_ft",
		"continent", "iso_country", "iso_region", "municipality", "scheduled_service",
		"icao_code", "iata_code", "gps_code", "local_code", "home_link", "wikipedia_link", "keywords",
	},
	"countries": {"id", "code", "name", "continent", "wikipedia_link", "keywords"},
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
	CREATE INDEX IF NOT EXISTS airports_latitude ON airports(latitude_deg);`

	if _, err := db.Exec(indexes); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}
