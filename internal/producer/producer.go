// Package producer publishes messages to a Redis stream and exposes
// pending message metrics for kubetrainer.
package producer

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const defaultMaxStreamLength = 1000

var (
	errRedisNotInitialized   = errors.New("redis client is not initialized")
	errStreamWriteFailed     = errors.New("an event has not been written to the redis stream")
	errConsumerGroupNotFound = errors.New("consumer group not found")
)

// Producer publishes messages to a Redis stream.
type Producer struct {
	rdb             *redis.Client
	maxStreamLength int
	streamName      string
}

// NewProducer creates a Producer that publishes to streamName.
func NewProducer(redisClient *redis.Client, streamName string) *Producer {
	return &Producer{
		rdb:             redisClient,
		streamName:      streamName,
		maxStreamLength: defaultMaxStreamLength,
	}
}

// Publish writes a single message to the Redis stream.
func (p *Producer) Publish(ctx context.Context, message string) error {
	if p.rdb == nil {
		return errRedisNotInitialized
	}

	err := p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: p.streamName,
		MaxLen: int64(p.maxStreamLength),
		ID:     "",
		Values: map[string]any{
			"msg": message,
		},
	}).Err()
	if err != nil {
		return errStreamWriteFailed
	}
	return nil
}

// GetPendingMessagesCount returns the number of pending and undelivered messages
// for the given consumer group.
func (p *Producer) GetPendingMessagesCount(ctx context.Context, consumerGroupName string) (int64, error) {
	if p.rdb == nil {
		return 0, nil
	}

	groups, err := p.rdb.XInfoGroups(ctx, p.streamName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get consumer groups: %w", err)
	}

	var targetGroup *redis.XInfoGroup
	for _, g := range groups {
		if g.Name == consumerGroupName {
			targetGroup = &g
			break
		}
	}
	if targetGroup == nil {
		return 0, fmt.Errorf("%w: %q", errConsumerGroupNotFound, consumerGroupName)
	}

	var undeliveredCount int64
	lastID := targetGroup.LastDeliveredID
	if lastID == "0-0" {
		undeliveredCount, err = p.rdb.XLen(ctx, p.streamName).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to get stream length: %w", err)
		}
	} else {
		messages, err := p.rdb.XRange(ctx, p.streamName, "("+lastID, "+").Result()
		if err != nil {
			return 0, fmt.Errorf("failed to get undelivered messages: %w", err)
		}
		undeliveredCount = int64(len(messages))
	}

	pending, err := p.rdb.XPending(ctx, p.streamName, consumerGroupName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending messages: %w", err)
	}

	return pending.Count + undeliveredCount, nil
}
