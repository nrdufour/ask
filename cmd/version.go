package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ASK - Airport Swiss Knife",
	Long:  `All software has versions. This is ASK's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Airport Swiss Knife v0.1 -- HEAD")
	},
}
