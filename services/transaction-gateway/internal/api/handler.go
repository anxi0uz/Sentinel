package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"gihub.com/anxi0uz/sentinel/pkg/models"
	"gihub.com/anxi0uz/sentinel/services/transaction-gateway/internal/service"
	"github.com/google/uuid"
)

type Server struct {
	producer *service.Producer
}

func NewServer(producer *service.Producer) *Server {
	return &Server{producer: producer}
}

func (s *Server) SubmitTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req TransactionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "Error while decoding body", slog.String("error", err.Error()))
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	tx := models.Transaction{
		ID:        uuid.New(),
		UserID:    req.UserId,
		Amount:    req.Amount,
		Currency:  req.Currency,
		IP:        req.Ip,
		Country:   req.Country,
		Timestamp: time.Now(),
	}

	if err := s.producer.Publish(ctx, tx); err != nil {
		slog.ErrorContext(ctx, "error while publishing transaction", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	writeJSON(w, 202, TransactionResponse{Id: &tx.ID})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
