// Package consumer implements a Redis stream consumer that simulates
// CPU-intensive work for kubetrainer demos.
package consumer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
)

const (
	bufferSize         = 32 * 1024
	pipeChannelBuffer  = 2
	minIdleTimeMs      = 60_000
	stuckClaimBatch    = 10
)

// errInvalidSize is returned when a non-positive byte size is provided.
var errInvalidSize = errors.New("size must be greater than 0")

// Consumer reads messages from a Redis stream consumer group and performs
// simulated work for each message it processes.
type Consumer struct {
	rdb               *redis.Client
	streamName        string
	consumerGroupName string
	dataSizeBytes     int64
}

// NewConsumer creates a Consumer that reads from streamName using consumerGroupName.
func NewConsumer(redisClient *redis.Client, streamName string, consumerGroupName string, dataSizeBytes int64) *Consumer {
	return &Consumer{
		rdb:               redisClient,
		streamName:        streamName,
		consumerGroupName: consumerGroupName,
		dataSizeBytes:     dataSizeBytes,
	}
}

// InitConsumer creates the Redis stream consumer group if it does not exist.
func (c *Consumer) InitConsumer(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.streamName, c.consumerGroupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// Consume processes pending then new messages from the stream until ctx is canceled.
func (c *Consumer) Consume(ctx context.Context) error {
	uniqueID := xid.New().String()

	if err := c.processPendingBacklog(ctx, uniqueID); err != nil {
		return err
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("consumer context done: %w", ctx.Err())
		case <-ticker.C:
			if err := c.claimStuckMessages(ctx, uniqueID); err != nil {
				fmt.Printf("Error claiming stuck messages: %v\n", err)
			}
		default:
			if err := c.readAndProcessNew(ctx, uniqueID); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) processPendingBacklog(ctx context.Context, uniqueID string) error {
	pending := c.rdb.XPending(ctx, c.streamName, c.consumerGroupName).Val()
	if pending.Count == 0 {
		return nil
	}
	entries, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.consumerGroupName,
		Consumer: uniqueID,
		Streams:  []string{c.streamName, "0"},
		Count:    1,
	}).Result()
	if err != nil {
		return fmt.Errorf("error reading pending messages: %w", err)
	}
	if len(entries) == 0 || len(entries[0].Messages) == 0 {
		return nil
	}
	if err := c.processMessages(ctx, entries[0].Messages, "pending"); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) readAndProcessNew(ctx context.Context, uniqueID string) error {
	entries, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.consumerGroupName,
		Consumer: uniqueID,
		Streams:  []string{c.streamName, ">"},
		Count:    1,
		Block:    0,
	}).Result()
	if err != nil {
		return fmt.Errorf("error reading new messages: %w", err)
	}
	if len(entries) == 0 || len(entries[0].Messages) == 0 {
		return nil
	}
	if err := c.processMessages(ctx, entries[0].Messages, "new"); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) processMessages(ctx context.Context, messages []redis.XMessage, msgType string) error {
	for _, msg := range messages {
		fmt.Printf("Processing %s message ID %s: %v\n", msgType, msg.ID, msg.Values)
		if _, err := SimulateWork(ctx, c.dataSizeBytes); err != nil {
			fmt.Printf("Simulate work finished with error for message ID %s: %v\n", msg.ID, err)
			return fmt.Errorf("error simulating work: %w", err)
		}
		fmt.Printf("Simulate work finished successfully for message ID %s\n", msg.ID)
		if err := c.rdb.XAck(ctx, c.streamName, c.consumerGroupName, msg.ID).Err(); err != nil {
			return fmt.Errorf("error acknowledging message: %w", err)
		}
	}
	return nil
}

func (c *Consumer) claimStuckMessages(ctx context.Context, uniqueID string) error {
	pendingResult := c.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.streamName,
		Group:  c.consumerGroupName,
		Count:  stuckClaimBatch,
		Idle:   minIdleTimeMs,
		Start:  "-",
		End:    "+",
	}).Val()

	if len(pendingResult) == 0 {
		return nil
	}

	var idsToClaim []string
	for _, p := range pendingResult {
		if p.Consumer != uniqueID && p.Idle >= minIdleTimeMs {
			idsToClaim = append(idsToClaim, p.ID)
		}
	}
	if len(idsToClaim) == 0 {
		return nil
	}

	claimed, err := c.rdb.XClaim(ctx, &redis.XClaimArgs{
		Stream:   c.streamName,
		Group:    c.consumerGroupName,
		Consumer: uniqueID,
		MinIdle:  minIdleTimeMs,
		Messages: idsToClaim,
	}).Result()
	if err != nil {
		return fmt.Errorf("error claiming stuck messages: %w", err)
	}
	for _, msg := range claimed {
		fmt.Printf("Claimed stuck message ID %s: %v\n", msg.ID, msg.Values)
	}
	return nil
}

// GenerateRandomData writes the specified amount of random data to the provided io.Writer.
// It returns any error encountered during the write operation.
func GenerateRandomData(w io.Writer, sizeBytes int64) error {
	if sizeBytes <= 0 {
		return errInvalidSize
	}

	buffer := make([]byte, bufferSize)
	var bytesWritten int64
	totalBytes := sizeBytes

	for bytesWritten < totalBytes {
		if _, err := rand.Read(buffer); err != nil {
			return fmt.Errorf("error generating random data: %w", err)
		}

		remaining := totalBytes - bytesWritten
		writeSize := min(remaining, int64(bufferSize))

		n, err := w.Write(buffer[:writeSize])
		bytesWritten += int64(n)
		if err != nil {
			return fmt.Errorf("error writing random data: %w", err)
		}
	}

	return nil
}

// CalculateSHA256 computes the SHA256 checksum of the data read from the provided io.Reader.
// It returns the checksum as a hexadecimal-encoded string and any error encountered.
func CalculateSHA256(r io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("error reading data for hashing: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// SimulateWork performs CPU and I/O intensive work by generating random data
// and calculating its SHA256 checksum. It returns the checksum and any error encountered.
// The amount of data generated is determined by the provided sizeBytes parameter.
func SimulateWork(ctx context.Context, sizeBytes int64) (string, error) {
	pr, pw := io.Pipe()

	hashChan := make(chan string, 1)
	errChan := make(chan error, pipeChannelBuffer)

	go func() {
		defer func() {
			_ = pw.Close()
		}()
		if err := GenerateRandomData(pw, sizeBytes); err != nil {
			errChan <- fmt.Errorf("error generating random data: %w", err)
			return
		}
		errChan <- nil
	}()

	go func() {
		hash, err := CalculateSHA256(pr)
		if err != nil {
			errChan <- fmt.Errorf("error calculating hash: %w", err)
			return
		}
		hashChan <- hash
	}()

	for {
		select {
		case <-ctx.Done():
			_ = pr.Close()
			_ = pw.Close()
			return "", fmt.Errorf("simulate work canceled: %w", ctx.Err())
		case err := <-errChan:
			if err != nil {
				_ = pr.Close()
				return "", fmt.Errorf("error in data generation: %w", err)
			}
			select {
			case hash := <-hashChan:
				_ = pr.Close()
				return hash, nil
			case <-ctx.Done():
				_ = pr.Close()
				_ = pw.Close()
				return "", fmt.Errorf("simulate work canceled: %w", ctx.Err())
			}
		}
	}
}
