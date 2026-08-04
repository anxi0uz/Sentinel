package models

import (
	"time"

	"github.com/google/uuid"
)

type ScoredTransaction struct {
	ID                 uuid.UUID `db:"id"`
	TransactionID      uuid.UUID `db:"transaction_id"`
	Score              int       `db:"score"`
	TriggeredRules     []string  `db:"triggered_rules"`
	ProcessedAt        time.Time `db:"processed_at"`
	TransactionPayload []byte    `db:"transaction_payload"`
}

type ScoredTransactionEvent struct {
	Transaction    Transaction `json:"Transaction"`
	User           User        `json:"User"`
	Score          int         `json:"Score"`
	TriggeredRules []string    `json:"TriggeredRules"`
	ProcessedAt    time.Time   `json:"ProcessedAt"`
}
