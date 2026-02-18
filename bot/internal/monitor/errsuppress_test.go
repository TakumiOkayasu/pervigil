package monitor

import (
	"errors"
	"testing"
	"time"
)

func TestErrorSuppressor_FirstError(t *testing.T) {
	s := NewErrorSuppressor()

	msg, ok := s.Check("nic", errors.New("no hwmon for eth2"))
	if !ok {
		t.Fatal("expected first error to be logged")
	}
	if msg != "no hwmon for eth2" {
		t.Fatalf("unexpected msg: %s", msg)
	}
}

func TestErrorSuppressor_SuppressRepeated(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := NewErrorSuppressor(
		WithSuppressInterval(time.Hour),
		WithSuppressNowFunc(func() time.Time { return now }),
	)
	err := errors.New("no hwmon for eth2")

	s.Check("nic", err) // first: logged

	now = now.Add(30 * time.Minute) // 30min later
	_, ok := s.Check("nic", err)
	if ok {
		t.Fatal("expected repeated error within interval to be suppressed")
	}
}

func TestErrorSuppressor_LogAfterInterval(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := NewErrorSuppressor(
		WithSuppressInterval(time.Hour),
		WithSuppressNowFunc(func() time.Time { return now }),
	)
	err := errors.New("no hwmon for eth2")

	s.Check("nic", err)

	// Simulate 59 suppressed calls
	for i := 0; i < 59; i++ {
		now = now.Add(time.Minute)
		s.Check("nic", err)
	}

	now = now.Add(time.Minute) // total 60min → interval reached
	msg, ok := s.Check("nic", err)
	if !ok {
		t.Fatal("expected error to be logged after interval")
	}

	expected := "no hwmon for eth2 (suppressed 59 times)"
	if msg != expected {
		t.Fatalf("expected %q, got %q", expected, msg)
	}
}

func TestErrorSuppressor_LogAfterInterval_ZeroSuppressed(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := NewErrorSuppressor(
		WithSuppressInterval(time.Hour),
		WithSuppressNowFunc(func() time.Time { return now }),
	)
	err := errors.New("no hwmon for eth2")

	s.Check("nic", err)

	now = now.Add(time.Hour)
	msg, ok := s.Check("nic", err)
	if !ok {
		t.Fatal("expected error to be logged after interval")
	}
	if msg != "no hwmon for eth2" {
		t.Fatalf("expected raw message, got %q", msg)
	}
}

func TestErrorSuppressor_DifferentError(t *testing.T) {
	s := NewErrorSuppressor()

	s.Check("nic", errors.New("error A"))

	msg, ok := s.Check("nic", errors.New("error B"))
	if !ok {
		t.Fatal("expected different error to be logged immediately")
	}
	if msg != "error B" {
		t.Fatalf("unexpected msg: %s", msg)
	}
}

func TestErrorSuppressor_ErrorCleared(t *testing.T) {
	s := NewErrorSuppressor()
	err := errors.New("no hwmon for eth2")

	s.Check("nic", err)

	// Clear error
	_, ok := s.Check("nic", nil)
	if ok {
		t.Fatal("expected nil error to not be logged")
	}

	// Next error should be logged immediately
	msg, ok := s.Check("nic", err)
	if !ok {
		t.Fatal("expected error after clear to be logged")
	}
	if msg != "no hwmon for eth2" {
		t.Fatalf("unexpected msg: %s", msg)
	}
}

func TestErrorSuppressor_MultipleKeys(t *testing.T) {
	s := NewErrorSuppressor()

	msg1, ok1 := s.Check("nic", errors.New("nic error"))
	msg2, ok2 := s.Check("log", errors.New("log error"))

	if !ok1 || msg1 != "nic error" {
		t.Fatalf("expected nic error logged, got ok=%v msg=%s", ok1, msg1)
	}
	if !ok2 || msg2 != "log error" {
		t.Fatalf("expected log error logged, got ok=%v msg=%s", ok2, msg2)
	}

	// Suppressing one key should not affect another
	_, nicSuppressed := s.Check("nic", errors.New("nic error"))
	_, logSuppressed := s.Check("log", errors.New("log error"))
	if nicSuppressed {
		t.Fatal("expected nic repeated error to be suppressed")
	}
	if logSuppressed {
		t.Fatal("expected log repeated error to be suppressed")
	}
}
