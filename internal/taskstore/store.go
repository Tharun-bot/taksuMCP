// Package taskstore defines the durable TaskStore interface and
// backend implementations (memory, sqlite, postgres). Legality of any
// transition is decided by internal/taskstate — this package is only
// responsible for durably recording state and enforcing idempotent
// creation. See docs/architecture.md.
package taskstore

import (
	"context"
	"errors"
	"time"

	"github.com/Tharun-bot/taksuMCP/internal/taskstate"
)

// ErrNotFound is returned when a taskId does not correspond to a known
// task. Callers (mcpwire) map this to JSON-RPC -32602 per spec.
var ErrNotFound = errors.New("task not found")

// ErrInvalidTransition is re-exported so callers don't need to import
// taskstate directly just to check the error.
var ErrInvalidTransition = taskstate.ErrInvalidTransition

// Task is the durable record for one task. Field names mirror the
// spec's Task shape (docs/spec-lock.md) so mapping to the wire format
// in internal/mcpwire stays mechanical.
type Task struct {
	TaskID         string
	Status         taskstate.State
	StatusMessage  string
	CreatedAt      time.Time
	LastUpdatedAt  time.Time
	TTLMs          *int64 // nil = unlimited, per spec
	PollIntervalMs *int64

	// Result / Error / InputRequests are populated depending on
	// Status. Left as opaque JSON so this package has no dependency
	// on MCP wire types.
	Result        []byte // set when Status == completed
	Error         []byte // set when Status == failed
	InputRequests []byte // set when Status == input_required
}

// CreateParams carries what's needed to create a task. IdempotencyKey
// is caller-supplied (e.g. derived from the originating request) and
// MUST produce the same TaskID on repeated calls with the same key —
// this is what makes retried tools/call requests safe.
type CreateParams struct {
	IdempotencyKey string
	TTLMs          *int64
	PollIntervalMs *int64
}

// TaskStore is implemented by each backend (memory, sqlite, postgres).
// All methods must be safe for concurrent use.
//
// NOTE: List is intentionally NOT exposed over the wire protocol —
// the MCP Tasks spec has no tasks/list, deliberately, to prevent
// cross-caller task enumeration (see docs/spec-lock.md). List exists
// here solely for the self-hosted HTMX dashboard, a different trust
// boundary (the operator, not an arbitrary MCP client). Do not wire
// this into internal/mcpwire.
type TaskStore interface {
	// Create durably creates a task and returns it. Calling Create
	// again with the same IdempotencyKey MUST return the existing
	// task rather than creating a second one.
	Create(ctx context.Context, params CreateParams) (Task, error)

	// Get returns the current state of a task. Returns ErrNotFound if
	// the taskId is unknown or has expired and been purged.
	Get(ctx context.Context, taskID string) (Task, error)

	// UpdateState applies event via taskstate.Transition and
	// durably persists the result. Returns ErrInvalidTransition
	// (wrapped) if the transition is illegal. resultPayload/
	// errorPayload/inputRequests are attached depending on the event;
	// pass nil for whichever don't apply.
	UpdateState(ctx context.Context, taskID string, event taskstate.Event, payload StatePayload) (Task, error)

	// List returns all tasks. Dashboard-only — see interface doc above.
	List(ctx context.Context) ([]Task, error)

	// Cancel signals cancellation intent. Per spec this is cooperative:
	// implementations may choose whether/when to actually transition
	// to cancelled; Cancel itself just must not error for a valid,
	// non-terminal taskID.
	Cancel(ctx context.Context, taskID string) error
}

// StatePayload carries the data attached to a state transition. Only
// the field matching the target state needs to be set.
type StatePayload struct {
	StatusMessage string
	Result        []byte
	Error         []byte
	InputRequests []byte
}
