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

// Generate1GBOfRandomData writes 1GB of random data to the provided io.Writer.
// It returns any error encountered during the write operation.
func Generate1GBOfRandomData(w io.Writer) error {
	// 1GB = 1024 * 1024 * 1024 bytes
	const bufferSize = 32 * 1024 // 32KB buffer size
	buffer := make([]byte, bufferSize)
	bytesWritten := 0
	totalBytes := 1024 * 1024 * 1024 // 1GB

	for bytesWritten < totalBytes {
		// Fill buffer with random data
		_, err := rand.Read(buffer)
		if err != nil {
			return fmt.Errorf("error generating random data: %w", err)
		}

		// Calculate how many bytes to write in this iteration
		remaining := totalBytes - bytesWritten
		writeSize := bufferSize
		if remaining < bufferSize {
			writeSize = remaining
		}

		// Write the data
		n, err := w.Write(buffer[:writeSize])
		bytesWritten += n
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

// SimulateWork performs CPU and I/O intensive work by generating 1GB of random data
// and calculating its SHA256 checksum. It returns the checksum and any error encountered.
func SimulateWork() (string, error) {
	// Create a pipe to connect the writer and reader
	pr, pw := io.Pipe()
	defer pr.Close()

	// Channel to collect the hash result
	hashChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start a goroutine to calculate the hash as data is being written
	go func() {
		hash, err := CalculateSHA256(pr)
		if err != nil {
			errChan <- fmt.Errorf("error calculating hash: %w", err)
			return
		}
		hashChan <- hash
	}()

	// Generate 1GB of random data and write it to the pipe
	err := Generate1GBOfRandomData(pw)
	if err != nil {
		pw.Close() // Ensure the pipe is closed in case of error
		return "", fmt.Errorf("error generating random data: %w", err)
	}

	// Close the writer to signal EOF to the reader
	if err := pw.Close(); err != nil {
		return "", fmt.Errorf("error closing pipe writer: %w", err)
	}

	// Wait for the hash calculation to complete
	select {
	case hash := <-hashChan:
		return hash, nil
	case err := <-errChan:
		return "", err
	}
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
				// Simulate work
				_, err := SimulateWork()
				if err != nil {
					return fmt.Errorf("error simulating work: %w", err)
				}
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
