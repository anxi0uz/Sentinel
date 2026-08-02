package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID        uuid.UUID `json:"ID"`
	UserID    uuid.UUID `json:"UserID"`
	Amount    float64   `json:"Amount"`
	Currency  string    `json:"Currency"`
	IP        string    `json:"IP"`
	Country   string    `json:"Country"`
	Timestamp time.Time `json:"Timestamp"`
}
