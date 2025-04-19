package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/sgaunet/kubetrainer/internal/database"
	htmlhandlers "github.com/sgaunet/kubetrainer/internal/html/handlers"
)

type WebServer struct {
	srv            *http.Server
	router         *chi.Mux
	htmlController *htmlhandlers.Controller
	db             database.Database
	rdb            *redis.Client
	streamName     string
}

type WebServerOption func(*WebServer)

func WithRedisClient(rdb *redis.Client) WebServerOption {
	return func(w *WebServer) {
		w.rdb = rdb
	}
}

func WithDB(db database.Database) WebServerOption {
	return func(w *WebServer) {
		w.db = db
	}
}

func WithStreamName(streamName string) WebServerOption {
	return func(w *WebServer) {
		w.streamName = streamName
	}
}

func NewWebServer(opts ...WebServerOption) *WebServer {
	// Create a new WebServer instance
	w := &WebServer{
		streamName: os.Getenv("REDIS_STREAMNAME"), // Default stream name from env
	}
	// Create middlewares, router, and server
	w.router = chi.NewRouter()
	// add common middlewares
	w.router.Use(middleware.Logger)
	w.router.Use(middleware.Recoverer)
	w.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", 3000),
		Handler: w.router,
	}

	// Setup options
	for _, opt := range opts {
		opt(w)
	}

	// initialize html and json controllers
	w.htmlController = htmlhandlers.NewController(w.db, w.rdb, w.streamName)
	// Setup routes
	w.PublicRoutes()

	return w
}

func (w *WebServer) Start() error {
	return w.srv.ListenAndServe()
}

func (w *WebServer) Shutdown(ctx context.Context) error {
	return w.srv.Shutdown(ctx)
}
