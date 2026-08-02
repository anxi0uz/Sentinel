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
	ID                  uuid.UUID `db:"id" json:"ID"`
	ScoredTransactionID uuid.UUID `db:"scored_transaction_id" json:"ScoredTransactionID"`
	Severity            Severity  `db:"severity" json:"Severity"`
	CreatedAt           time.Time `db:"created_at" json:"CreatedAt"`
}
