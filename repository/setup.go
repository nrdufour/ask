/*
Copyright © 2024 Nicolas Dufour
*/
package repository

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Test if a directory exists
func IsDirectoryExists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func IsRepositoryDirectoryExists() bool {
	path := viper.GetString("repository")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

// Ensure a directory exists (create it if not)
func EnsureDirectoryExists(path string) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Directory " + path + " successfully created!")
	}
}

func EnsureRepositoryDirExists() {
	repoDir := viper.GetString("repository")

	EnsureDirectoryExists(repoDir)
}

// Ensure the repository has a copy of ourairport git repo cloned
func EnsureDataDirExists() {
	repoDir := viper.GetString("repository")
	dataDir := repoDir + "/" + viper.GetString("data")

	EnsureDirectoryExists(dataDir)
}

func RetrieveDataFromGit() {
	repoDir := viper.GetString("repository")
	dataDir := repoDir + "/" + viper.GetString("data")
	gitDir := dataDir + "/.git"

	if IsDirectoryExists(gitDir) {
		// do an update instead
		r, err := git.PlainOpen(dataDir)
		cobra.CheckErr(err)
		w, err := r.Worktree()
		cobra.CheckErr(err)
		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		cobra.CheckErr(err)
	} else {
		_, err := git.PlainClone(dataDir, false, &git.CloneOptions{
			URL:      "https://github.com/davidmegginson/ourairports-data",
			Progress: os.Stdout,
			Depth:    1,
		})
		cobra.CheckErr(err)
	}
}
