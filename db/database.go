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

// InitializeDatabase creates the database and imports airport data
func InitializeDatabase() error {
	dbPath, err := createDatabase()
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	err = ImportAirportsCSV(dbPath)
	if err != nil {
		return fmt.Errorf("failed to import airports data: %w", err)
	}

	err = ImportCountriesCSV(dbPath)
	if err != nil {
		return fmt.Errorf("failed to import countries data: %w", err)
	}

	fmt.Println("Database initialized successfully!")
	return nil
}

// createDatabase creates the SQLite database and airports table
func createDatabase() (string, error) {
	repoDir := viper.GetString("repository")
	dbDir := filepath.Join(repoDir, viper.GetString("db"))

	// Ensure db directory exists
	err := os.MkdirAll(dbDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "ask.db")

	// Remove existing database file to ensure clean recreation
	if _, err := os.Stat(dbPath); err == nil {
		err = os.Remove(dbPath)
		if err != nil {
			return "", fmt.Errorf("failed to remove existing database: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create airports table
	createAirportsTableSQL := `CREATE TABLE IF NOT EXISTS airports (
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
	);`

	_, err = db.Exec(createAirportsTableSQL)
	if err != nil {
		return "", fmt.Errorf("failed to create airports table: %w", err)
	}

	// Create countries table
	createCountriesTableSQL := `CREATE TABLE IF NOT EXISTS countries (
		id INTEGER,
		code TEXT,
		name TEXT,
		continent TEXT,
		wikipedia_link TEXT,
		keywords TEXT
	);`

	_, err = db.Exec(createCountriesTableSQL)
	if err != nil {
		return "", fmt.Errorf("failed to create countries table: %w", err)
	}

	// Create import_status table to track git commit information
	createImportStatusTableSQL := `CREATE TABLE IF NOT EXISTS import_status (
		table_name TEXT PRIMARY KEY,
		last_import_date TEXT,
		git_commit_hash TEXT,
		git_commit_date TEXT,
		record_count INTEGER
	);`

	_, err = db.Exec(createImportStatusTableSQL)
	if err != nil {
		return "", fmt.Errorf("failed to create import_status table: %w", err)
	}

	fmt.Printf("Database created at: %s\n", dbPath)
	return dbPath, nil
}
