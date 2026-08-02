package scorer

import (
	"reflect"
	"testing"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
)

func TestScore(t *testing.T) {
	tx := models.EnrichedTransaction{
		Transaction: models.Transaction{Amount: 75_000, Country: "KP"},
	}
	rules := []models.FraudRule{
		{Name: "high_amount", Field: "amount", Operator: "gt", Threshold: 50_000, ScoreDelta: 40, Active: true},
		{Name: "blocked_country", Field: "country", Operator: "eq", Values: []string{"KP"}, ScoreDelta: 60, Active: true},
		{Name: "disabled", Field: "country", Operator: "eq", Values: []string{"KP"}, ScoreDelta: 100, Active: false},
	}

	score, triggered := Score(rules, tx)
	if score != 100 {
		t.Fatalf("expected score 100, got %d", score)
	}
	if want := []string{"high_amount", "blocked_country"}; !reflect.DeepEqual(triggered, want) {
		t.Fatalf("expected triggered rules %v, got %v", want, triggered)
	}
}

func TestAppliesRejectsUnknownFields(t *testing.T) {
	tx := models.EnrichedTransaction{}
	tests := []models.FraudRule{
		{Field: "missing", Operator: "lt", Threshold: 1},
		{Field: "missing", Operator: "not_in", Values: []string{"value"}},
		{Field: "country", Operator: "eq"},
	}
	for _, rule := range tests {
		if applies(rule, tx) {
			t.Fatalf("rule with operator %q and field %q unexpectedly applied", rule.Operator, rule.Field)
		}
	}
}

func TestImpossibleTravel(t *testing.T) {
	lastSeen := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	base := models.EnrichedTransaction{
		Transaction: models.Transaction{Country: "FI", Timestamp: lastSeen.Add(30 * time.Minute)},
		User:        models.User{LastCountry: "SE", LastSeenAt: lastSeen},
	}

	tests := []struct {
		name string
		tx   models.EnrichedTransaction
		want bool
	}{
		{name: "different country too soon", tx: base, want: true},
		{name: "same country", tx: withCountry(base, "SE"), want: false},
		{name: "enough time elapsed", tx: withTimestamp(base, lastSeen.Add(3*time.Hour)), want: false},
		{name: "out of order event", tx: withTimestamp(base, lastSeen.Add(-time.Minute)), want: false},
		{name: "missing history", tx: withLastSeen(base, time.Time{}), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := impossibleTravel(tt.tx, 2); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func withCountry(tx models.EnrichedTransaction, country string) models.EnrichedTransaction {
	tx.Transaction.Country = country
	return tx
}

func withTimestamp(tx models.EnrichedTransaction, timestamp time.Time) models.EnrichedTransaction {
	tx.Transaction.Timestamp = timestamp
	return tx
}

func withLastSeen(tx models.EnrichedTransaction, timestamp time.Time) models.EnrichedTransaction {
	tx.User.LastSeenAt = timestamp
	return tx
}
