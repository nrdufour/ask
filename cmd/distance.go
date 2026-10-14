package cmd

import (
	"fmt"
	"math"

	"ask/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(distanceCmd)
}

var distanceCmd = &cobra.Command{
	Use:     "distance <from> <to>",
	Short:   "Great-circle distance and initial true course between two airports",
	Example: `  ask distance LFPG KJFK`,
	Args:    cobra.ExactArgs(2),
	RunE: withStore(func(cmd *cobra.Command, st *store.Store, args []string) error {
		r, err := st.Route(args[0], args[1])
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(r)
		}

		dep, dest := r.DepartureAirport, r.DestinationAirport
		w := newTable()
		fmt.Fprintf(w, "From\t%s\t%s, %s\n", dep.Code(), dep.Name, place(dep))
		fmt.Fprintf(w, "To\t%s\t%s, %s\n", dest.Code(), dest.Name, place(dest))
		w.Flush()
		fmt.Printf("\n%s NM  (%s km, %s statute miles)\n",
			thousands(int(math.Round(r.DistanceNM))), thousands(int(math.Round(r.DistanceKM))), thousands(int(math.Round(r.DistanceSM))))
		fmt.Printf("Initial true course %03.0f°\n", math.Mod(math.Round(r.InitialBearingDeg), 360))
		return nil
	}),
}
