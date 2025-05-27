package consumer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
)

type Consumer struct {
	rdb               *redis.Client
	streamName        string
	consumerGroupName string
	dataSizeBytes    int64
}

func NewConsumer(redisClient *redis.Client, streamName string, consumerGroupName string, dataSizeBytes int64) *Consumer {
	return &Consumer{
		rdb:               redisClient,
		streamName:        streamName,
		consumerGroupName: consumerGroupName,
		dataSizeBytes:    dataSizeBytes,
	}
}

func (c *Consumer) InitConsumer(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.streamName, c.consumerGroupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// GenerateRandomData writes the specified amount of random data to the provided io.Writer.
// It returns any error encountered during the write operation.
func GenerateRandomData(w io.Writer, sizeBytes int64) error {
	if sizeBytes <= 0 {
		return fmt.Errorf("size must be greater than 0")
	}

	const bufferSize = 32 * 1024 // 32KB buffer size
	buffer := make([]byte, bufferSize)
	var bytesWritten int64 = 0
	totalBytes := sizeBytes

	for bytesWritten < totalBytes {
		// Fill buffer with random data
		_, err := rand.Read(buffer)
		if err != nil {
			return fmt.Errorf("error generating random data: %w", err)
		}

		// Calculate how many bytes to write in this iteration
		remaining := totalBytes - bytesWritten
		writeSize := int64(bufferSize)
		if remaining < int64(bufferSize) {
			writeSize = remaining
		}

		// Write the data
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

	// Copy the reader content to the hasher
	_, err := io.Copy(hasher, r)
	if err != nil {
		return "", fmt.Errorf("error reading data for hashing: %w", err)
	}

	// Get the final hash sum and encode it to hex
	hashSum := hasher.Sum(nil)
	return hex.EncodeToString(hashSum), nil
}

// SimulateWork performs CPU and I/O intensive work by generating random data
// and calculating its SHA256 checksum. It returns the checksum and any error encountered.
// The amount of data generated is determined by the provided sizeBytes parameter.
func SimulateWork(ctx context.Context, sizeBytes int64) (string, error) {
	// Create a pipe to connect the writer and reader
	pr, pw := io.Pipe()

	// Channel to collect the results
	hashChan := make(chan string, 1)
	errChan := make(chan error, 2)

	// Start a goroutine to generate the data and write it to the pipe
	go func() {
		defer pw.Close()
		err := GenerateRandomData(pw, sizeBytes)
		if err != nil {
			errChan <- fmt.Errorf("error generating random data: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Start a goroutine to calculate the hash as data is being written
	go func() {
		hash, err := CalculateSHA256(pr)
		if err != nil {
			errChan <- fmt.Errorf("error calculating hash: %w", err)
			return
		}
		hashChan <- hash
	}()

	// Wait for error, hash, or context cancellation
	for {
		select {
		case <-ctx.Done():
			pr.Close()
			pw.Close()
			return "", ctx.Err()
		case err := <-errChan:
			if err != nil {
				pr.Close() // Ensure the reader is closed on error
				return "", fmt.Errorf("error in data generation: %w", err)
			}
			// Wait for hash result
			select {
			case hash := <-hashChan:
				pr.Close() // Close the reader after we've got the hash
				return hash, nil
			case <-ctx.Done():
				pr.Close()
				pw.Close()
				return "", ctx.Err()
			}
		}
	}
}

func (c *Consumer) claimStuckMessages(ctx context.Context, uniqueID string) error {
	// Consider messages stuck if idle for more than 1 minute (60000 ms)
	const minIdleTimeMs = 60000
	const batchSize = 10

	pendingResult := c.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   c.streamName,
		Group:    c.consumerGroupName,
		Count:    batchSize,
		Idle:     minIdleTimeMs,
		Start:    "-",
		End:      "+",
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
	if len(claimed) > 0 {
		for _, msg := range claimed {
			fmt.Printf("Claimed stuck message ID %s: %v\n", msg.ID, msg.Values)
		}
	}
	return nil
}

func (c *Consumer) Consume(ctx context.Context) error {
	uniqueID := xid.New().String()

	// Helper to process messages
	processMessages := func(messages []redis.XMessage, msgType string) error {
		for _, msg := range messages {
			fmt.Printf("Processing %s message ID %s: %v\n", msgType, msg.ID, msg.Values)
			_, err := SimulateWork(ctx, c.dataSizeBytes)
			if err != nil {
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

	// Handle pending messages
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
			if err := processMessages(entries[0].Messages, "pending"); err != nil {
				return err
			}
		}
	}

	// Set up a ticker to check for stuck messages every minute
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// Read new messages in a loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.claimStuckMessages(ctx, uniqueID); err != nil {
				fmt.Printf("Error claiming stuck messages: %v\n", err)
			}
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
				if err := processMessages(entries[0].Messages, "new"); err != nil {
					return err
				}
			}
		}
	}
}

