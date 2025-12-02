package producer

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Producer struct {
	rdb             *redis.Client
	maxStreamLength int
	streamName      string
}

func NewProducer(redisClient *redis.Client, streamName string) *Producer {
	return &Producer{
		rdb:             redisClient,
		streamName:      streamName,
		maxStreamLength: 1000, // Default max stream length
	}
}

func (p *Producer) Publish(ctx context.Context, message string) error {
	// Check if Redis client is initialized
	if p.rdb == nil {
		return errors.New("redis client is not initialized")
	}

	err := p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: p.streamName,
		MaxLen: int64(p.maxStreamLength),
		ID:     "",
		Values: map[string]interface{}{
			"msg": message,
		},
	}).Err()

	if err != nil {
		return errors.New("an event has not been written to the redis stream")
	}
	return nil
}

func (p *Producer) GetPendingMessagesCount(ctx context.Context, consumerGroupName string) (int64, error) {
	// Check if Redis client is initialized
	if p.rdb == nil {
		return 0, nil
	}

	// Get consumer group information
	groups, err := p.rdb.XInfoGroups(ctx, p.streamName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get consumer groups: %w", err)
	}

	// Find target consumer group
	var targetGroup *redis.XInfoGroup
	for _, g := range groups {
		if g.Name == consumerGroupName {
			targetGroup = &g
			break
		}
	}
	if targetGroup == nil {
		return 0, fmt.Errorf("consumer group %q not found", consumerGroupName)
	}

	// Handle undelivered messages (never read by group)
	var undeliveredCount int64
	lastID := targetGroup.LastDeliveredID
	if lastID == "0-0" { // Group has never read messages
		undeliveredCount, err = p.rdb.XLen(ctx, p.streamName).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to get stream length: %w", err)
		}
	} else { // Get messages after last delivered ID
		messages, err := p.rdb.XRange(ctx, p.streamName, "("+lastID, "+").Result()
		if err != nil {
			return 0, fmt.Errorf("failed to get undelivered messages: %w", err)
		}
		undeliveredCount = int64(len(messages))
	}

	// Get pending messages (delivered but unacknowledged)
	pending, err := p.rdb.XPending(ctx, p.streamName, consumerGroupName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending messages: %w", err)
	}

	return pending.Count + undeliveredCount, nil
}
