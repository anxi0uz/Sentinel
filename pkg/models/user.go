package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `db:"id" json:"ID"`
	Country     string    `db:"country" json:"Country"`
	LastIP      string    `db:"last_ip" json:"LastIP"`
	LastCountry string    `db:"last_country" json:"LastCountry"`
	LastSeenAt  time.Time `db:"last_seen_at" json:"LastSeenAt"`
	CreatedAt   time.Time `db:"created_at" json:"CreatedAt"`
}
