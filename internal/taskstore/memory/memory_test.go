package memory

import (
	"context"
	"sync"
	"testing"

	"github.com/Tharun-bot/taksuMCP/internal/taskstate"
	"github.com/Tharun-bot/taksuMCP/internal/taskstore"
)

func TestCreate_IdempotentUnderConcurrency(t *testing.T) {
	s := New()
	ctx := context.Background()

	const goroutines = 50
	const sameKey = "idempotency-key-under-test"

	var wg sync.WaitGroup
	ids := make([]string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task, err := s.Create(ctx, taskstore.CreateParams{IdempotencyKey: sameKey})
			if err != nil {
				t.Errorf("Create() error: %v", err)
				return
			}
			ids[i] = task.TaskID
		}(i)
	}
	wg.Wait()

	first := ids[0]
	if first == "" {
		t.Fatal("expected non-empty task ID")
	}
	for i, id := range ids {
		if id != first {
			t.Fatalf("goroutine %d got TaskID %q, want %q (same idempotency key must produce one task)", i, id, first)
		}
	}

	all, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("List() returned %d tasks, want exactly 1", len(all))
	}
}

func TestCreate_DifferentKeysProduceDifferentTasks(t *testing.T) {
	s := New()
	ctx := context.Background()

	t1, _ := s.Create(ctx, taskstore.CreateParams{IdempotencyKey: "key-a"})
	t2, _ := s.Create(ctx, taskstore.CreateParams{IdempotencyKey: "key-b"})

	if t1.TaskID == t2.TaskID {
		t.Fatalf("different idempotency keys produced the same TaskID %q", t1.TaskID)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := New()
	_, err := s.Get(context.Background(), "does-not-exist")
	if err != taskstore.ErrNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestUpdateState_IllegalTransitionRejected(t *testing.T) {
	s := New()
	ctx := context.Background()

	task, _ := s.Create(ctx, taskstore.CreateParams{IdempotencyKey: "k"})

	// working -> completed is legal; do it once to reach a terminal state.
	task, err := s.UpdateState(ctx, task.TaskID, taskstate.EventCompleted, taskstore.StatePayload{
		Result: []byte(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("expected legal transition to succeed, got %v", err)
	}
	if task.Status != taskstate.StateCompleted {
		t.Fatalf("status = %q, want completed", task.Status)
	}

	// completed -> anything must now be rejected.
	_, err = s.UpdateState(ctx, task.TaskID, taskstate.EventInputRequested, taskstore.StatePayload{})
	if err == nil {
		t.Fatal("expected error transitioning out of terminal state, got nil")
	}
}

func TestUpdateState_UnknownTaskID(t *testing.T) {
	s := New()
	_, err := s.UpdateState(context.Background(), "ghost", taskstate.EventCompleted, taskstore.StatePayload{})
	if err != taskstore.ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestCancel_TerminalTaskDoesNotError(t *testing.T) {
	s := New()
	ctx := context.Background()

	task, _ := s.Create(ctx, taskstore.CreateParams{IdempotencyKey: "k"})
	_, _ = s.UpdateState(ctx, task.TaskID, taskstate.EventCompleted, taskstore.StatePayload{Result: []byte(`{}`)})

	// Per spec, cancel arriving after completion is not an error —
	// it's a cooperative, best-effort signal.
	if err := s.Cancel(ctx, task.TaskID); err != nil {
		t.Fatalf("Cancel() on terminal task returned error: %v", err)
	}
}

func TestCancel_UnknownTaskID(t *testing.T) {
	s := New()
	err := s.Cancel(context.Background(), "ghost")
	if err != taskstore.ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
