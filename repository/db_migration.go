/*
Copyright © 2024 Nicolas Dufour
*/
package repository

import ( // "errors"
	// "fmt"
	// "os"
	// "github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func EnsureDbDirExists() {
	repoDir := viper.GetString("repository")
	dbDir := repoDir + "/" + viper.GetString("db")

	EnsureDirectoryExists(dbDir)
}

// Ensure the db exists

// Ensure the db has the latest data
