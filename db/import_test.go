package db

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := CreateSchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestImportCSVMatchesColumnsByHeader(t *testing.T) {
	db := newTestDB(t)

	// Columns out of schema order, one the schema doesn't know, and empty
	// fields: the import maps by name, skips the unknown column and stores
	// NULL for empties.
	csv := `name,ident,brand_new_column,elevation_ft,id,icao_code
Charles de Gaulle,LFPG,whatever,392,1,LFPG
"Quoted, with comma",XX-0001,,,2,
`
	n, err := ImportCSV(db, "airports", strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("imported %d rows, want 2", n)
	}

	var name string
	var elevation sql.NullInt64
	var icao sql.NullString
	if err := db.QueryRow(`SELECT name, elevation_ft, icao_code FROM airports WHERE ident = 'LFPG'`).Scan(&name, &elevation, &icao); err != nil {
		t.Fatal(err)
	}
	if name != "Charles de Gaulle" || elevation.Int64 != 392 || icao.String != "LFPG" {
		t.Errorf("LFPG row: %q, %v, %v", name, elevation, icao)
	}

	if err := db.QueryRow(`SELECT name, elevation_ft, icao_code FROM airports WHERE ident = 'XX-0001'`).Scan(&name, &elevation, &icao); err != nil {
		t.Fatal(err)
	}
	if name != "Quoted, with comma" || elevation.Valid || icao.Valid {
		t.Errorf("empty fields should be NULL: %q, %v, %v", name, elevation, icao)
	}
}

func TestImportCSVReplacesTable(t *testing.T) {
	db := newTestDB(t)

	for _, csv := range []string{"id,code,name\n1,FR,France\n2,CH,Switzerland\n", "id,code,name\n3,ZW,Zimbabwe\n"} {
		if _, err := ImportCSV(db, "countries", strings.NewReader(csv)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	var name string
	if err := db.QueryRow(`SELECT count(*), max(name) FROM countries`).Scan(&count, &name); err != nil {
		t.Fatal(err)
	}
	if count != 1 || name != "Zimbabwe" {
		t.Errorf("after reimport: %d rows, %q; want only Zimbabwe", count, name)
	}
}

func TestImportCSVErrors(t *testing.T) {
	db := newTestDB(t)

	tests := map[string]struct{ table, csv string }{
		"unknown table":   {"hangars", "id\n1\n"},
		"no known column": {"airports", "colour,size\nred,big\n"},
		"empty file":      {"airports", ""},
		"ragged row":      {"airports", "id,ident\n1,LFPG,extra\n"},
	}
	for name, tt := range tests {
		if _, err := ImportCSV(db, tt.table, strings.NewReader(tt.csv)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}

	// A failed import leaves the previous data in place.
	if _, err := ImportCSV(db, "countries", strings.NewReader("id,code,name\n1,FR,France\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportCSV(db, "countries", strings.NewReader("id,code,name\n2,CH,Switzerland\n3,ZW\n")); err == nil {
		t.Fatal("ragged reimport: no error")
	}
	var name string
	if err := db.QueryRow(`SELECT name FROM countries`).Scan(&name); err != nil || name != "France" {
		t.Errorf("after a failed reimport: %q, %v; want France kept", name, err)
	}
}

func TestEveryTableHasACSVFile(t *testing.T) {
	for _, table := range tableOrder {
		if csvFiles[table] == "" || columns[table] == nil {
			t.Errorf("table %s lacks a CSV file name or columns", table)
		}
	}
}
