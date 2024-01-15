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
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the database",
	Long:  `Allow multiple types of queries on the database`,
	Run: func(cmd *cobra.Command, args []string) {
		doQuery()
	},
}

func doQuery() {
	if !repository.IsRepositoryDirectoryExists() {
		fmt.Println("Warning! the repository doesn't exist!")
		fmt.Println("Please, set it up with the `init` command")
		os.Exit(1)
	}
}
