package api

import (
	"testing"

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
