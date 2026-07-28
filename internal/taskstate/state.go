// Package taskstate implements the MCP Tasks extension state machine
// as a pure, side-effect-free function. It has no knowledge of storage,
// networking, or persistence — see docs/architecture.md for why this
// separation exists.
package taskstate

import (
	"errors"
	"fmt"
)

// State is a task's current lifecycle state, per the MCP Tasks
// extension spec (io.modelcontextprotocol/tasks, SEP-2663).
type State string

const (
	StateWorking       State = "working"
	StateInputRequired State = "input_required"
	StateCompleted     State = "completed"
	StateFailed        State = "failed"
	StateCancelled     State = "cancelled"
)

// IsTerminal reports whether no further transitions are legal from
// this state. All three terminal states are final per spec.
func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCancelled:
		return true
	default:
		return false
	}
}

func (s State) valid() bool {
	switch s {
	case StateWorking, StateInputRequired, StateCompleted, StateFailed, StateCancelled:
		return true
	default:
		return false
	}
}

// Event is an input that may drive a state transition.
type Event string

const (
	// EventInputRequested: server needs client input (working -> input_required).
	EventInputRequested Event = "input_requested"
	// EventInputReceived: client fulfilled inputRequests via tasks/update
	// (input_required -> working).
	EventInputReceived Event = "input_received"
	// EventCompleted: request finished, including tool results with
	// isError:true (working|input_required -> completed).
	EventCompleted Event = "completed"
	// EventFailed: a JSON-RPC protocol error occurred during execution
	// (working|input_required -> failed). MUST NOT be used for
	// tool-level errors — those are EventCompleted.
	EventFailed Event = "failed"
	// EventCancelled: cancellation took effect (working|input_required
	// -> cancelled). Cooperative — may never fire even after tasks/cancel.
	EventCancelled Event = "cancelled"
)

// ErrInvalidTransition is returned (wrapped) for any transition not
// explicitly allowed by the state machine, including any transition
// attempted from a terminal state.
var ErrInvalidTransition = errors.New("invalid state transition")

// Transition computes the next state for a given current state and
// event. It is a pure function: no I/O, no locking, no persistence.
// Illegal transitions return ErrInvalidTransition wrapped with detail
// — they are never silently allowed and Transition never panics.
func Transition(current State, event Event) (State, error) {
	if !current.valid() {
		return current, fmt.Errorf("%w: unknown state %q", ErrInvalidTransition, current)
	}

	if current.IsTerminal() {
		return current, fmt.Errorf("%w: state %q is terminal, cannot apply event %q",
			ErrInvalidTransition, current, event)
	}

	switch current {
	case StateWorking:
		switch event {
		case EventInputRequested:
			return StateInputRequired, nil
		case EventCompleted:
			return StateCompleted, nil
		case EventFailed:
			return StateFailed, nil
		case EventCancelled:
			return StateCancelled, nil
		default:
			return current, fmt.Errorf("%w: event %q not legal from state %q",
				ErrInvalidTransition, event, current)
		}

	case StateInputRequired:
		switch event {
		case EventInputReceived:
			return StateWorking, nil
		case EventCompleted:
			return StateCompleted, nil
		case EventFailed:
			return StateFailed, nil
		case EventCancelled:
			return StateCancelled, nil
		default:
			return current, fmt.Errorf("%w: event %q not legal from state %q",
				ErrInvalidTransition, event, current)
		}

	default:
		// Unreachable given the validity + terminal checks above,
		// but kept explicit rather than assuming.
		return current, fmt.Errorf("%w: unhandled state %q", ErrInvalidTransition, current)
	}
}
