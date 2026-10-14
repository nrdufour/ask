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
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func dataDir() string {
	return filepath.Join(viper.GetString("repository"), viper.GetString("data"))
}

// getGitCommitInfo retrieves the latest git commit information from the data directory
func getGitCommitInfo() (string, string, error) {
	repo, err := git.PlainOpen(dataDir())
	if err != nil {
		return "", "", fmt.Errorf("failed to open git repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", "", fmt.Errorf("failed to get commit object: %w", err)
	}

	return ref.Hash().String(), commit.Author.When.Format(time.RFC3339), nil
}

// importTable imports <table>.csv from the data directory and records the
// import in import_status.
func importTable(db *sql.DB, table, commitHash, commitDate string) error {
	file, err := os.Open(filepath.Join(dataDir(), csvFiles[table]))
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	count, err := ImportCSV(db, table, file)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT OR REPLACE INTO import_status
		(table_name, last_import_date, git_commit_hash, git_commit_date, record_count)
		VALUES (?, ?, ?, ?, ?)`,
		table, time.Now().Format(time.RFC3339), commitHash, commitDate, count)
	if err != nil {
		return fmt.Errorf("failed to update import status: %w", err)
	}

	fmt.Printf("Successfully imported %d %s records\n", count, table)
	return nil
}

// ImportCSV replaces the content of table with the rows of r. Columns are
// matched by header name, so CSV columns the schema doesn't know are skipped
// rather than breaking the import. Empty fields are stored as NULL.
func ImportCSV(db *sql.DB, table string, r io.Reader) (int, error) {
	known := columns[table]
	if known == nil {
		return 0, fmt.Errorf("unknown table %q", table)
	}

	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// For each schema column, the index of the matching CSV column.
	var cols []string
	var idx []int
	for i, name := range header {
		for _, k := range known {
			if name == k {
				cols = append(cols, name)
				idx = append(idx, i)
			}
		}
	}
	if len(cols) == 0 {
		return 0, fmt.Errorf("CSV header %v matches no %s column", header, table)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM " + table); err != nil {
		return 0, fmt.Errorf("failed to clear existing data: %w", err)
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",")
	stmt, err := tx.Prepare(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(cols, ","), placeholders))
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	args := make([]any, len(cols))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read CSV record: %w", err)
		}

		for i, j := range idx {
			if record[j] == "" {
				args[i] = nil
			} else {
				args[i] = record[j]
			}
		}

		if _, err := stmt.Exec(args...); err != nil {
			return 0, fmt.Errorf("failed to insert record: %w", err)
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return count, nil
}

// ImportDir imports, into tables already created, every table whose CSV file
// exists in dir (tables are named as in the database: runways.csv,
// frequencies.csv), without recording an import status. Tests load their
// fixtures with it.
func ImportDir(db *sql.DB, dir string) error {
	for _, table := range tableOrder {
		f, err := os.Open(filepath.Join(dir, table+".csv"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		_, err = ImportCSV(db, table, f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
	}
	return nil
}
