package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const alertThreshold = 80

type ScoredRepository interface {
	Save(context.Context, models.ScoredTransactionEvent) (bool, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Save persists the scored transaction and, when necessary, its alert and
// outbox event in one transaction. It returns false for an already processed
// transaction.
func (r *PostgresRepository) Save(ctx context.Context, event models.ScoredTransactionEvent) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin alert transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	scoredID := uuid.New()
	err = tx.QueryRow(ctx, `
		INSERT INTO scored_transactions (id, transaction_id, score, triggered_rules, processed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (transaction_id) DO NOTHING
		RETURNING id`,
		scoredID,
		event.Transaction.ID,
		event.Score,
		event.TriggeredRules,
		event.ProcessedAt,
	).Scan(&scoredID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit duplicate transaction: %w", err)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert scored transaction: %w", err)
	}

	severity, needsAlert := severityForScore(event.Score)
	if needsAlert {
		alert := models.Alert{
			ID:                  uuid.New(),
			ScoredTransactionID: scoredID,
			Severity:            severity,
			CreatedAt:           time.Now().UTC(),
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO alerts (id, scored_transaction_id, severity, created_at)
			VALUES ($1, $2, $3, $4)`,
			alert.ID, alert.ScoredTransactionID, alert.Severity, alert.CreatedAt,
		); err != nil {
			return false, fmt.Errorf("insert alert: %w", err)
		}

		payload, err := json.Marshal(alert)
		if err != nil {
			return false, fmt.Errorf("marshal alert: %w", err)
		}
		if err := outbox.Enqueue(ctx, tx, outbox.Event{
			ID:      uuid.New(),
			Topic:   "alerts",
			Key:     alert.ID.String(),
			Payload: payload,
		}); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit scored transaction: %w", err)
	}
	return true, nil
}

func severityForScore(score int) (models.Severity, bool) {
	switch {
	case score >= 120:
		return models.SeverityCritical, true
	case score >= 90:
		return models.SeverityHigh, true
	case score >= alertThreshold:
		return models.SeverityMedium, true
	default:
		return "", false
	}
}
