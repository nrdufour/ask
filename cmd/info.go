package cmd

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"ask/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info <code>",
	Short: "An airport's runways, frequencies and nearby navaids",
	Example: `  ask info LFPG
  ask info FR-0496 --json`,
	Args: cobra.ExactArgs(1),
	RunE: withStore(func(cmd *cobra.Command, st *store.Store, args []string) error {
		d, err := st.Detail(args[0])
		if err != nil {
			return err
		}
		if jsonOutput {
			return printJSON(d)
		}

		a := d.Airport
		where := []string{}
		for _, p := range []string{a.Municipality, d.RegionName, d.CountryName} {
			if p != "" {
				where = append(where, p)
			}
		}
		fmt.Printf("%s  %s\n%s\n", a.Code(), a.Name, strings.Join(where, ", "))
		fmt.Printf("%s", typeLabel(a.Type))
		if a.IataCode != "" {
			fmt.Printf(", IATA %s", a.IataCode)
		}
		if a.ElevationFt != nil {
			fmt.Printf(", elevation %s ft", thousands(*a.ElevationFt))
		}
		fmt.Printf("\n%.4f, %.4f\n", a.LatitudeDeg, a.LongitudeDeg)

		if len(d.Runways) > 0 {
			fmt.Println("\nRunways")
			w := newTable()
			for _, r := range d.Runways {
				ident := r.LowEnd.Ident
				if r.HighEnd.Ident != "" {
					ident += "/" + r.HighEnd.Ident
				}
				length := ""
				if r.LengthFt != nil {
					length = thousands(*r.LengthFt) + " ft"
				}
				notes := []string{r.SurfaceLabel}
				if r.Lighted {
					notes = append(notes, "lighted")
				}
				if r.Closed {
					notes = append(notes, "closed")
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", ident, length, headings(r), strings.Join(notes, ", "))
			}
			w.Flush()
		}

		if len(d.Frequencies) > 0 {
			fmt.Println("\nFrequencies")
			w := newTable()
			for _, f := range d.Frequencies {
				fmt.Fprintf(w, "  %s\t%s\t%s\n", f.Type, strconv.FormatFloat(f.MHz, 'f', 3, 64), f.Description)
			}
			w.Flush()
		}

		if len(d.Navaids) > 0 {
			fmt.Printf("\nNavaids within %d NM\n", store.NavaidRadiusNM)
			w := newTable()
			for _, n := range d.Navaids {
				freq := n.DMEChannel
				if n.FrequencyKHz != nil && n.Type != "DME" && n.Type != "TACAN" {
					if strings.HasPrefix(n.Type, "NDB") {
						freq = fmt.Sprintf("%d kHz", *n.FrequencyKHz)
					} else {
						freq = strconv.FormatFloat(float64(*n.FrequencyKHz)/1000, 'f', 2, 64) + " MHz"
					}
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%.1f NM\t%03.0f°\n", n.Ident, n.Name, n.Type, freq, n.DistanceNM, n.BearingDeg)
			}
			w.Flush()
		}
		return nil
	}),
}

// headings formats a runway's true headings, "085°/265°"; designator-based
// ones are magnetic and marked with ≈.
func headings(r store.Runway) string {
	var parts []string
	for _, e := range []store.RunwayEnd{r.LowEnd, r.HighEnd} {
		if e.HeadingDeg == nil {
			continue
		}
		h := fmt.Sprintf("%03.0f°", math.Mod(math.Round(*e.HeadingDeg)+360, 360))
		if e.HeadingSource == store.HeadingFromDesignator {
			h = "≈" + h
		}
		parts = append(parts, h)
	}
	return strings.Join(parts, "/")
}
