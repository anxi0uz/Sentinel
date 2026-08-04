package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/google/uuid"
)

func TestValidateTransactionRequest(t *testing.T) {
	valid := TransactionRequest{
		UserId:   uuid.New(),
		Amount:   10,
		Currency: "EUR",
		Ip:       "127.0.0.1",
		Country:  "FI",
	}
	tests := []struct {
		name   string
		mutate func(*TransactionRequest)
		valid  bool
	}{
		{name: "valid", mutate: func(*TransactionRequest) {}, valid: true},
		{name: "missing user", mutate: func(r *TransactionRequest) { r.UserId = uuid.Nil }},
		{name: "non-positive amount", mutate: func(r *TransactionRequest) { r.Amount = 0 }},
		{name: "invalid currency", mutate: func(r *TransactionRequest) { r.Currency = "EU" }},
		{name: "missing ip", mutate: func(r *TransactionRequest) { r.Ip = " " }},
		{name: "invalid country", mutate: func(r *TransactionRequest) { r.Country = "FIN" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := valid
			tt.mutate(&request)
			err := validateTransactionRequest(request)
			if tt.valid && err != nil {
				t.Fatalf("expected valid request, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestTransactionEventBuildsDashboardReadModel(t *testing.T) {
	transactionID := uuid.New()
	userID := uuid.New()
	snapshot := models.EnrichedTransaction{
		Transaction: models.Transaction{
			ID:        transactionID,
			UserID:    userID,
			Amount:    2450,
			Currency:  "EUR",
			IP:        "203.0.113.15",
			Country:   "FI",
			Timestamp: time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
		},
		User: models.User{
			ID:          userID,
			Country:     "FI",
			LastIP:      "198.51.100.7",
			LastCountry: "SE",
			LastSeenAt:  time.Date(2026, 8, 2, 11, 30, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
		},
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	scored := models.ScoredTransaction{
		ID:                 uuid.New(),
		TransactionID:      transactionID,
		Score:              95,
		TriggeredRules:     []string{"high_amount"},
		ProcessedAt:        time.Date(2026, 8, 2, 12, 0, 1, 0, time.UTC),
		TransactionPayload: payload,
	}
	alert := models.Alert{
		ID:                  uuid.New(),
		ScoredTransactionID: scored.ID,
		Severity:            models.SeverityHigh,
	}

	event, err := transactionEvent(scored, &alert, true)
	if err != nil {
		t.Fatal(err)
	}
	if event.Transaction == nil || event.Transaction.Id != transactionID || event.Transaction.Amount != 2450 {
		t.Fatalf("transaction = %+v", event.Transaction)
	}
	if event.User == nil || event.User.Id != userID || event.User.LastCountry != "SE" {
		t.Fatalf("user = %+v", event.User)
	}
	if event.Severity == nil || *event.Severity != HIGH {
		t.Fatalf("severity = %v", event.Severity)
	}
	if event.DeliveryStatus != PUBLISHED {
		t.Fatalf("delivery status = %q, want %q", event.DeliveryStatus, PUBLISHED)
	}
}

func TestTransactionEventWithoutAlertNeedsNoDelivery(t *testing.T) {
	scored := models.ScoredTransaction{
		TransactionID:  uuid.New(),
		TriggeredRules: []string{},
		ProcessedAt:    time.Now(),
	}

	event, err := transactionEvent(scored, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if event.Severity != nil || event.AlertId != nil {
		t.Fatalf("unexpected alert fields: %+v", event)
	}
	if event.DeliveryStatus != NOTREQUIRED {
		t.Fatalf("delivery status = %q, want %q", event.DeliveryStatus, NOTREQUIRED)
	}
}

func TestValidateListTransactionsParams(t *testing.T) {
	limit, offset, err := validateListTransactionsParams(ListTransactionsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if limit != 50 || offset != 0 {
		t.Fatalf("defaults = (%d, %d), want (50, 0)", limit, offset)
	}

	zero := 0
	tooMany := 101
	negative := -1
	invalidSeverity := Severity("LOW")
	for _, params := range []ListTransactionsParams{
		{Limit: &zero},
		{Limit: &tooMany},
		{Offset: &negative},
		{MinScore: &negative},
		{Severity: &invalidSeverity},
	} {
		if _, _, err := validateListTransactionsParams(params); err == nil {
			t.Fatalf("params %+v: expected validation error", params)
		}
	}
}
