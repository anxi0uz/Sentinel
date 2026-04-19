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
	ID                  uuid.UUID
	ScoredTransactionID uuid.UUID
	Severity            Severity
	CreatedAt           time.Time
}
