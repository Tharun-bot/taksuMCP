// Package clock abstracts time so tests can drive time-dependent logic
// (like the TTL reaper) deterministically, without real sleeps.
package clock

import (
	"sync"
	"time"
)

// Clock is anything that can report the current time.
type Clock interface {
	Now() time.Time
}

// Real is the production Clock, backed by time.Now.
type Real struct{}

func (Real) Now() time.Time { return time.Now() }

// Fake is a manually-advanced Clock for tests. Zero value is not
// usable — construct with NewFake.
type Fake struct {
	mu sync.Mutex
	t  time.Time
}

// NewFake returns a Fake clock starting at the given time.
func NewFake(start time.Time) *Fake {
	return &Fake{t: start}
}

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Advance moves the fake clock forward by d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}
