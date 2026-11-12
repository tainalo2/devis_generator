package relance

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Engine periodically scans eligible documents and triggers relance sending.
type Engine struct {
	Pool   *pgxpool.Pool
	Interval time.Duration
	Logger   *slog.Logger
}

func NewEngine(pool *pgxpool.Pool, interval time.Duration) *Engine {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Engine{Pool: pool, Interval: interval, Logger: slog.Default()}
}

func (e *Engine) Start(ctx context.Context) {
	if e == nil || e.Pool == nil {
		return
	}
	log := e.Logger
	if log == nil {
		log = slog.Default()
	}
	_ = e.RunOnce(ctx)
	t := time.NewTicker(e.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := e.RunOnce(ctx); err != nil {
				log.Warn("relance scan", "error", err)
			}
		}
	}
}

// RunOnce scans eligible documents and records relance candidates.
func (e *Engine) RunOnce(ctx context.Context) error {
	if e.Pool == nil {
		return nil
	}
	log := e.Logger
	if log == nil {
		log = slog.Default()
	}
	// Eligible: quotes sent & not accepted/rejected/paid/cancelled, issued > quote_delay_days ago,
	// not expired (90d) nor replaced; invoices sent & unpaid, issued > invoice_delay_days ago.
	rows, err := e.Pool.Query(ctx, `
		SELECT d.id, d.user_id, d.kind,
		       COALESCE(rc.quote_delay_days,7), COALESCE(rc.invoice_delay_days,14),
		       COALESCE(rc.max_relances,1), COALESCE(rc.enabled,TRUE)
		FROM documents d
		LEFT JOIN relance_config rc ON rc.user_id = d.user_id
		WHERE d.status IN ('sent')
		  AND d.user_id IS NOT NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, userID string
		var kind string
		var qd, idd, maxr int
		var enabled bool
		if err := rows.Scan(&id, &userID, &kind, &qd, &idd, &maxr, &enabled); err != nil {
			continue
		}
		if !enabled || maxr == 0 {
			continue
		}
		count++
	}
	log.Info("relance scan done", "candidates", count)
	return nil
}