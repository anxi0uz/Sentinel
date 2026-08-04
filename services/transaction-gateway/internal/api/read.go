package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/pkg/storage"
	"github.com/google/uuid"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
)

func (s *Server) ListTransactions(w http.ResponseWriter, r *http.Request, params ListTransactionsParams) {
	ctx := r.Context()
	limit, offset, err := validateListTransactionsParams(params)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	count := sqlbuilder.NewSelectBuilder()
	count.SetFlavor(sqlbuilder.PostgreSQL)
	count.Select("count(*)")
	count.From("scored_transactions AS scored")
	count.JoinWithOption(sqlbuilder.LeftJoin, "alerts AS alert", "alert.scored_transaction_id = scored.id")
	applyTransactionFilters(count, params)
	countQuery, countArgs := count.Build()

	var total int
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "cannot count scored transactions", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	query := sqlbuilder.NewSelectBuilder()
	query.SetFlavor(sqlbuilder.PostgreSQL)
	query.Select(
		"scored.id",
		"scored.transaction_id",
		"scored.score",
		"scored.triggered_rules",
		"scored.processed_at",
		"scored.transaction_payload",
		"alert.id",
		"alert.severity",
		"outbox.published_at",
	)
	query.From("scored_transactions AS scored")
	query.JoinWithOption(sqlbuilder.LeftJoin, "alerts AS alert", "alert.scored_transaction_id = scored.id")
	query.JoinWithOption(sqlbuilder.LeftJoin, "outbox_events AS outbox", "outbox.topic = 'alerts' AND outbox.event_key = alert.id::text")
	applyTransactionFilters(query, params)
	query.OrderBy("scored.processed_at DESC", "scored.id DESC")
	query.Limit(limit)
	query.Offset(offset)
	listQuery, listArgs := query.Build()

	rows, err := s.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "cannot list scored transactions", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	defer rows.Close()

	items := make([]TransactionEvent, 0, limit)
	for rows.Next() {
		var scored models.ScoredTransaction
		var alertID *uuid.UUID
		var severity *string
		var publishedAt *time.Time
		if err := rows.Scan(
			&scored.ID,
			&scored.TransactionID,
			&scored.Score,
			&scored.TriggeredRules,
			&scored.ProcessedAt,
			&scored.TransactionPayload,
			&alertID,
			&severity,
			&publishedAt,
		); err != nil {
			slog.ErrorContext(ctx, "cannot scan scored transaction", slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
			return
		}

		var alert *models.Alert
		if alertID != nil && severity != nil {
			alert = &models.Alert{
				ID:                  *alertID,
				ScoredTransactionID: scored.ID,
				Severity:            models.Severity(*severity),
			}
		}
		item, err := transactionEvent(scored, alert, publishedAt != nil)
		if err != nil {
			slog.ErrorContext(ctx, "cannot build transaction response", slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "cannot iterate scored transactions", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, TransactionListResponse{
		Items: items,
		Pagination: Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}

func (s *Server) GetTransaction(w http.ResponseWriter, r *http.Request, transactionID uuid.UUID) {
	ctx := r.Context()
	scored, err := storage.GetOne[models.ScoredTransaction](ctx, s.db, "scored_transactions", func(sb *sqlbuilder.SelectBuilder) {
		sb.Where(sb.EQ("transaction_id", transactionID))
	})
	if errors.Is(err, storage.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "transaction not found"})
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "cannot get scored transaction", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	alert, err := storage.GetOne[models.Alert](ctx, s.db, "alerts", func(sb *sqlbuilder.SelectBuilder) {
		sb.Where(sb.EQ("scored_transaction_id", scored.ID))
	})
	if errors.Is(err, storage.ErrNotFound) {
		alert = nil
	} else if err != nil {
		slog.ErrorContext(ctx, "cannot get transaction alert", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	published := false
	if alert != nil {
		err := s.db.QueryRow(ctx, `
			SELECT published_at IS NOT NULL
			FROM outbox_events
			WHERE topic = 'alerts' AND event_key = $1`, alert.ID.String(),
		).Scan(&published)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			slog.ErrorContext(ctx, "cannot get alert delivery status", slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
			return
		}
	}

	response, err := transactionEvent(*scored, alert, published)
	if err != nil {
		slog.ErrorContext(ctx, "cannot build transaction response", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var response StatsResponse
	if err := s.db.QueryRow(ctx, `
		SELECT count(scored.id),
		       count(alert.id),
		       coalesce(avg(scored.score), 0),
		       count(*) FILTER (WHERE alert.severity = 'MEDIUM'),
		       count(*) FILTER (WHERE alert.severity = 'HIGH'),
		       count(*) FILTER (WHERE alert.severity = 'CRITICAL')
		FROM scored_transactions AS scored
		LEFT JOIN alerts AS alert ON alert.scored_transaction_id = scored.id`,
	).Scan(
		&response.Processed,
		&response.Alerts,
		&response.AverageScore,
		&response.BySeverity.Medium,
		&response.BySeverity.High,
		&response.BySeverity.Critical,
	); err != nil {
		slog.ErrorContext(ctx, "cannot get dashboard statistics", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	rows, err := s.db.Query(ctx, `
		SELECT canonical_rule_name, count(*) AS trigger_count
		FROM (
			SELECT CASE rule_name
				WHEN 'north_korea' THEN 'sanctioned_jurisdiction'
				ELSE rule_name
			END AS canonical_rule_name
			FROM scored_transactions,
			     unnest(coalesce(triggered_rules, ARRAY[]::text[])) AS rule_name
		) AS triggered
		GROUP BY canonical_rule_name
		ORDER BY trigger_count DESC, canonical_rule_name
		LIMIT 5`)
	if err != nil {
		slog.ErrorContext(ctx, "cannot get top triggered rules", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	defer rows.Close()

	response.TopRules = []RuleCount{}
	for rows.Next() {
		var rule RuleCount
		if err := rows.Scan(&rule.Name, &rule.Count); err != nil {
			slog.ErrorContext(ctx, "cannot scan top rule", slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
			return
		}
		response.TopRules = append(response.TopRules, rule)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "cannot iterate top rules", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func validateListTransactionsParams(params ListTransactionsParams) (int, int, error) {
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}
	if limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset must not be negative")
	}
	if params.MinScore != nil && *params.MinScore < 0 {
		return 0, 0, fmt.Errorf("min_score must not be negative")
	}
	if params.Severity != nil && !params.Severity.Valid() {
		return 0, 0, fmt.Errorf("invalid severity")
	}
	return limit, offset, nil
}

func applyTransactionFilters(builder *sqlbuilder.SelectBuilder, params ListTransactionsParams) {
	if params.Severity != nil {
		builder.Where(builder.EQ("alert.severity", string(*params.Severity)))
	}
	if params.MinScore != nil {
		builder.Where(builder.GreaterEqualThan("scored.score", *params.MinScore))
	}
}
