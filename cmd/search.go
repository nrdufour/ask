package cmd

import (
	"fmt"
	"os"
	"strings"

	"ask/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringP("country", "c", "", "only airports in this country (ISO code, e.g. FR)")
	searchCmd.Flags().IntP("limit", "n", 20, fmt.Sprintf("maximum number of results (up to %d)", store.MaxSearchLimit))
}

var searchCmd = &cobra.Command{
	Use:   "search <name, city or code>",
	Short: "Find airports by name, city or code",
	Example: `  ask search heathrow
  ask search paris --country FR
  ask search JFK --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: withStore(func(cmd *cobra.Command, st *store.Store, args []string) error {
		country, _ := cmd.Flags().GetString("country")
		limit, _ := cmd.Flags().GetInt("limit")

		result, err := st.Search(strings.Join(args, " "), country, limit)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(result)
		}

		if result.Count == 0 {
			fmt.Println("No airport matches. Try part of the name, the city, or a 3- or 4-letter code.")
			return nil
		}
		w := newTable()
		fmt.Fprintln(w, "CODE\tIATA\tNAME\tPLACE\tTYPE")
		for _, a := range result.Airports {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", a.Code(), a.IataCode, a.Name, place(a), typeLabel(a.Type))
		}
		w.Flush()
		if result.Truncated {
			fmt.Fprintf(os.Stderr, "\nShowing the first %d matches. Narrow the search, or raise --limit.\n", result.Count)
		}
		return nil
	}),
}
