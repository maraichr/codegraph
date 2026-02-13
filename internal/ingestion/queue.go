package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

const (
	StreamName    = "codegraph:ingest"
	GroupName     = "codegraph-workers"
	MaxRetries    = 3
	ClaimTimeout  = 5 * time.Minute
)

// IngestMessage is the payload enqueued for worker processing.
type IngestMessage struct {
	IndexRunID uuid.UUID `json:"index_run_id"`
	ProjectID  uuid.UUID `json:"project_id"`
	SourceID   uuid.UUID `json:"source_id"`
	SourceType string    `json:"source_type"`
	Trigger    string    `json:"trigger"` // "manual", "webhook", "schedule"
}

// Producer enqueues ingestion jobs to the Valkey stream.
type Producer struct {
	client valkey.Client
}

func NewProducer(client valkey.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Enqueue(ctx context.Context, msg IngestMessage) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal message: %w", err)
	}

	resp := p.client.Do(ctx, p.client.B().Xadd().
		Key(StreamName).Id("*").
		FieldValue().FieldValue("data", string(data)).
		Build())
	if err := resp.Error(); err != nil {
		return "", fmt.Errorf("xadd: %w", err)
	}

	id, err := resp.ToString()
	if err != nil {
		return "", fmt.Errorf("parse xadd response: %w", err)
	}
	return id, nil
}

// Consumer reads ingestion jobs from the Valkey stream.
type Consumer struct {
	client     valkey.Client
	consumerID string
	logger     *slog.Logger
}

func NewConsumer(client valkey.Client, consumerID string, logger *slog.Logger) *Consumer {
	return &Consumer{client: client, consumerID: consumerID, logger: logger}
}

// EnsureGroup creates the consumer group if it doesn't exist.
func (c *Consumer) EnsureGroup(ctx context.Context) error {
	resp := c.client.Do(ctx, c.client.B().XgroupCreate().
		Key(StreamName).Group(GroupName).Id("0").Mkstream().Build())
	if err := resp.Error(); err != nil {
		// BUSYGROUP means group already exists — that's fine
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return fmt.Errorf("xgroup create: %w", err)
		}
	}
	return nil
}

// Consume blocks until a message is available, processes it via handler, and ACKs.
func (c *Consumer) Consume(ctx context.Context, handler func(context.Context, IngestMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp := c.client.Do(ctx, c.client.B().Xreadgroup().
			Group(GroupName, c.consumerID).
			Count(1).Block(5000).
			Streams().Key(StreamName).Id(">").
			Build())

		if err := resp.Error(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Timeout is normal for BLOCK reads
			continue
		}

		results, err := resp.AsXRead()
		if err != nil {
			continue
		}

		for _, messages := range results {
			for _, msg := range messages {
				dataStr, ok := msg.FieldValues["data"]
				if !ok {
					c.logger.Warn("message missing data field", slog.String("id", msg.ID))
					c.ack(ctx, msg.ID)
					continue
				}

				var ingestMsg IngestMessage
				if err := json.Unmarshal([]byte(dataStr), &ingestMsg); err != nil {
					c.logger.Error("unmarshal message", slog.String("error", err.Error()), slog.String("id", msg.ID))
					c.ack(ctx, msg.ID)
					continue
				}

				if err := handler(ctx, ingestMsg); err != nil {
					c.logger.Error("handle message", slog.String("error", err.Error()),
						slog.String("id", msg.ID),
						slog.String("index_run_id", ingestMsg.IndexRunID.String()))
					// Message stays pending for retry via XCLAIM
				} else {
					c.ack(ctx, msg.ID)
				}
			}
		}
	}
}

func (c *Consumer) ack(ctx context.Context, msgID string) {
	resp := c.client.Do(ctx, c.client.B().Xack().
		Key(StreamName).Group(GroupName).Id(msgID).Build())
	if err := resp.Error(); err != nil {
		c.logger.Error("xack failed", slog.String("error", err.Error()), slog.String("id", msgID))
	}
}
