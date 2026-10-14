/*
Copyright © 2024 Nicolas Dufour
*/
package repository

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func IsRepositoryDirectoryExists() bool {
	_, err := os.Stat(viper.GetString("repository"))
	return !errors.Is(err, os.ErrNotExist)
}

// RetrieveDataFromGit clones the ourairports.com data repository, or pulls it
// when a clone is already present.
func RetrieveDataFromGit() error {
	dataDir := filepath.Join(viper.GetString("repository"), viper.GetString("data"))
	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(dataDir, ".git")); err == nil {
		r, err := git.PlainOpen(dataDir)
		if err != nil {
			return err
		}
		w, err := r.Worktree()
		if err != nil {
			return err
		}
		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return err
	}

	_, err := git.PlainClone(dataDir, false, &git.CloneOptions{
		URL:      "https://github.com/davidmegginson/ourairports-data",
		Progress: os.Stdout,
		Depth:    1,
	})
	return err
}
