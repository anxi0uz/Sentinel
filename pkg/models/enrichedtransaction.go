package models

type EnrichedTransaction struct {
	Transaction Transaction `json:"Transaction"`
	User        User        `json:"User"`
}
