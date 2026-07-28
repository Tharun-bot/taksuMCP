// Package reaper implements TTL-based expiry for tasks. It depends
// only on the minimal Deleter interface (List + Delete) — not on any
// concrete backend — so swapping the in-memory store for SQLite or
// Postgres later requires no change here.
package reaper

import (
	"context"
	"log/slog"
	"time"

	"github.com/Tharun-bot/taksuMCP/internal/clock"
	"github.com/Tharun-bot/taksuMCP/internal/taskstore"
)

// Deleter is the subset of taskstore.TaskStore the reaper needs.
type Deleter interface {
	List(ctx context.Context) ([]taskstore.Task, error)
	Delete(ctx context.Context, taskID string) error
}

// Reaper periodically purges tasks whose TTL has elapsed.
type Reaper struct {
	store Deleter
	clock clock.Clock
	log   *slog.Logger
}

// New constructs a Reaper. Pass clock.Real{} in production;
// tests should pass a clock.Fake and call Sweep directly.
func New(store Deleter, c clock.Clock, log *slog.Logger) *Reaper {
	if log == nil {
		log = slog.Default()
	}
	return &Reaper{store: store, clock: c, log: log}
}

// Sweep runs one purge pass against the reaper's clock. Exposed
// directly so tests can call it deterministically instead of waiting
// on a real ticker. A task with TTLMs == nil never expires, per spec.
// The boundary is inclusive: a task expires the instant
// now >= createdAt + ttlMs, not strictly after.
func (r *Reaper) Sweep(ctx context.Context) (purged int, err error) {
	tasks, err := r.store.List(ctx)
	if err != nil {
		return 0, err
	}

	now := r.clock.Now()
	for _, t := range tasks {
		if t.TTLMs == nil {
			continue
		}
		expiresAt := t.CreatedAt.Add(time.Duration(*t.TTLMs) * time.Millisecond)
		if now.Before(expiresAt) {
			continue
		}
		if err := r.store.Delete(ctx, t.TaskID); err != nil {
			r.log.Error("reaper: failed to delete expired task", "task_id", t.TaskID, "error", err)
			continue
		}
		purged++
	}
	return purged, nil
}

// Start runs Sweep on a real ticker until ctx is cancelled.
// Production use only — tests call Sweep directly with a fake clock.
func (r *Reaper) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.Sweep(ctx); err != nil {
				r.log.Error("reaper: sweep failed", "error", err)
			}
		}
	}
}
