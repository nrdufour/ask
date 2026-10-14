package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"ask/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rangeCmd)
	rangeCmd.Flags().StringSliceP("type", "t", []string{"large_airport", "medium_airport"},
		"airport types to include: "+strings.Join(store.AirportTypes, ", "))
	rangeCmd.Flags().Bool("all-types", false, "include every airport type")
}

var rangeCmd = &cobra.Command{
	Use:   "range <origin> <nautical miles>",
	Short: "List the airports within a distance of an origin, nearest first",
	Example: `  ask range LFPG 200
  ask range ENSB 1000 --type large_airport,medium_airport,small_airport
  ask range NFFN 700 --all-types`,
	Args: cobra.ExactArgs(2),
	RunE: withStore(func(cmd *cobra.Command, st *store.Store, args []string) error {
		rangeNM, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return fmt.Errorf("%q is not a number of nautical miles", args[1])
		}
		types, _ := cmd.Flags().GetStringSlice("type")
		if all, _ := cmd.Flags().GetBool("all-types"); all {
			types = nil
		}

		result, err := st.InRange(args[0], rangeNM, types)
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(result)
		}

		o := result.OriginAirport
		fmt.Printf("%d airports within %s NM of %s, %s\n\n", result.Count, strconv.FormatFloat(rangeNM, 'f', -1, 64), o.Code(), o.Name)
		if result.Count == 0 {
			fmt.Println("Widen the range, or include more types with --type or --all-types.")
			return nil
		}
		w := newTable()
		fmt.Fprintln(w, "NM\tCODE\tNAME\tPLACE\tTYPE")
		for _, ra := range result.Airports {
			a := ra.Airport
			fmt.Fprintf(w, "%.1f\t%s\t%s\t%s\t%s\n", ra.DistanceNM, a.Code(), a.Name, place(a), typeLabel(a.Type))
		}
		return w.Flush()
	}),
}
