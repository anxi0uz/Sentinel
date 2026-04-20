package models

type EnrichedTransaction struct {
	Transaction Transaction
	User        User
}
