package legal

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Advisory lock key for legal refresh (stable int64 across pods).
const refreshAdvisoryLockKey int64 = 0x4C4547414C52 // "LEGALR"

// Refresher periodically pulls TEDB + BCE and updates legal_values under an advisory lock.
type Refresher struct {
	Pool     *pgxpool.Pool
	Service  *Service
	Interval time.Duration
	Logger   *slog.Logger
}

func NewRefresher(pool *pgxpool.Pool, svc *Service, interval time.Duration) *Refresher {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Refresher{Pool: pool, Service: svc, Interval: interval, Logger: slog.Default()}
}

// Start runs RefreshOnce immediately then on Interval until ctx is cancelled.
func (r *Refresher) Start(ctx context.Context) {
	if r == nil || r.Service == nil {
		return
	}
	log := r.Logger
	if log == nil {
		log = slog.Default()
	}
	_ = r.RefreshOnce(ctx)
	t := time.NewTicker(r.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.RefreshOnce(ctx); err != nil {
				log.Warn("legal refresh", "error", err)
			}
		}
	}
}

// RefreshOnce acquires the advisory lock (non-blocking) and refreshes upstream keys.
func (r *Refresher) RefreshOnce(ctx context.Context) error {
	if r.Pool == nil {
		return r.refreshUnlocked(ctx)
	}
	conn, err := r.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	var got bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, refreshAdvisoryLockKey).Scan(&got); err != nil {
		return err
	}
	if !got {
		if r.Logger != nil {
			r.Logger.Info("legal refresh skipped (lock held by another replica)")
		}
		return nil
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, refreshAdvisoryLockKey)
	}()
	return r.refreshUnlocked(ctx)
}

func (r *Refresher) refreshUnlocked(ctx context.Context) error {
	svc := r.Service
	if svc == nil || svc.Store == nil {
		return ErrNotConfigured
	}
	log := r.Logger
	if log == nil {
		log = slog.Default()
	}

	if svc.TEDB != nil {
		r.fetchAndStore(ctx, svc.TEDB, KeyTEDBVATRatesFR, log)
	}
	if svc.BCE != nil {
		r.fetchAndStore(ctx, svc.BCE, KeyBCEKeyRate, log)
	}
	r.deriveAndStore(ctx, log)
	// Insee Sirene / Légifrance-PISTE : stubs listés dans l'API status uniquement (pas d'appel).
	_ = svc.Stubs
	return nil
}

func (r *Refresher) fetchAndStore(ctx context.Context, p Provider, key string, log *slog.Logger) {
	v, err := p.Fetch(ctx, key)
	if err != nil {
		log.Warn("legal fetch failed", "provider", p.Name(), "key", key, "error", err)
		prev, _, _ := r.Service.Store.Latest(ctx, key)
		val := prev.Value
		if len(val) == 0 {
			val = json.RawMessage(`{}`)
		}
		_ = r.Service.Store.Append(ctx, Variable{
			Key: key, Value: val, Source: p.Name(), FetchedAt: time.Now().UTC(),
		}, StatusError, err.Error())
		return
	}
	v.Status = StatusVerified
	if err := r.Service.Store.Upsert(ctx, v); err != nil {
		log.Warn("legal upsert failed", "key", key, "error", err)
	}
}

func (r *Refresher) deriveAndStore(ctx context.Context, log *slog.Logger) {
	var legalInterest, bceRate float64
	if v, _, err := r.Service.Store.Latest(ctx, KeyLegalInterestRate); err == nil {
		var m struct {
			AnnualPercent float64 `json:"annual_percent"`
		}
		_ = json.Unmarshal(v.Value, &m)
		legalInterest = m.AnnualPercent
	}
	if v, _, err := r.Service.Store.Latest(ctx, KeyBCEKeyRate); err == nil {
		var m KeyRate
		_ = json.Unmarshal(v.Value, &m)
		bceRate = m.AnnualPercent
	}
	rate, rule := DeriveLatePenaltyRate(legalInterest, bceRate)
	if rate <= 0 {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"annual_percent":   rate,
		"rule":             rule,
		"legal_interest":   legalInterest,
		"bce_key_rate":     bceRate,
		"bce_plus_10":      bceRate + 10,
		"three_times_legal": legalInterest * 3,
		"note":             "À valider juridiquement — max(3×intérêt légal, BCE+10)",
	})
	v := Variable{
		Key:       KeyLatePenaltyEffective,
		Value:     payload,
		Source:    "derived:" + rule,
		FetchedAt: time.Now().UTC(),
		Status:    StatusVerified,
	}
	if err := r.Service.Store.Upsert(ctx, v); err != nil {
		log.Warn("legal derive upsert failed", "error", err)
	}
}
