package generator

import (
	"testing"
	"time"
)

func TestNewEventBuildsObviousFraudScenario(t *testing.T) {
	now := time.Date(2026, 8, 2, 14, 0, 0, 0, time.UTC)
	event := NewEvent(ScenarioObviousFraud, now)

	if event.Transaction.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("transaction ID was not generated")
	}
	if event.User.ID != event.Transaction.UserID {
		t.Fatalf("user ID %s does not match transaction user ID %s", event.User.ID, event.Transaction.UserID)
	}
	if event.Transaction.Amount <= 50000 {
		t.Fatalf("amount = %.2f, want an amount above the high_amount threshold", event.Transaction.Amount)
	}
	if event.Transaction.Country != "KP" {
		t.Fatalf("country = %q, want KP", event.Transaction.Country)
	}
	if !event.Transaction.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", event.Transaction.Timestamp, now)
	}
}

func TestNewEventBuildsNormalScenario(t *testing.T) {
	event := NewEvent(ScenarioNormal, time.Now().UTC())

	if event.Transaction.Amount >= 50000 {
		t.Fatalf("normal amount = %.2f", event.Transaction.Amount)
	}
	if event.Transaction.Country == "KP" {
		t.Fatal("normal event unexpectedly uses blocked country")
	}
}
