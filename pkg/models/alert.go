package models

import (
	"time"

	"github.com/google/uuid"
)

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type Alert struct {
	ID                  uuid.UUID `db:"id"`
	ScoredTransactionID uuid.UUID `db:"scored_transaction_id"`
	Severity            Severity  `db:"severity"`
	CreatedAt           time.Time `db:"created_at"`
}
