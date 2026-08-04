package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type fakeReader struct {
	messages chan kafka.Message
	commits  chan kafka.Message
}

func (r *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case message := <-r.messages:
		return message, nil
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	}
}

func (r *fakeReader) CommitMessages(_ context.Context, messages ...kafka.Message) error {
	for _, message := range messages {
		r.commits <- message
	}
	return nil
}

type repositoryFunc func(context.Context, models.ScoredTransactionEvent) (bool, error)

func (f repositoryFunc) Save(ctx context.Context, event models.ScoredTransactionEvent) (bool, error) {
	return f(ctx, event)
}

func TestAlertServiceCommitsAfterPersistence(t *testing.T) {
	event := models.ScoredTransactionEvent{
		Transaction: models.Transaction{ID: uuid.New()},
		Score:       95,
		ProcessedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	reader := &fakeReader{messages: make(chan kafka.Message, 1), commits: make(chan kafka.Message, 1)}
	reader.messages <- kafka.Message{Offset: 7, Value: payload}
	saved := make(chan models.ScoredTransactionEvent, 1)
	repository := repositoryFunc(func(_ context.Context, event models.ScoredTransactionEvent) (bool, error) {
		saved <- event
		return true, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		NewAlertService(reader, repository).Run(ctx)
		close(done)
	}()

	select {
	case got := <-saved:
		if got.Transaction.ID != event.Transaction.ID {
			t.Fatalf("saved transaction %s, want %s", got.Transaction.ID, event.Transaction.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("repository was not called")
	}
	select {
	case committed := <-reader.commits:
		if committed.Offset != 7 {
			t.Fatalf("committed offset %d, want 7", committed.Offset)
		}
	case <-time.After(time.Second):
		t.Fatal("message was not committed")
	}
	cancel()
	<-done
}

func TestAlertServiceDoesNotCommitPersistenceFailure(t *testing.T) {
	event := models.ScoredTransactionEvent{
		Transaction: models.Transaction{ID: uuid.New()},
		ProcessedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	reader := &fakeReader{messages: make(chan kafka.Message, 1), commits: make(chan kafka.Message, 1)}
	reader.messages <- kafka.Message{Value: payload}
	called := make(chan struct{}, 1)
	repository := repositoryFunc(func(_ context.Context, _ models.ScoredTransactionEvent) (bool, error) {
		called <- struct{}{}
		return false, errors.New("database unavailable")
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		NewAlertService(reader, repository).Run(ctx)
		close(done)
	}()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("repository was not called")
	}
	cancel()
	<-done
	select {
	case <-reader.commits:
		t.Fatal("failed message must not be committed")
	default:
	}
}

func TestAlertServiceCommitsSemanticallyInvalidEventWithoutPersistingIt(t *testing.T) {
	payload, err := json.Marshal(models.ScoredTransactionEvent{})
	if err != nil {
		t.Fatal(err)
	}
	reader := &fakeReader{messages: make(chan kafka.Message, 1), commits: make(chan kafka.Message, 1)}
	reader.messages <- kafka.Message{Offset: 9, Value: payload}
	repositoryCalled := make(chan struct{}, 1)
	repository := repositoryFunc(func(_ context.Context, _ models.ScoredTransactionEvent) (bool, error) {
		repositoryCalled <- struct{}{}
		return true, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		NewAlertService(reader, repository).Run(ctx)
		close(done)
	}()

	select {
	case committed := <-reader.commits:
		if committed.Offset != 9 {
			t.Fatalf("committed offset %d, want 9", committed.Offset)
		}
	case <-time.After(time.Second):
		t.Fatal("invalid event was not committed")
	}
	select {
	case <-repositoryCalled:
		t.Fatal("invalid event was persisted")
	default:
	}
	cancel()
	<-done
}

func TestSeverityForScore(t *testing.T) {
	tests := []struct {
		score    int
		severity models.Severity
		alert    bool
	}{
		{79, "", false},
		{80, models.SeverityMedium, true},
		{89, models.SeverityMedium, true},
		{90, models.SeverityHigh, true},
		{119, models.SeverityHigh, true},
		{120, models.SeverityCritical, true},
	}
	for _, tt := range tests {
		severity, alert := severityForScore(tt.score)
		if severity != tt.severity || alert != tt.alert {
			t.Errorf("score %d: got (%q, %v), want (%q, %v)", tt.score, severity, alert, tt.severity, tt.alert)
		}
	}
}

func TestMarshalTransactionSnapshotPreservesScoringInput(t *testing.T) {
	transactionID := uuid.New()
	userID := uuid.New()
	event := models.ScoredTransactionEvent{
		Transaction: models.Transaction{
			ID:        transactionID,
			UserID:    userID,
			Amount:    1599.50,
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
		},
	}

	payload, err := marshalTransactionSnapshot(event)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot models.EnrichedTransaction
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Transaction != event.Transaction {
		t.Fatalf("transaction snapshot = %+v, want %+v", snapshot.Transaction, event.Transaction)
	}
	if snapshot.User != event.User {
		t.Fatalf("user snapshot = %+v, want %+v", snapshot.User, event.User)
	}
}
