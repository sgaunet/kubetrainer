package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/sgaunet/kubetrainer/internal/database"
	htmlhandlers "github.com/sgaunet/kubetrainer/internal/html/handlers"
)

const (
	httpListenPort    = 3000
	readHeaderTimeout = 10 * time.Second
)

// WebServer wires together the HTTP router, controllers, and dependencies.
type WebServer struct {
	srv               *http.Server
	router            *chi.Mux
	htmlController    *htmlhandlers.Controller
	db                database.Database
	rdb               *redis.Client
	streamName        string
	consumerGroupName string
}

// WebServerOption mutates a WebServer during construction.
type WebServerOption func(*WebServer)

// WithRedisClient injects the Redis client used by handlers.
func WithRedisClient(rdb *redis.Client) WebServerOption {
	return func(w *WebServer) {
		w.rdb = rdb
	}
}

// WithDB injects the database used by handlers.
func WithDB(db database.Database) WebServerOption {
	return func(w *WebServer) {
		w.db = db
	}
}

// WithStreamName overrides the Redis stream name.
func WithStreamName(streamName string) WebServerOption {
	return func(w *WebServer) {
		w.streamName = streamName
	}
}

// WithConsumerGroupName overrides the Redis consumer group name.
func WithConsumerGroupName(consumerGroupName string) WebServerOption {
	return func(w *WebServer) {
		w.consumerGroupName = consumerGroupName
	}
}

// NewWebServer constructs a configured WebServer.
func NewWebServer(opts ...WebServerOption) *WebServer {
	w := &WebServer{
		streamName:        os.Getenv("REDIS_STREAMNAME"),
		consumerGroupName: os.Getenv("REDIS_STREAMGROUP"),
	}
	w.router = chi.NewRouter()
	w.router.Use(middleware.Logger)
	w.router.Use(middleware.Recoverer)
	w.srv = &http.Server{
		Addr:              fmt.Sprintf(":%d", httpListenPort),
		Handler:           w.router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	for _, opt := range opts {
		opt(w)
	}

	w.htmlController = htmlhandlers.NewController(w.db, w.rdb, w.streamName, w.consumerGroupName)
	w.PublicRoutes()

	return w
}

// Start begins serving HTTP traffic. It blocks until the server stops.
func (w *WebServer) Start() error {
	if err := w.srv.ListenAndServe(); err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (w *WebServer) Shutdown(ctx context.Context) error {
	if err := w.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	return nil
}
