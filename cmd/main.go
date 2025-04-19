package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/sgaunet/kubetrainer/internal/server"
	"github.com/sgaunet/kubetrainer/pkg/config"
)

func main() {
	var (
		err                   error
		cfg                   *config.Config
		configurationFileName string
		wOpts                 []server.WebServerOption
	)
	// debug.SetMemoryLimit(1024 * 1024 * 1024 * 2)
	flag.StringVar(&configurationFileName, "f", "", "Configuration file")
	flag.Parse()

	if len(configurationFileName) == 0 {
		cfg, err = config.LoadConfigFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot load configuration: %s\n", err.Error())
			os.Exit(1)
		}
	} else {
		cfg, err = config.LoadConfigFromFile(configurationFileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot load configuration: %s\n", err.Error())
			os.Exit(1)
		}
	}

	if cfg.IsRedisConfig() {
		redisClient, err := initRedisConnection(cfg.RedisCfg.RedisDSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error during redis initialization: %s\n", err.Error())
			os.Exit(1)
		}
		wOpts = append(wOpts, server.WithRedisClient(redisClient))
	}
	if cfg.IsDBConfig() {
		pg, err := initDB(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error during database initialization: %s\n", err.Error())
			os.Exit(1)
		}
		wOpts = append(wOpts, server.WithDB(pg))
	}
	// Initialize the web server with the options
	w := server.NewWebServer(wOpts...)

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
