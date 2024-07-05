package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	htmlhandlers "github.com/sgaunet/kubetrainer/internal/html/handlers"
)

type WebServer struct {
	srv            *http.Server
	router         *chi.Mux
	htmlController *htmlhandlers.Controller
}

func NewWebServer(db *sql.DB) (*WebServer, error) {
	// Create a new WebServer instance
	w := &WebServer{}
	// Create middlewares, router, and server
	w.router = chi.NewRouter()
	// add common middlewares
	w.router.Use(middleware.Logger)
	w.router.Use(middleware.Recoverer)
	w.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", 3000),
		Handler: w.router,
	}
	// initialize html and json controllers
	w.htmlController = htmlhandlers.NewController()
	// Setup routes
	w.PublicRoutes()
	return w, nil
}

func (w *WebServer) Start() error {
	return w.srv.ListenAndServe()
}

func (w *WebServer) Shutdown(ctx context.Context) error {
	return w.srv.Shutdown(ctx)
}
