// Package memory implements taskstore.TaskStore with a mutex-guarded
// in-memory map. Reference/test backend — not durable across restarts.
package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

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
	now         func() time.Time // overridable for tests
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		byID:        make(map[string]taskstore.Task),
		idempotency: make(map[string]string),
		now:         time.Now,
	}
}

// deriveTaskID computes a deterministic ID from the idempotency key.
// Using a hash (not uuid.New()) is what makes Create idempotent: the
// same key always maps to the same ID without needing a lookup first.
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
	now := s.now()
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
		return taskstore.Task{}, err // wraps taskstore.ErrInvalidTransition
	}

	task.Status = next
	task.LastUpdatedAt = s.now()
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
		// Cooperative + already-terminal: acking is still correct
		// per spec (cancel may arrive after completion). Not an error.
		return nil
	}

	next, err := taskstate.Transition(task.Status, taskstate.EventCancelled)
	if err != nil {
		// Server is allowed to decline; ack without changing state.
		return nil
	}
	task.Status = next
	task.LastUpdatedAt = s.now()
	s.byID[taskID] = task
	return nil
}

var _ taskstore.TaskStore = (*Store)(nil)
