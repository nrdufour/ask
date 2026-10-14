package cmd

import (
	"fmt"
	"time"

	"ask/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(timeCmd)
}

var timeCmd = &cobra.Command{
	Use:     "time <code>...",
	Short:   "Local time and timezone at airports",
	Example: `  ask time LFPG KJFK RJTT`,
	Args:    cobra.MinimumNArgs(1),
	RunE: withStore(func(cmd *cobra.Command, st *store.Store, args []string) error {
		var times []*store.AirportTime
		for _, code := range args {
			t, err := st.LocalTime(code)
			if err != nil {
				return err
			}
			times = append(times, t)
		}
		if jsonOutput {
			if len(times) == 1 {
				return printJSON(times[0])
			}
			return printJSON(times)
		}

		w := newTable()
		for i, t := range times {
			local, err := time.Parse(time.RFC3339, t.LocalTime)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "%s\t%s\tUTC%s\t%s\t%s\n", args[i], local.Format("Mon 15:04"), t.UTCOffset, t.Timezone, t.Name)
		}
		return w.Flush()
	}),
}
