package models

import "time"

type ScoredTransaction struct {
	Transaction    Transaction
	Score          int
	TriggeredRules []string
	ProcessedAt    time.Time
}
