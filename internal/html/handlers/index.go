package handlers

import (
	"context"
	"net/http"

	"github.com/sgaunet/kubetrainer/internal/html/views"
)

func (h *Controller) Index(w http.ResponseWriter, r *http.Request) {
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), "").Render(context.Background(), w)
}
