/*
Copyright © 2024 Nicolas Dufour
*/
package cmd

import (
	"ask/db"
	"ask/repository"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the local database",
	Long:  `Will download the data and setup a local database`,
	Run: func(cmd *cobra.Command, args []string) {
		doInit()
	},
}

func doInit() {
	if err := repository.RetrieveDataFromGit(); err != nil {
		fmt.Printf("Error retrieving data: %v\n", err)
		os.Exit(1)
	}

	if err := db.InitializeDatabase(); err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
}
