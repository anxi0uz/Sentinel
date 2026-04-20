package models

import "github.com/google/uuid"

type FraudRule struct {
	ID         uuid.UUID `db:"id"`
	Name       string    `db:"name"`
	Field      string    `db:"field"`
	Operator   string    `db:"operator"`
	Threshold  float64   `db:"threshold"`
	Values     []string  `db:"values"`
	ScoreDelta float64   `db:"score_delta"`
	Active     bool      `db:"active"`
}
