package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/pkg/storage"
	"github.com/anxi0uz/sentinel/services/transaction-gateway/internal/service"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	producer *service.Producer
	db       *pgxpool.Pool
}

func NewServer(producer *service.Producer, db *pgxpool.Pool) *Server {
	return &Server{producer: producer, db: db}
}

func (s *Server) SubmitTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req TransactionRequest

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		slog.ErrorContext(ctx, "Error while decoding body", slog.String("error", err.Error()))
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "request body must contain one JSON object"})
		return
	}
	if err := validateTransactionRequest(req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := storage.GetOne[models.User](ctx, s.db, "users", func(sb *sqlbuilder.SelectBuilder) {
		sb.Where(sb.EQ("id", req.UserId))
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			slog.ErrorContext(ctx, "no user with that transaction was found")
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		slog.ErrorContext(ctx, "Error while getting user", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
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

	enrtx := models.EnrichedTransaction{
		User:        *user,
		Transaction: tx,
	}

	if err := s.producer.Publish(ctx, enrtx); err != nil {
		slog.ErrorContext(ctx, "error while publishing transaction", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	writeJSON(w, 202, TransactionResponse{Id: &tx.ID})
}

func validateTransactionRequest(req TransactionRequest) error {
	if req.UserId == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}
	if req.Amount < 0.01 {
		return fmt.Errorf("amount must be at least 0.01")
	}
	if len(req.Currency) != 3 {
		return fmt.Errorf("currency must contain exactly 3 characters")
	}
	if len(req.Country) != 2 {
		return fmt.Errorf("country must contain exactly 2 characters")
	}
	if strings.TrimSpace(req.Ip) == "" {
		return fmt.Errorf("ip is required")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode HTTP response", slog.String("error", err.Error()))
	}
}
