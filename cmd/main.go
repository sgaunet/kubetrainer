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
	"github.com/sgaunet/kubetrainer/internal/consumer"
	"github.com/sgaunet/kubetrainer/internal/server"
	"github.com/sgaunet/kubetrainer/pkg/config"
)

func main() {
	var (
		err                   error
		cfg                   *config.Config
		configurationFileName string
		wOpts                 []server.WebServerOption
		consumerMode          bool
	)
	// debug.SetMemoryLimit(1024 * 1024 * 1024 * 2)
	flag.StringVar(&configurationFileName, "f", "", "Configuration file")
	flag.BoolVar(&consumerMode, "consumer", false, "Run in consumer mode")
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

	// Create a channel to listen for OS signals
	stopChan := make(chan os.Signal, 5)
	// Notify the stopChan when an interrupt or terminate signal is received
	signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	if consumerMode {
		if !cfg.IsRedisConfig() {
			fmt.Fprintf(os.Stderr, "Redis configuration is required for consumer mode\n")
			os.Exit(1)
		}

		redisClient, err := initRedisConnection(cfg.RedisCfg.RedisDSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error during redis initialization: %s\n", err.Error())
			os.Exit(1)
		}
		defer redisClient.Close()

		// Create consumer with configurable data size
		c := consumer.NewConsumer(
			redisClient,
			os.Getenv("REDIS_STREAMNAME"),
			os.Getenv("REDIS_STREAMGROUP"),
			cfg.ProducerCfg.DataSizeBytes,
		)
		ctx := context.Background()

		// Initialize consumer group
		err = c.InitConsumer(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error initializing consumer: %s\n", err.Error())
			os.Exit(1)
		}

		// Start consuming in a goroutine
		go func() {
			for {
				err := c.Consume(ctx)
				if err != nil {
					log.Printf("Error consuming message: %v\n", err)
				}
			}
		}()

		// Wait for stop signal
		<-stopChan
		fmt.Println("\nShutting down consumer...")
		os.Exit(0)
	}

	// Web server mode
	if cfg.IsRedisConfig() {
		redisClient, err := initRedisConnection(cfg.RedisCfg.RedisDSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error during redis initialization: %s\n", err.Error())
			os.Exit(1)
		}
		wOpts = append(wOpts, server.WithRedisClient(redisClient))
		wOpts = append(wOpts, server.WithStreamName(os.Getenv("REDIS_STREAMNAME")))
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

	// Start the HTTP server in a goroutine
	go func() {
		// If ListenAndServe returns an error and it's not a server closed error,
		// then log it as a fatal error.
		if err := w.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error starting server: %s\n", err.Error())
		}
	}()

	// Wait for stop signal
	<-stopChan
	fmt.Println("\nShutting down server...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	if err := w.Shutdown(ctx); err != nil {
		fmt.Printf("error during shutdown: %s\n", err.Error())
	}
}
