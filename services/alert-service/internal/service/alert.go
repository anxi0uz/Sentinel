package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type MessageReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
}

type AlertService struct {
	reader     MessageReader
	repository ScoredRepository
}

func NewAlertService(reader MessageReader, repository ScoredRepository) *AlertService {
	return &AlertService{
		reader:     reader,
		repository: repository,
	}
}

func (s *AlertService) Run(ctx context.Context) {
	for {
		msg, err := s.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.ErrorContext(ctx, "error while fetching message", slog.String("error", err.Error()))
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			continue
		}

		var sc models.ScoredTransactionEvent
		if err := json.Unmarshal(msg.Value, &sc); err != nil {
			slog.ErrorContext(ctx, "error while unmarshal message", slog.String("error", err.Error()))
			s.retry(ctx, "commit invalid scored message", func() error {
				return s.reader.CommitMessages(ctx, msg)
			})
			continue
		}
		if err := validateScoredEvent(sc); err != nil {
			slog.ErrorContext(ctx, "invalid scored message", slog.String("error", err.Error()))
			s.retry(ctx, "commit invalid scored message", func() error {
				return s.reader.CommitMessages(ctx, msg)
			})
			continue
		}

		var created bool
		if !s.retry(ctx, "persist scored transaction", func() error {
			var err error
			created, err = s.repository.Save(ctx, sc)
			return err
		}) {
			return
		}
		if !created {
			slog.InfoContext(ctx, "scored transaction already processed",
				slog.String("transaction_id", sc.Transaction.ID.String()))
		}
		if !s.retry(ctx, "commit scored message", func() error {
			return s.reader.CommitMessages(ctx, msg)
		}) {
			return
		}
	}
}

func validateScoredEvent(event models.ScoredTransactionEvent) error {
	if event.Transaction.ID == uuid.Nil {
		return fmt.Errorf("transaction id is required")
	}
	if event.ProcessedAt.IsZero() {
		return fmt.Errorf("processed_at is required")
	}
	return nil
}

func (s *AlertService) retry(ctx context.Context, operation string, fn func() error) bool {
	for {
		if err := fn(); err == nil {
			return true
		} else if ctx.Err() == nil {
			slog.ErrorContext(ctx, operation+" failed; retrying", slog.String("error", err.Error()))
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
}
