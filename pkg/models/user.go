package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Country     string
	LastIP      string
	LastCountry string
	LastSeenAt  time.Time
	CreatedAt   time.Time
}
