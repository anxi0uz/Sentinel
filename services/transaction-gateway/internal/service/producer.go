package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(writer *kafka.Writer) *Producer {
	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, tx models.EnrichedTransaction) error {
	payload, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("marshal transaction: %w", err)
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.Transaction.ID.String()),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	slog.InfoContext(ctx, "transaction published", slog.String("id", tx.Transaction.ID.String()))
	return nil
}
