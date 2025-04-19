package handlers

import (
	"context"
	"net/http"

	"github.com/sgaunet/kubetrainer/internal/html/views"
)

func (h *Controller) Index(w http.ResponseWriter, r *http.Request) {
	var postgresState bool
	if h.db != nil && h.db.GetDB() != nil {
		err := h.db.GetDB().Ping()
		postgresState = err == nil
	}
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), postgresState, "").Render(context.Background(), w)
}
