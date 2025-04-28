package consumer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
)

type Consumer struct {
	rdb               *redis.Client
	streamName        string
	consumerGroupName string
}

func NewConsumer(redisClient *redis.Client, streamName string, consumerGroupName string) *Consumer {
	return &Consumer{
		rdb:               redisClient,
		streamName:        streamName,
		consumerGroupName: consumerGroupName,
	}
}

func (c *Consumer) InitConsumer(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.streamName, c.consumerGroupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

func (c *Consumer) Consume(ctx context.Context) error {
	uniqueID := xid.New().String()

	// First, handle any pending messages
	pending := c.rdb.XPending(ctx, c.streamName, c.consumerGroupName).Val()
	if pending.Count > 0 {
		entries, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.consumerGroupName,
			Consumer: uniqueID,
			Streams:  []string{c.streamName, "0"}, // Read pending messages
			Count:    1,
		}).Result()
		if err != nil {
			return fmt.Errorf("error reading pending messages: %w", err)
		}
		if len(entries) > 0 && len(entries[0].Messages) > 0 {
			for _, msg := range entries[0].Messages {
				fmt.Printf("Processing pending message ID %s: %v\n", msg.ID, msg.Values)
				// Simulate work by sleeping for 10 seconds
				time.Sleep(10 * time.Second)
				if err := c.rdb.XAck(ctx, c.streamName, c.consumerGroupName, msg.ID).Err(); err != nil {
					return fmt.Errorf("error acknowledging message: %w", err)
				}
			}
		}
	}

	// Then read new messages
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			entries, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.consumerGroupName,
				Consumer: uniqueID,
				Streams:  []string{c.streamName, ">"}, // Read new messages
				Count:    1,
				Block:    0, // Block indefinitely
			}).Result()
			if err != nil {
				return fmt.Errorf("error reading new messages: %w", err)
			}

			if len(entries) > 0 && len(entries[0].Messages) > 0 {
				for _, msg := range entries[0].Messages {
					fmt.Printf("Processing new message ID %s: %v\n", msg.ID, msg.Values)
					// Simulate work by sleeping for 10 seconds
					time.Sleep(10 * time.Second)
					if err := c.rdb.XAck(ctx, c.streamName, c.consumerGroupName, msg.ID).Err(); err != nil {
						return fmt.Errorf("error acknowledging message: %w", err)
					}
				}
			}
		}
	}
}
