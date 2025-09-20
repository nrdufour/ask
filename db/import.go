/*
Copyright © 2024 Nicolas Dufour
*/
package db

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

// ImportAirportsCSV imports the airports.csv file into the database
func ImportAirportsCSV(dbPath string) error {
	repoDir := viper.GetString("repository")
	dataDir := filepath.Join(repoDir, viper.GetString("data"))
	csvPath := filepath.Join(dataDir, "airports.csv")

	// Check if CSV file exists
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return fmt.Errorf("airports.csv not found at %s", csvPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	fmt.Printf("CSV header: %v\n", header)

	// Clear existing data
	_, err = db.Exec("DELETE FROM airports")
	if err != nil {
		return fmt.Errorf("failed to clear existing data: %w", err)
	}

	// Prepare insert statement
	placeholders := strings.Repeat("?,", len(header)-1) + "?"
	insertSQL := fmt.Sprintf("INSERT INTO airports VALUES (%s)", placeholders)

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Begin transaction for better performance
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	recordCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		// Convert record to interface slice for SQL parameters
		args := make([]interface{}, len(record))
		for i, v := range record {
			args[i] = v
		}

		_, err = tx.Stmt(stmt).Exec(args...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert record: %w", err)
		}

		recordCount++
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported %d airport records\n", recordCount)
	return nil
}

// ImportCountriesCSV imports the countries.csv file into the database
func ImportCountriesCSV(dbPath string) error {
	repoDir := viper.GetString("repository")
	dataDir := filepath.Join(repoDir, viper.GetString("data"))
	csvPath := filepath.Join(dataDir, "countries.csv")

	// Check if CSV file exists
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return fmt.Errorf("countries.csv not found at %s", csvPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	fmt.Printf("CSV header: %v\n", header)

	// Clear existing data
	_, err = db.Exec("DELETE FROM countries")
	if err != nil {
		return fmt.Errorf("failed to clear existing data: %w", err)
	}

	// Prepare insert statement
	placeholders := strings.Repeat("?,", len(header)-1) + "?"
	insertSQL := fmt.Sprintf("INSERT INTO countries VALUES (%s)", placeholders)

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Begin transaction for better performance
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	recordCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		// Convert record to interface slice for SQL parameters
		args := make([]interface{}, len(record))
		for i, v := range record {
			args[i] = v
		}

		_, err = tx.Stmt(stmt).Exec(args...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert record: %w", err)
		}

		recordCount++
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully imported %d country records\n", recordCount)
	return nil
}
