package producer

import (
	"context"
	"errors"

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
