package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type Event struct {
	ID      uuid.UUID
	Topic   string
	Key     string
	Payload []byte
}

// Enqueue stores an event in the caller's database transaction. The unique
// topic/key pair makes retries idempotent.
func Enqueue(ctx context.Context, tx pgx.Tx, event Event) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (id, topic, event_key, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (topic, event_key) DO NOTHING`,
		event.ID, event.Topic, event.Key, event.Payload,
	)
	if err != nil {
		return fmt.Errorf("enqueue outbox event: %w", err)
	}
	return nil
}

type MessageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

type Publisher struct {
	db           *pgxpool.Pool
	writer       MessageWriter
	pollInterval time.Duration
	retryDelay   time.Duration
}

func NewPublisher(db *pgxpool.Pool, writer MessageWriter, pollInterval time.Duration) *Publisher {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &Publisher{
		db:           db,
		writer:       writer,
		pollInterval: pollInterval,
		retryDelay:   5 * time.Second,
	}
}

func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	p.drain(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.drain(ctx)
		}
	}
}

func (p *Publisher) drain(ctx context.Context) {
	for ctx.Err() == nil {
		published, err := p.publishNext(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "failed to publish outbox event", slog.String("error", err.Error()))
			return
		}
		if !published {
			return
		}
	}
}

func (p *Publisher) publishNext(ctx context.Context) (bool, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin outbox transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var event Event
	err = tx.QueryRow(ctx, `
		SELECT id, topic, event_key, payload
		FROM outbox_events
		WHERE published_at IS NULL AND next_attempt_at <= now()
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`).Scan(&event.ID, &event.Topic, &event.Key, &event.Payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("select outbox event: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: event.Topic,
		Key:   []byte(event.Key),
		Value: event.Payload,
	})
	if err != nil {
		if _, updateErr := tx.Exec(ctx, `
			UPDATE outbox_events
			SET attempts = attempts + 1, last_error = $2, next_attempt_at = $3
			WHERE id = $1`, event.ID, err.Error(), time.Now().Add(p.retryDelay)); updateErr != nil {
			return false, fmt.Errorf("publish event: %v; record failure: %w", err, updateErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return false, fmt.Errorf("publish event: %v; commit failure state: %w", err, commitErr)
		}
		return false, fmt.Errorf("write event to kafka: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = now(), attempts = attempts + 1, last_error = NULL
		WHERE id = $1`, event.ID); err != nil {
		return false, fmt.Errorf("mark outbox event published: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit published outbox event: %w", err)
	}
	return true, nil
}
