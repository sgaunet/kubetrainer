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

type Controller struct {
	livenessState     atomic.Bool
	readinessState    atomic.Bool
	db                database.Database
	rdb               *redis.Client
	producer          *producer.Producer
	consumerGroupName string
}

// NewController creates a new controller
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

// UpdateLivenessState updates the liveness state
func (h *Controller) UpdateLivenessState() {
	state := h.livenessState.Load()
	h.livenessState.Store(!state)
}

func (h *Controller) UpdateReadinessState() {
	state := h.readinessState.Load()
	h.readinessState.Store(!state)
}

func (h *Controller) GetPendingMessagesCount(ctx context.Context) int64 {
	count, err := h.producer.GetPendingMessagesCount(ctx, h.consumerGroupName)
	if err != nil {
		fmt.Println("Error getting pending messages count:", err)
		return 0
	}
	return count
}

func (h *Controller) ChangeLivenessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateLivenessState()
	count := h.GetPendingMessagesCount(r.Context())
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), count, "").Render(r.Context(), w)
}

func (h *Controller) ChangeReadinessState(w http.ResponseWriter, r *http.Request) {
	h.UpdateReadinessState()
	count := h.GetPendingMessagesCount(r.Context())
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), count, "").Render(r.Context(), w)
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

// PublishTime publishes current time to Redis stream
func (h *Controller) PublishTime(w http.ResponseWriter, r *http.Request) {
	countParamStr := chi.URLParam(r, "count")
	countParam, _ := strconv.Atoi(countParamStr)
	if countParam < 1 {
		countParam = 1
	}
	for i := 0; i < int(countParam); i++ {
		currentTime := time.Now().Format(time.RFC3339)
		err := h.producer.Publish(r.Context(), currentTime)
		if err != nil {
			count := h.GetPendingMessagesCount(r.Context())
			views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), count, err.Error()).Render(r.Context(), w)
			return
		}
	}
	count := h.GetPendingMessagesCount(r.Context())
	views.IndexPage(h.livenessState.Load(), h.readinessState.Load(), h.db.IsConnected(), h.IsRedisConnected(), count, "").Render(r.Context(), w)
}
