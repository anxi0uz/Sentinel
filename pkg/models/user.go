package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `db:"id"`
	Country     string    `db:"country"`
	LastIP      string    `db:"last_ip"`
	LastCountry string    `db:"last_country"`
	LastSeenAt  time.Time `db:"last_seen_at"`
	CreatedAt   time.Time `db:"created_at"`
}
