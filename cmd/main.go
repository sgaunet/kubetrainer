package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/sgaunet/kubetrainer/internal/server"
)

func main() {
	var (
		err error
		// cfg config.Config
		db *sql.DB
		// configurationFileName string
	)
	// debug.SetMemoryLimit(1024 * 1024 * 1024 * 2)
	// flag.StringVar(&configurationFileName, "f", "", "Configuration file")
	// flag.Parse()

	// if len(configurationFileName) == 0 {
	// 	fmt.Fprintf(os.Stderr, "no configuration file provided\n")
	// 	os.Exit(1)
	// }
	// cfg, err = config.LoadConfigFromFile(configurationFileName)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "cannot load configuration: %s\n", err.Error())
	// 	os.Exit(1)
	// }

	// pg, err := initDB(&cfg)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "error during database initialization: %s\n", err.Error())
	// 	os.Exit(1)
	// }

	// w, err := server.NewWebServer(cfg, pg.DB)
	w, err := server.NewWebServer(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error during webserver initialization: %s\n", err.Error())
		os.Exit(1)
	}

	// Create a channel to listen for OS signals
	stopChan := make(chan os.Signal, 5)
	// Notify the stopChan when an interrupt or terminate signal is received
	signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start the HTTP server in a goroutine
	go func() {
		// If ListenAndServe returns an error and it's not a server closed error,
		// then log it as a fatal error.
		if err := w.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	// Wait until we get a stop signal
	<-stopChan

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	// Make sure to cancel the context when done
	defer cancel()

	// Initiate graceful shutdown
	// If it doesn't complete in 15 seconds, it will be forcefully stopped
	if err := w.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	} else {
		log.Println("Server stopped gracefully")
	}
}
