package monitor

import (
	"fmt"
	"time"
)

type suppressEntry struct {
	lastMsg    string
	lastLogged time.Time
	count      int
}

// ErrorSuppressor suppresses repeated identical errors, logging them
// only once per interval with a count of suppressed occurrences.
type ErrorSuppressor struct {
	interval time.Duration
	nowFunc  func() time.Time
	entries  map[string]*suppressEntry
}

// SuppressOption configures ErrorSuppressor.
type SuppressOption func(*ErrorSuppressor)

// WithSuppressInterval sets the minimum interval between repeated error logs.
func WithSuppressInterval(d time.Duration) SuppressOption {
	return func(s *ErrorSuppressor) {
		s.interval = d
	}
}

// WithSuppressNowFunc sets a custom time source (for testing).
func WithSuppressNowFunc(f func() time.Time) SuppressOption {
	return func(s *ErrorSuppressor) {
		s.nowFunc = f
	}
}

// NewErrorSuppressor creates a new ErrorSuppressor.
func NewErrorSuppressor(opts ...SuppressOption) *ErrorSuppressor {
	s := &ErrorSuppressor{
		interval: time.Hour,
		nowFunc:  time.Now,
		entries:  make(map[string]*suppressEntry),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Check determines whether an error for the given key should be logged.
// Returns the message to log and whether it should be logged.
// A nil error resets the entry so the next error is logged immediately.
func (s *ErrorSuppressor) Check(key string, err error) (string, bool) {
	if err == nil {
		delete(s.entries, key)
		return "", false
	}

	msg := err.Error()
	now := s.nowFunc()
	entry, exists := s.entries[key]

	if !exists || entry.lastMsg != msg {
		s.entries[key] = &suppressEntry{
			lastMsg:    msg,
			lastLogged: now,
			count:      0,
		}
		return msg, true
	}

	if now.Sub(entry.lastLogged) >= s.interval {
		var result string
		if entry.count > 0 {
			result = fmt.Sprintf("%s (suppressed %d times)", msg, entry.count)
		} else {
			result = msg
		}
		entry.lastLogged = now
		entry.count = 0
		return result, true
	}

	entry.count++
	return "", false
}
