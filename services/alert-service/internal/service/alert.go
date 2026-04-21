package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/pkg/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type AlertService struct {
	reader *kafka.Reader
	writer *kafka.Writer
	db     *pgxpool.Pool
}

func NewAlertService(reader *kafka.Reader, writer *kafka.Writer, db *pgxpool.Pool) *AlertService {
	return &AlertService{
		reader: reader,
		writer: writer,
		db:     db,
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
			continue
		}

		var sc models.ScoredTransactionEvent
		if err := json.Unmarshal(msg.Value, &sc); err != nil {
			slog.ErrorContext(ctx, "error while unmarshal message", slog.String("error", err.Error()))
			if err := s.reader.CommitMessages(ctx, msg); err != nil {
				slog.ErrorContext(ctx, "error while commiting wrong message", slog.String("error", err.Error()))
			}
			continue
		}

		scoredtx := models.ScoredTransaction{
			ID:             uuid.New(),
			TransactionID:  sc.Transaction.ID,
			Score:          sc.Score,
			TriggeredRules: sc.TriggeredRules,
			ProcessedAt:    sc.ProcessedAt,
		}

		if err := storage.Create(ctx, "scored_transactions", scoredtx, s.db); err != nil {
			slog.ErrorContext(ctx, "error while persistance scored transaction", slog.String("error", err.Error()))
			continue
		}

		if scoredtx.Score >= 80 {
			var severity string
			if scoredtx.Score >= 80 && scoredtx.Score <= 89 {
				severity = "MEDIUM"
			}
			if scoredtx.Score >= 90 && scoredtx.Score <= 119 {
				severity = "HIGH"
			}
			if scoredtx.Score >= 120 {
				severity = "CRITICAL"
			}

			alert := models.Alert{
				ID:                  uuid.New(),
				ScoredTransactionID: scoredtx.ID,
				Severity:            models.Severity(severity),
				CreatedAt:           time.Now(),
			}
			if err := storage.Create(ctx, "alerts", alert, s.db); err != nil {
				slog.ErrorContext(ctx, "error while persistance alert", slog.String("error", err.Error()))
				continue
			}

			payload, err := json.Marshal(alert)
			if err != nil {
				slog.ErrorContext(ctx, "error while marshal alert", slog.String("error", err.Error()))
				continue
			}
			if err := s.writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(alert.ID.String()),
				Value: payload,
			}); err != nil {
				slog.ErrorContext(ctx, "error while writing message to kafka")
				continue
			}
		}
		if err := s.reader.CommitMessages(ctx, msg); err != nil {
			slog.ErrorContext(ctx, "error while commiting wrong message", slog.String("error", err.Error()))
		}
	}
}
