# Architecture

## Why the state machine is a separate package from storage

`internal/taskstate` contains exactly one thing: a pure function,
`Transition(State, Event) (State, error)`. It has no storage backend,
no locking, no network code, and no dependency on `internal/taskstore`.

This separation is deliberate:

1. **Testability.** The entire state machine — every legal and illegal
   transition — is covered by a single in-memory table-driven test
   with zero setup (no DB, no mocks, no goroutines). That test suite
   stays fast and deterministic forever, regardless of what storage
   backend gets added later.

2. **Backend independence.** `internal/taskstore` (memory, sqlite,
   postgres backends) all call the *same* `Transition` function to
   decide legality before persisting a change. The rules for "is this
   transition allowed" live in exactly one place — a backend can't
   drift from another by reimplementing the logic slightly differently.

3. **Spec fidelity.** The MCP Tasks extension defines the state machine
   independently of any implementation concern (see `docs/spec-lock.md`).
   Keeping our code structured the same way makes it easy to verify our
   implementation still matches the spec after future spec revisions —
   diff one small file, not a storage layer.

Rule of thumb going forward: if a change is about *whether* a
transition is legal, it belongs in `taskstate`. If it's about *durably
recording* a transition (locking, persistence, concurrency), it
belongs in `taskstore`.