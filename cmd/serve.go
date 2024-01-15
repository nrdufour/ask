/*
Copyright © 2024 Nicolas Dufour
*/
package cmd

import (
	"fmt"
	"os"

	"ask/repository"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a http server",
	Long:  `Start a http server for remote queries`,
	Run: func(cmd *cobra.Command, args []string) {
		doServe()
	},
}

func doServe() {
	if !repository.IsRepositoryDirectoryExists() {
		fmt.Println("Warning! the repository doesn't exist!")
		fmt.Println("Please, set it up with the `init` command")
		os.Exit(1)
	}
}
