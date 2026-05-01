// Package handlers contains the HTTP controllers serving kubetrainer's HTML views.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sgaunet/kubetrainer/internal/database"
	"github.com/sgaunet/kubetrainer/internal/html/views"
	"github.com/sgaunet/kubetrainer/internal/producer"
)

// Controller serves HTML views and tracks liveness/readiness state.
type Controller struct {
	livenessState     atomic.Bool
	readinessState    atomic.Bool
	db                database.Database
	rdb               *redis.Client
	producer          *producer.Producer
	consumerGroupName string
}

// NewController creates a new controller.
func NewController(db database.Database, rdb *redis.Client, streamName string, consumerGroupName string) *Controller {
	c := &Controller{
		db:                db,
		rdb:               rdb,
		producer:          producer.NewProducer(rdb, streamName),
		consumerGroupName: consumerGroupName,
	}
	c.livenessState.Store(true)
	c.readinessState.Store(true)
	return c
}

// UpdateLivenessState toggles the liveness state.
func (h *Controller) UpdateLivenessState() {
	state := h.livenessState.Load()
	h.livenessState.Store(!state)
}

// UpdateReadinessState toggles the readiness state.
func (h *Controller) UpdateReadinessState() {
	state := h.readinessState.Load()
	h.readinessState.Store(!state)
}

// GetPendingMessagesCount returns the count of pending messages for the configured consumer group.
func (h *Controller) GetPendingMessagesCount(ctx context.Context) int64 {
	if h.producer == nil {
		return 0
	}
	count, err := h.producer.GetPendingMessagesCount(ctx, h.consumerGroupName)
	if err != nil {
		fmt.Println("Error getting pending messages count:", err)
		return 0
	}
	return count
}

// ChangeLivenessState toggles liveness then renders the index page.
func (h *Controller) ChangeLivenessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateLivenessState()
	count := h.GetPendingMessagesCount(r.Context())
	if err := views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(r.Context()), count, "").Render(r.Context(), w); err != nil {
		fmt.Println("error rendering index page:", err)
	}
}

// ChangeReadinessState toggles readiness then renders the index page.
func (h *Controller) ChangeReadinessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateReadinessState()
	count := h.GetPendingMessagesCount(r.Context())
	if err := views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(r.Context()), count, "").Render(r.Context(), w); err != nil {
		fmt.Println("error rendering index page:", err)
	}
}

// Readiness returns the current readiness state as an HTTP status.
func (h *Controller) Readiness(w http.ResponseWriter, _ *http.Request) {
	state := h.readinessState.Load()
	if !state {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte("not ready yet")); err != nil {
			fmt.Println("error writing readiness response:", err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		fmt.Println("error writing readiness response:", err)
	}
}

// Liveness returns the current liveness state as an HTTP status.
func (h *Controller) Liveness(w http.ResponseWriter, _ *http.Request) {
	state := h.livenessState.Load()
	if !state {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte("not live")); err != nil {
			fmt.Println("error writing liveness response:", err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		fmt.Println("error writing liveness response:", err)
	}
}

// IsDBConnected checks if database is connected.
func (h *Controller) IsDBConnected() bool {
	if h.db == nil {
		return false
	}
	return h.db.IsConnected()
}

// IsRedisConnected checks if Redis is connected.
func (h *Controller) IsRedisConnected(ctx context.Context) bool {
	if h.rdb == nil {
		return false
	}
	err := h.rdb.Ping(ctx).Err()
	return err == nil
}

// PublishTime publishes current time to Redis stream.
func (h *Controller) PublishTime(w http.ResponseWriter, r *http.Request) {
	countParamStr := chi.URLParam(r, "count")
	countParam, _ := strconv.Atoi(countParamStr)
	if countParam < 1 {
		countParam = 1
	}

	if h.producer == nil {
		count := h.GetPendingMessagesCount(r.Context())
		if err := views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(r.Context()), count, "Redis is not configured").Render(r.Context(), w); err != nil {
			fmt.Println("error rendering index page:", err)
		}
		return
	}

	for range countParam {
		currentTime := time.Now().Format(time.RFC3339)
		err := h.producer.Publish(r.Context(), currentTime)
		if err != nil {
			count := h.GetPendingMessagesCount(r.Context())
			if renderErr := views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(r.Context()), count, err.Error()).Render(r.Context(), w); renderErr != nil {
				fmt.Println("error rendering index page:", renderErr)
			}
			return
		}
	}
	count := h.GetPendingMessagesCount(r.Context())
	if err := views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.IsDBConnected(), h.IsRedisConnected(r.Context()), count, "").Render(r.Context(), w); err != nil {
		fmt.Println("error rendering index page:", err)
	}
}
