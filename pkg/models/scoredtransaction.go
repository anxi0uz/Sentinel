package models

import (
	"time"

	"github.com/google/uuid"
)

type ScoredTransaction struct {
	ID             uuid.UUID `db:"id"`
	TransactionID  uuid.UUID `db:"transaction_id"`
	Score          int       `db:"score"`
	TriggeredRules []string  `db:"triggered_rules"`
	ProcessedAt    time.Time `db:"processed_at"`
}

type ScoredTransactionEvent struct {
	Transaction    Transaction
	Score          int
	TriggeredRules []string
	ProcessedAt    time.Time
}
