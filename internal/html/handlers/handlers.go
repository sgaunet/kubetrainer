package handlers

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/sgaunet/kubetrainer/internal/html/views"
)

type Controller struct {
	livenessState  atomic.Bool
	readinessState atomic.Bool
}

// NewController creates a new controller
func NewController() *Controller {
	c := &Controller{}
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
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), "").Render(context.Background(), w)
}

func (h *Controller) ChangeReadinessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateReadinessState()
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), "").Render(context.Background(), w)
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
