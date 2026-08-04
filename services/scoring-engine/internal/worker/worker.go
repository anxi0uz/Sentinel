package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/services/scoring-engine/internal/scorer"
	"github.com/segmentio/kafka-go"
)

type MessageReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
}

type MessageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

type RuleProvider interface {
	Get() []models.FraudRule
}

type Pool struct {
	reader    MessageReader
	writer    MessageWriter
	cache     RuleProvider
	workerNum int
	workers   []chan kafka.Message
}

func New(reader MessageReader, writer MessageWriter, cache RuleProvider, workerNum int) *Pool {
	if workerNum < 1 {
		workerNum = 1
	}
	workers := make([]chan kafka.Message, workerNum)
	for i := range workers {
		workers[i] = make(chan kafka.Message, 2)
	}
	return &Pool{
		reader:    reader,
		writer:    writer,
		cache:     cache,
		workerNum: workerNum,
		workers:   workers,
	}
}

func (p *Pool) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, messages := range p.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.work(ctx, messages)
		}()
	}
	p.readLoop(ctx)
	for _, messages := range p.workers {
		close(messages)
	}
	wg.Wait()
}

func (p *Pool) work(ctx context.Context, messages <-chan kafka.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			var tx models.EnrichedTransaction
			if err := json.Unmarshal(msg.Value, &tx); err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal message", slog.String("error", err.Error()))
				p.retry(ctx, "commit invalid transaction", func() error {
					return p.reader.CommitMessages(ctx, msg)
				})
				continue
			}

			rules := p.cache.Get()
			score, triggered := scorer.Score(rules, tx)
			sc := models.ScoredTransactionEvent{
				Transaction:    tx.Transaction,
				User:           tx.User,
				Score:          score,
				TriggeredRules: triggered,
				ProcessedAt:    time.Now(),
			}
			payload, err := json.Marshal(sc)
			if err != nil {
				slog.ErrorContext(ctx, "failed to marshal scored transaction", slog.String("error", err.Error()))
				p.retry(ctx, "commit unprocessable transaction", func() error {
					return p.reader.CommitMessages(ctx, msg)
				})
				continue
			}
			if !p.retry(ctx, "publish scored transaction", func() error {
				return p.writer.WriteMessages(ctx, kafka.Message{
					Topic: "scored",
					Key:   []byte(tx.Transaction.ID.String()),
					Value: payload,
				})
			}) {
				return
			}
			if !p.retry(ctx, "commit scored transaction", func() error {
				return p.reader.CommitMessages(ctx, msg)
			}) {
				return
			}
		}
	}
}

func (p *Pool) retry(ctx context.Context, operation string, fn func() error) bool {
	for {
		if err := fn(); err == nil {
			return true
		} else if ctx.Err() == nil {
			slog.ErrorContext(ctx, operation+" failed; retrying", slog.String("error", err.Error()))
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
}

func (p *Pool) readLoop(ctx context.Context) {
	for {
		msg, err := p.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.ErrorContext(ctx, "failed to fetch message", slog.String("error", err.Error()))
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			continue
		}
		workerIndex := msg.Partition % len(p.workers)
		if workerIndex < 0 {
			workerIndex = -workerIndex
		}
		select {
		case p.workers[workerIndex] <- msg:
		case <-ctx.Done():
			return
		}
	}
}
