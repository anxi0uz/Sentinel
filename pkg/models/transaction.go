package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Amount    float64
	Currency  string
	IP        string
	Country   string
	Timestamp time.Time
}
