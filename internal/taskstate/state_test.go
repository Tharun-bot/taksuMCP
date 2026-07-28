package taskstate

import (
	"errors"
	"testing"
)

// allStates and allEvents let the table below assert every state x
// event combination explicitly — not just the happy path.
var allStates = []State{
	StateWorking, StateInputRequired, StateCompleted, StateFailed, StateCancelled,
}

var allEvents = []Event{
	EventInputRequested, EventInputReceived, EventCompleted, EventFailed, EventCancelled,
}

func TestTransition_AllPairs(t *testing.T) {
	type want struct {
		next    State
		wantErr bool
	}

	// Every legal transition, explicitly. Anything not listed here for
	// a given (state, event) pair is expected to error.
	legal := map[State]map[Event]State{
		StateWorking: {
			EventInputRequested: StateInputRequired,
			EventCompleted:      StateCompleted,
			EventFailed:         StateFailed,
			EventCancelled:      StateCancelled,
		},
		StateInputRequired: {
			EventInputReceived: StateWorking,
			EventCompleted:     StateCompleted,
			EventFailed:        StateFailed,
			EventCancelled:     StateCancelled,
		},
		// StateCompleted, StateFailed, StateCancelled: no legal
		// transitions at all — terminal.
	}

	for _, s := range allStates {
		for _, e := range allEvents {
			s, e := s, e // capture
			name := string(s) + "/" + string(e)
			t.Run(name, func(t *testing.T) {
				next, err := Transition(s, e)

				expectedNext, ok := legal[s][e]
				if !ok {
					// Expected to be illegal.
					if err == nil {
						t.Fatalf("Transition(%q, %q) = %q, nil; want error", s, e, next)
					}
					if !errors.Is(err, ErrInvalidTransition) {
						t.Fatalf("Transition(%q, %q) error = %v; want wrapping ErrInvalidTransition", s, e, err)
					}
					// State must be unchanged on error.
					if next != s {
						t.Fatalf("Transition(%q, %q) returned state %q on error; want unchanged %q", s, e, next, s)
					}
					return
				}

				// Expected to be legal.
				if err != nil {
					t.Fatalf("Transition(%q, %q) unexpected error: %v", s, e, err)
				}
				if next != expectedNext {
					t.Fatalf("Transition(%q, %q) = %q; want %q", s, e, next, expectedNext)
				}
			})
		}
	}
}

func TestTransition_UnknownState(t *testing.T) {
	_, err := Transition(State("bogus"), EventCompleted)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition for unknown state, got %v", err)
	}
}

func TestTransition_NeverPanics(t *testing.T) {
	// Fuzz-lite: every combination of state x event, plus garbage
	// values, must return an error rather than panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Transition panicked: %v", r)
		}
	}()

	garbageStates := append(allStates, State(""), State("WORKING"), State("👀"))
	garbageEvents := append(allEvents, Event(""), Event("COMPLETE"))

	for _, s := range garbageStates {
		for _, e := range garbageEvents {
			_, _ = Transition(s, e)
		}
	}
}

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateWorking, false},
		{StateInputRequired, false},
		{StateCompleted, true},
		{StateFailed, true},
		{StateCancelled, true},
	}
	for _, tt := range tests {
		if got := tt.state.IsTerminal(); got != tt.want {
			t.Errorf("State(%q).IsTerminal() = %v; want %v", tt.state, got, tt.want)
		}
	}
}
