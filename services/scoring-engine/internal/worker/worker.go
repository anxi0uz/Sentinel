package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/services/scoring-engine/internal/cache"
	"github.com/anxi0uz/sentinel/services/scoring-engine/internal/scorer"
	"github.com/segmentio/kafka-go"
)

type Pool struct {
	reader    *kafka.Reader
	writer    *kafka.Writer
	cache     *cache.RuleCache
	workerNum int
	ch        chan kafka.Message
}

func New(reader *kafka.Reader, writer *kafka.Writer, cache *cache.RuleCache, workerNum int) *Pool {
	return &Pool{
		reader:    reader,
		writer:    writer,
		cache:     cache,
		workerNum: workerNum,
		ch:        make(chan kafka.Message, workerNum*2),
	}
}

func (p *Pool) Run(ctx context.Context) {
	go p.readLoop(ctx)
	for range p.workerNum {
		go p.work(ctx)
	}
	<-ctx.Done()
}

func (p *Pool) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-p.ch:
			var tx models.EnrichedTransaction
			if err := json.Unmarshal(msg.Value, &tx); err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal message", slog.String("error", err.Error()))
				if err := p.reader.CommitMessages(ctx, msg); err != nil {
					slog.ErrorContext(ctx, "error to commit message", slog.String("error", err.Error()))
				}
				continue
			}

			rules := p.cache.Get()
			score, triggered := scorer.Score(rules, tx)
			sc := models.ScoredTransactionEvent{
				Transaction:    tx.Transaction,
				Score:          score,
				TriggeredRules: triggered,
				ProcessedAt:    time.Now(),
			}
			payload, err := json.Marshal(sc)
			if err != nil {
				slog.ErrorContext(ctx, "failed to marshal scored transaction", slog.String("error", err.Error()))
				if err := p.reader.CommitMessages(ctx, msg); err != nil {
					slog.ErrorContext(ctx, "error to commit message", slog.String("error", err.Error()))
				}
				continue
			}
			if err := p.writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(tx.Transaction.ID.String()),
				Value: payload,
			}); err != nil {
				slog.ErrorContext(ctx, "failed to write scored transaction to kafka", slog.String("error", err.Error()))
				continue
			}
			if err := p.reader.CommitMessages(ctx, msg); err != nil {
				slog.ErrorContext(ctx, "error to commit message", slog.String("error", err.Error()))
				continue
			}
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
			continue
		}
		select {
		case p.ch <- msg:
		case <-ctx.Done():
			return
		}
	}
}
