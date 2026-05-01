package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/sgaunet/kubetrainer/internal/consumer"
	"github.com/sgaunet/kubetrainer/internal/server"
	"github.com/sgaunet/kubetrainer/pkg/config"
)

const (
	stopChanBufferSize      = 5
	consumerShutdownTimeout = 120 * time.Second
	serverShutdownTimeout   = 15 * time.Second
)

// errRedisRequired is returned when consumer mode lacks Redis configuration.
var errRedisRequired = errors.New("redis configuration is required for consumer mode")

// errShutdownTimeout is returned when the consumer fails to shutdown within the deadline.
var errShutdownTimeout = errors.New("graceful shutdown timed out, forcing exit")

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		configurationFileName string
		consumerMode          bool
	)
	flag.StringVar(&configurationFileName, "f", "", "Configuration file")
	flag.BoolVar(&consumerMode, "consumer", false, "Run in consumer mode")
	flag.Parse()

	cfg, err := loadConfig(configurationFileName)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	stopChan := make(chan os.Signal, stopChanBufferSize)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	if consumerMode {
		return runConsumer(cfg, stopChan)
	}
	return runWebServer(cfg, stopChan)
}

func loadConfig(filename string) (*config.Config, error) {
	if len(filename) == 0 {
		cfg, err := config.LoadConfigFromEnv()
		if err != nil {
			return nil, fmt.Errorf("loading config from env: %w", err)
		}
		return cfg, nil
	}
	cfg, err := config.LoadConfigFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("loading config from file: %w", err)
	}
	return cfg, nil
}

func runConsumer(cfg *config.Config, stopChan <-chan os.Signal) error {
	if !cfg.IsRedisConfig() {
		return errRedisRequired
	}

	redisClient, err := initRedisConnection(cfg.RedisCfg.RedisDSN)
	if err != nil {
		return fmt.Errorf("error during redis initialization: %w", err)
	}
	defer func() {
		if cerr := redisClient.Close(); cerr != nil {
			log.Printf("error closing redis client: %v\n", cerr)
		}
	}()

	c := consumer.NewConsumer(
		redisClient,
		os.Getenv("REDIS_STREAMNAME"),
		os.Getenv("REDIS_STREAMGROUP"),
		cfg.ProducerCfg.DataSizeBytes,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.InitConsumer(ctx); err != nil {
		return fmt.Errorf("error initializing consumer: %w", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		consumeLoop(ctx, c)
	})

	<-stopChan
	fmt.Println("\nShutting down consumer...")
	return waitForConsumerShutdown(cancel, &wg)
}

func consumeLoop(ctx context.Context, c *consumer.Consumer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.Consume(ctx); err != nil {
				log.Printf("Error consuming message: %v\n", err)
			}
		}
	}
}

func waitForConsumerShutdown(cancel context.CancelFunc, wg *sync.WaitGroup) error {
	shutdownDone := make(chan struct{})
	go func() {
		cancel()
		wg.Wait()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		fmt.Println("Consumer shutdown gracefully.")
		return nil
	case <-time.After(consumerShutdownTimeout):
		return errShutdownTimeout
	}
}

func runWebServer(cfg *config.Config, stopChan <-chan os.Signal) error {
	wOpts, err := buildWebServerOptions(cfg)
	if err != nil {
		return err
	}

	w := server.NewWebServer(wOpts...)

	serverErr := make(chan error, 1)
	go func() {
		if serr := w.Start(); serr != nil && !errors.Is(serr, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("error starting server: %w", serr)
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return err
		}
	case <-stopChan:
		fmt.Println("\nShutting down server...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	if err := w.Shutdown(ctx); err != nil {
		return fmt.Errorf("error during shutdown: %w", err)
	}
	return nil
}

func buildWebServerOptions(cfg *config.Config) ([]server.WebServerOption, error) {
	var wOpts []server.WebServerOption
	if cfg.IsRedisConfig() {
		redisClient, err := initRedisConnection(cfg.RedisCfg.RedisDSN)
		if err != nil {
			return nil, fmt.Errorf("error during redis initialization: %w", err)
		}
		wOpts = append(wOpts,
			server.WithRedisClient(redisClient),
			server.WithStreamName(os.Getenv("REDIS_STREAMNAME")),
		)
	}
	if cfg.IsDBConfig() {
		pg, err := initDB(cfg)
		if err != nil {
			return nil, fmt.Errorf("error during database initialization: %w", err)
		}
		wOpts = append(wOpts, server.WithDB(pg))
	}
	return wOpts, nil
}
