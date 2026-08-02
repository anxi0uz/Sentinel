package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnrichedTransactionDecodesLegacyKafkaPayload(t *testing.T) {
	payload := []byte(`{
		"Transaction": {
			"ID": "10000000-0000-0000-0000-000000000001",
			"UserID": "20000000-0000-0000-0000-000000000001",
			"Amount": 42.5,
			"Currency": "EUR",
			"IP": "1.2.3.4",
			"Country": "FI",
			"Timestamp": "2026-08-02T10:30:00Z"
		},
		"User": {
			"ID": "20000000-0000-0000-0000-000000000001",
			"Country": "SE",
			"LastIP": "5.6.7.8",
			"LastCountry": "SE",
			"LastSeenAt": "2026-08-02T10:00:00Z",
			"CreatedAt": "2026-01-01T00:00:00Z"
		}
	}`)

	var event EnrichedTransaction
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.Transaction.UserID != uuid.MustParse("20000000-0000-0000-0000-000000000001") {
		t.Fatalf("transaction user ID was lost: %s", event.Transaction.UserID)
	}
	if event.User.LastCountry != "SE" || event.User.LastIP != "5.6.7.8" {
		t.Fatalf("user history was lost: %+v", event.User)
	}
	if !event.User.LastSeenAt.Equal(time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("last seen = %s", event.User.LastSeenAt)
	}
}

func TestScoredTransactionUsesStableLegacyKafkaFieldNames(t *testing.T) {
	event := ScoredTransactionEvent{
		Transaction:    Transaction{ID: uuid.MustParse("10000000-0000-0000-0000-000000000001")},
		Score:          100,
		TriggeredRules: []string{"high_amount"},
		ProcessedAt:    time.Date(2026, 8, 2, 10, 30, 0, 0, time.UTC),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, field := range []string{`"Transaction"`, `"Score"`, `"TriggeredRules"`, `"ProcessedAt"`} {
		if !strings.Contains(body, field) {
			t.Fatalf("payload %s does not contain legacy field %s", body, field)
		}
	}
	if strings.Contains(body, `"triggered_rules"`) || strings.Contains(body, `"processed_at"`) {
		t.Fatalf("payload unexpectedly changed Kafka contract: %s", body)
	}
}
