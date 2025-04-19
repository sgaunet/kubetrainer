package handlers

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
	"github.com/sgaunet/kubetrainer/internal/database"
	"github.com/sgaunet/kubetrainer/internal/html/views"
)

type Controller struct {
	livenessState  atomic.Bool
	readinessState atomic.Bool
	db             database.Database
	rdb            *redis.Client
}

// NewController creates a new controller
func NewController(db database.Database, rdb *redis.Client) *Controller {
	c := &Controller{
		db:  db,
		rdb: rdb,
	}
	c.livenessState.Store(true)
	c.readinessState.Store(true)
	return c
}

// UpdateLivenessState updates the liveness state
func (h *Controller) UpdateLivenessState() {
	state := h.livenessState.Load()
	h.livenessState.Store(!state)
}

func (h *Controller) UpdateReadinessState() {
	state := h.readinessState.Load()
	h.readinessState.Store(!state)
}

func (h *Controller) ChangeLivenessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateLivenessState()
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), "").Render(context.Background(), w)
}

func (h *Controller) ChangeReadinessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateReadinessState()
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), "").Render(context.Background(), w)
}

func (h *Controller) Readiness(w http.ResponseWriter, r *http.Request) {
	state := h.readinessState.Load()
	if !state {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready yet"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Controller) Liveness(w http.ResponseWriter, r *http.Request) {
	state := h.livenessState.Load()
	if !state {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not live"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// IsRedisConnected checks if Redis is connected
func (h *Controller) IsRedisConnected() bool {
	if h.rdb == nil {
		return false
	}
	err := h.rdb.Ping(context.Background()).Err()
	return err == nil
}
