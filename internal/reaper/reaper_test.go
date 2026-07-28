package reaper

import (
	"context"
	"testing"
	"time"

	"github.com/Tharun-bot/taksuMCP/internal/clock"
	"github.com/Tharun-bot/taksuMCP/internal/taskstore"
	"github.com/Tharun-bot/taksuMCP/internal/taskstore/memory"
)

func ttl(ms int64) *int64 { return &ms }

func TestSweep_FiresExactlyAtTTLBoundary(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := clock.NewFake(start)

	store := memory.NewWithClock(fc)
	r := New(store, fc, nil)

	task, err := store.Create(ctx, taskstore.CreateParams{
		IdempotencyKey: "k",
		TTLMs:          ttl(1000), // expires at start + 1000ms
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// 1ms before the boundary: must NOT be purged.
	fc.Advance(999 * time.Millisecond)
	purged, err := r.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep() error: %v", err)
	}
	if purged != 0 {
		t.Fatalf("Sweep() purged %d tasks before TTL boundary, want 0", purged)
	}
	if _, err := store.Get(ctx, task.TaskID); err != nil {
		t.Fatalf("task should still exist before TTL boundary, Get() error: %v", err)
	}

	// Exactly at the boundary: MUST be purged.
	fc.Advance(1 * time.Millisecond) // now == start + 1000ms
	purged, err = r.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep() error: %v", err)
	}
	if purged != 1 {
		t.Fatalf("Sweep() purged %d tasks exactly at TTL boundary, want 1", purged)
	}
	if _, err := store.Get(ctx, task.TaskID); err != taskstore.ErrNotFound {
		t.Fatalf("task should be purged at TTL boundary, Get() error = %v, want ErrNotFound", err)
	}
}

func TestSweep_NilTTLNeverExpires(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := clock.NewFake(start)

	store := memory.NewWithClock(fc)
	r := New(store, fc, nil)

	task, _ := store.Create(ctx, taskstore.CreateParams{
		IdempotencyKey: "k",
		TTLMs:          nil, // unlimited
	})

	fc.Advance(365 * 24 * time.Hour) // one year later
	purged, err := r.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep() error: %v", err)
	}
	if purged != 0 {
		t.Fatalf("Sweep() purged %d tasks with nil TTL, want 0", purged)
	}
	if _, err := store.Get(ctx, task.TaskID); err != nil {
		t.Fatalf("task with nil TTL should never expire, Get() error: %v", err)
	}
}

func TestSweep_AfterBoundaryAlsoPurges(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := clock.NewFake(start)

	store := memory.NewWithClock(fc)
	r := New(store, fc, nil)

	task, _ := store.Create(ctx, taskstore.CreateParams{IdempotencyKey: "k", TTLMs: ttl(1000)})

	fc.Advance(5 * time.Second) // well past expiry
	purged, err := r.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep() error: %v", err)
	}
	if purged != 1 {
		t.Fatalf("Sweep() purged %d tasks well past TTL, want 1", purged)
	}
	_ = task
}

func TestSweep_MultipleTasksMixedExpiry(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fc := clock.NewFake(start)

	store := memory.NewWithClock(fc)
	r := New(store, fc, nil)

	_, _ = store.Create(ctx, taskstore.CreateParams{IdempotencyKey: "expires-soon", TTLMs: ttl(1000)})
	_, _ = store.Create(ctx, taskstore.CreateParams{IdempotencyKey: "expires-later", TTLMs: ttl(5000)})
	_, _ = store.Create(ctx, taskstore.CreateParams{IdempotencyKey: "never-expires", TTLMs: nil})

	fc.Advance(2 * time.Second) // only the 1000ms task should be gone

	purged, err := r.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep() error: %v", err)
	}
	if purged != 1 {
		t.Fatalf("Sweep() purged %d tasks, want exactly 1", purged)
	}

	remaining, _ := store.List(ctx)
	if len(remaining) != 2 {
		t.Fatalf("expected 2 tasks remaining, got %d", len(remaining))
	}
}
