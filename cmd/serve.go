/*
Copyright © 2024 Nicolas Dufour
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ask/repository"
	"ask/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
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

	port := viper.GetInt("server.port")
	srv := server.NewServer(port)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
