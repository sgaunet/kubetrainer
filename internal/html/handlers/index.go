package handlers

import (
	"fmt"
	"net/http"

	"github.com/sgaunet/kubetrainer/internal/html/views"
)

// Index renders the application's status dashboard.
func (h *Controller) Index(w http.ResponseWriter, r *http.Request) {
	count := h.GetPendingMessagesCount(r.Context())
	if err := views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(r.Context()), count, "").Render(r.Context(), w); err != nil {
		fmt.Println("error rendering index page:", err)
	}
}
