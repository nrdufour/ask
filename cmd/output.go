package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"ask/db"
	"ask/repository"
	"ask/store"

	"github.com/spf13/cobra"
)

var jsonOutput bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print results as JSON, as the HTTP API returns them")
}

// openStore opens the local database, explaining how to create it when missing.
func openStore() (*store.Store, error) {
	if !repository.IsRepositoryDirectoryExists() {
		return nil, errors.New("no local data yet: run `ask init` first")
	}
	st, err := store.Open(db.Path())
	if err != nil {
		return nil, fmt.Errorf("opening the database (run `ask init` to rebuild it): %w", err)
	}
	return st, nil
}

// withStore runs fn against the local database, as a cobra RunE.
func withStore(fn func(cmd *cobra.Command, st *store.Store, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// The arguments parsed; from here on, errors are about the data, and
		// the usage text would only bury them.
		cmd.SilenceUsage = true
		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		return fn(cmd, st, args)
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func place(a store.Airport) string {
	var parts []string
	for _, p := range []string{a.Municipality, a.IsoCountry} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

func typeLabel(t string) string {
	return strings.ReplaceAll(t, "_", " ")
}

// thousands formats n with comma separators: 3150 -> "3,150".
func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
