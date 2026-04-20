package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "Error while decoding body", slog.String("error", err.Error()))
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
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
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
