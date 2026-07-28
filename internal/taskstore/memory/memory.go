// Package memory implements taskstore.TaskStore with a mutex-guarded
// in-memory map. Reference/test backend — not durable across restarts.
package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/Tharun-bot/taksuMCP/internal/clock"
	"github.com/Tharun-bot/taksuMCP/internal/taskstate"
	"github.com/Tharun-bot/taksuMCP/internal/taskstore"
)

// Store is a mutex-guarded in-memory TaskStore.
type Store struct {
	mu   sync.Mutex
	byID map[string]taskstore.Task
	// idempotency maps a caller-supplied idempotency key to the
	// TaskID it produced, so repeated Create calls with the same key
	// return the same task instead of creating a duplicate.
	idempotency map[string]string
	clock       clock.Clock
}

// New returns an empty in-memory store using the real system clock.
func New() *Store {
	return NewWithClock(clock.Real{})
}

// NewWithClock returns an empty in-memory store using the given
// clock. Used in tests (e.g. by the reaper's fake-clock tests) to
// control time deterministically.
func NewWithClock(c clock.Clock) *Store {
	return &Store{
		byID:        make(map[string]taskstore.Task),
		idempotency: make(map[string]string),
		clock:       c,
	}
}

func deriveTaskID(idempotencyKey string) string {
	sum := sha256.Sum256([]byte(idempotencyKey))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Create(_ context.Context, params taskstore.CreateParams) (taskstore.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingID, ok := s.idempotency[params.IdempotencyKey]; ok {
		return s.byID[existingID], nil
	}

	taskID := deriveTaskID(params.IdempotencyKey)
	now := s.clock.Now()
	task := taskstore.Task{
		TaskID:         taskID,
		Status:         taskstate.StateWorking,
		CreatedAt:      now,
		LastUpdatedAt:  now,
		TTLMs:          params.TTLMs,
		PollIntervalMs: params.PollIntervalMs,
	}

	s.byID[taskID] = task
	s.idempotency[params.IdempotencyKey] = taskID
	return task, nil
}

func (s *Store) Get(_ context.Context, taskID string) (taskstore.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.byID[taskID]
	if !ok {
		return taskstore.Task{}, taskstore.ErrNotFound
	}
	return task, nil
}

func (s *Store) UpdateState(_ context.Context, taskID string, event taskstate.Event, payload taskstore.StatePayload) (taskstore.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.byID[taskID]
	if !ok {
		return taskstore.Task{}, taskstore.ErrNotFound
	}

	next, err := taskstate.Transition(task.Status, event)
	if err != nil {
		return taskstore.Task{}, err
	}

	task.Status = next
	task.LastUpdatedAt = s.clock.Now()
	if payload.StatusMessage != "" {
		task.StatusMessage = payload.StatusMessage
	}
	if payload.Result != nil {
		task.Result = payload.Result
	}
	if payload.Error != nil {
		task.Error = payload.Error
	}
	if payload.InputRequests != nil {
		task.InputRequests = payload.InputRequests
	}

	s.byID[taskID] = task
	return task, nil
}

func (s *Store) List(_ context.Context) ([]taskstore.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]taskstore.Task, 0, len(s.byID))
	for _, t := range s.byID {
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) Cancel(_ context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.byID[taskID]
	if !ok {
		return taskstore.ErrNotFound
	}
	if task.Status.IsTerminal() {
		return nil
	}

	next, err := taskstate.Transition(task.Status, taskstate.EventCancelled)
	if err != nil {
		return nil
	}
	task.Status = next
	task.LastUpdatedAt = s.clock.Now()
	s.byID[taskID] = task
	return nil
}

func (s *Store) Delete(_ context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[taskID]; !ok {
		return taskstore.ErrNotFound
	}
	delete(s.byID, taskID)

	// Clean up the idempotency mapping too, so a repeated Create with
	// the same key after deletion produces a fresh task rather than
	// silently resurrecting a stale one.
	for k, v := range s.idempotency {
		if v == taskID {
			delete(s.idempotency, k)
			break
		}
	}
	return nil
}

var _ taskstore.TaskStore = (*Store)(nil)
