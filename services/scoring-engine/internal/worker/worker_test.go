package worker

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type sliceReader struct {
	mu       sync.Mutex
	messages []kafka.Message
	commits  chan kafka.Message
}

func (r *sliceReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	if len(r.messages) > 0 {
		message := r.messages[0]
		r.messages = r.messages[1:]
		r.mu.Unlock()
		return message, nil
	}
	r.mu.Unlock()
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (r *sliceReader) CommitMessages(_ context.Context, messages ...kafka.Message) error {
	for _, message := range messages {
		r.commits <- message
	}
	return nil
}

type recordingWriter struct {
	messages chan kafka.Message
	err      error
}

func (w *recordingWriter) WriteMessages(_ context.Context, messages ...kafka.Message) error {
	if w.err != nil {
		w.messages <- messages[0]
		return w.err
	}
	for _, message := range messages {
		w.messages <- message
	}
	return nil
}

type staticRules []models.FraudRule

func (r staticRules) Get() []models.FraudRule { return r }

func TestPoolPreservesOrderWithinPartition(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	input := make([]kafka.Message, 0, len(ids))
	for offset, id := range ids {
		payload, err := json.Marshal(models.EnrichedTransaction{
			Transaction: models.Transaction{ID: id},
		})
		if err != nil {
			t.Fatal(err)
		}
		input = append(input, kafka.Message{Partition: 3, Offset: int64(offset), Value: payload})
	}
	reader := &sliceReader{messages: input, commits: make(chan kafka.Message, len(input))}
	writer := &recordingWriter{messages: make(chan kafka.Message, len(input))}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		New(reader, writer, staticRules{}, 4).Run(ctx)
		close(done)
	}()

	for i, id := range ids {
		select {
		case message := <-writer.messages:
			if message.Topic != "scored" {
				t.Fatalf("message %d topic = %q, want scored", i, message.Topic)
			}
			if string(message.Key) != id.String() {
				t.Fatalf("message %d key = %q, want %q", i, message.Key, id)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for produced message")
		}
		select {
		case committed := <-reader.commits:
			if committed.Offset != int64(i) {
				t.Fatalf("committed offset %d, want %d", committed.Offset, i)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for commit")
		}
	}
	cancel()
	<-done
}

func TestPoolDoesNotCommitFailedPublish(t *testing.T) {
	id := uuid.New()
	payload, err := json.Marshal(models.EnrichedTransaction{Transaction: models.Transaction{ID: id}})
	if err != nil {
		t.Fatal(err)
	}
	reader := &sliceReader{
		messages: []kafka.Message{{Partition: 0, Offset: 1, Value: payload}},
		commits:  make(chan kafka.Message, 1),
	}
	writer := &recordingWriter{messages: make(chan kafka.Message, 1), err: errors.New("kafka unavailable")}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		New(reader, writer, staticRules{}, 1).Run(ctx)
		close(done)
	}()

	select {
	case <-writer.messages:
	case <-time.After(time.Second):
		t.Fatal("writer was not called")
	}
	cancel()
	<-done
	select {
	case <-reader.commits:
		t.Fatal("message was committed after failed publish")
	default:
	}
}
