package handlers

import (
	"net/http"

	"github.com/sgaunet/kubetrainer/internal/html/views"
)

func (h *Controller) Index(w http.ResponseWriter, r *http.Request) {
	count := h.GetPendingMessagesCount(r.Context())
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(), count, "").Render(r.Context(), w)
}
