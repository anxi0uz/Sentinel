package cache

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/anxi0uz/sentinel/pkg/models"
	"github.com/anxi0uz/sentinel/pkg/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleCache struct {
	mu    sync.RWMutex
	rules []models.FraudRule
}

func (c *RuleCache) Get() []models.FraudRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]models.FraudRule(nil), c.rules...)
}

func (c *RuleCache) reload(ctx context.Context, db *pgxpool.Pool) error {
	rules, err := storage.GetAll[models.FraudRule](ctx, "fraud_rules", db)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.rules = rules
	c.mu.Unlock()
	return nil
}

func (c *RuleCache) Start(ctx context.Context, db *pgxpool.Pool, interval time.Duration) error {
	if err := c.reload(ctx, db); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.reload(ctx, db); err != nil {
					slog.ErrorContext(ctx, "failed to reload fraud rules", slog.String("error", err.Error()))
				}
			}
		}
	}()
	return nil
}
