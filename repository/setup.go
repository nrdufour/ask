package repository

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Test if the repository directory exists
func IsRepositoryDirectoryExists() bool {
	path := viper.GetString("repository")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

// Ensure the repository directory exists (create it if not)
func EnsureRepositoryDirectoryExists() {
	path := viper.GetString("repository")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Repository directory " + path + " successfully created!")
	}
}

// Ensure the repository has a copy of ourairport git repo cloned
