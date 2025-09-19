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
	// First, ensure the repo directory exists
	repository.EnsureRepositoryDirExists()

	// Then, ensure the data sub dir exists
	repository.EnsureDataDirExists()

	// Get the data
	repository.RetrieveDataFromGit()

	// Initialize the database and import airport data
	err := db.InitializeDatabase()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
}
