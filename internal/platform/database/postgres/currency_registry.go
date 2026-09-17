package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

type CurrencyRegistry struct {
	mu         sync.RWMutex
	currencies map[currency.Code]currency.Currency
	pool       *pgxpool.Pool
	queries    *sqlc.Queries
}

func NewCurrencyRegistry(ctx context.Context, pool *pgxpool.Pool, queries *sqlc.Queries) (*CurrencyRegistry, error) {
	r := &CurrencyRegistry{
		currencies: make(map[currency.Code]currency.Currency),
		pool:       pool,
		queries:    queries,
	}

	if err := r.Reload(ctx); err != nil {
		return nil, fmt.Errorf("failed initial currency cache load: %w", err)
	}

	go r.startListener(ctx)

	return r, nil
}

func (r *CurrencyRegistry) Get(code currency.Code) (currency.Currency, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	curr, ok := r.currencies[code]
	return curr, ok && curr.IsActive
}

func (r *CurrencyRegistry) Reload(ctx context.Context) error {
	dbCurrencies, err := r.queries.ListActiveCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("listing active currencies: %w", err)
	}

	freshMap := make(map[currency.Code]currency.Currency, len(dbCurrencies))
	for _, c := range dbCurrencies {
		code := currency.Code(c.Code)
		freshMap[code] = currency.Currency{
			Code:     code,
			Name:     c.Name,
			Exponent: int(c.Exponent),
			IsActive: c.IsActive,
		}
	}

	r.mu.Lock()
	r.currencies = freshMap
	r.mu.Unlock()

	slog.Info("[CurrencyRegistry] In-memory cache refreshed", "count", len(freshMap))
	return nil
}

// startListener manages a dedicated connection, listening for invalidation events with automatic reconnects
func (r *CurrencyRegistry) startListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := r.listenLoop(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("[CurrencyRegistry] Connection lost. Reconnecting in 3s...", "err", err)
				time.Sleep(3 * time.Second)

				// Re-sync cache upon reconnect to catch any updates missed during downtime
				if syncErr := r.Reload(ctx); syncErr != nil {
					slog.Error("[CurrencyRegistry] Cache re-sync failed after reconnect", "err", syncErr)
				}
			}
		}
	}
}

func (r *CurrencyRegistry) listenLoop(ctx context.Context) error {
	// Acquire a dedicated connection from the pool (LISTEN requires a persistent connection session)
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire listener connection: %w", err)
	}
	defer conn.Release()

	// Execute LISTEN command on channel 'currency_updated'
	_, err = conn.Exec(ctx, "LISTEN currency_updated")
	if err != nil {
		return fmt.Errorf("executing LISTEN failed: %w", err)
	}

	for {
		// Blocks until a notification arrives or context is cancelled
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err // Triggers outer reconnect loop
		}

		slog.Info("[CurrencyRegistry] Received notification for currency", "payload", notification.Payload)

		// Refresh in-memory cache
		if err := r.Reload(ctx); err != nil {
			slog.Error("[CurrencyRegistry] Failed to refresh cache", "err", err)
		}
	}
}
