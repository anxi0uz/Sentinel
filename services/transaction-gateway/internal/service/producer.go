package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gihub.com/anxi0uz/sentinel/pkg/models"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(writer *kafka.Writer) *Producer {
	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, tx models.Transaction) error {
	payload, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("marshal transaction: %w", err)
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.ID.String()),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	slog.InfoContext(ctx, "transaction published", slog.String("id", tx.ID.String()))
	return nil
}
