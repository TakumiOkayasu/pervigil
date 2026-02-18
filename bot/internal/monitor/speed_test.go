package monitor

import (
	"errors"
	"reflect"
	"testing"
)

// captureCommandRunner records command invocations for testing.
type captureCommandRunner struct {
	calls [][]string
}

func (r *captureCommandRunner) Run(name string, args ...string) error {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	return nil
}

// failCommandRunner always returns the configured error.
type failCommandRunner struct {
	err error
}

func (r *failCommandRunner) Run(_ string, _ ...string) error {
	return r.err
}

func TestEthtoolSpeedController_Limit(t *testing.T) {
	runner := &captureCommandRunner{}
	ctrl := NewEthtoolSpeedControllerWith(runner)

	if err := ctrl.Limit("eth2"); err != nil {
		t.Fatalf("Limit() error = %v", err)
	}

	want := []string{"ethtool", "-s", "eth2", "speed", speedLimitMbps, "duplex", "full", "autoneg", "off"}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if !reflect.DeepEqual(runner.calls[0], want) {
		t.Errorf("Limit() args = %v, want %v", runner.calls[0], want)
	}
}

func TestEthtoolSpeedController_Limit_Error(t *testing.T) {
	errExec := errors.New("ethtool: command failed")
	ctrl := NewEthtoolSpeedControllerWith(&failCommandRunner{err: errExec})

	err := ctrl.Limit("eth2")
	if !errors.Is(err, errExec) {
		t.Errorf("Limit() error = %v, want %v", err, errExec)
	}
}

func TestEthtoolSpeedController_Restore(t *testing.T) {
	runner := &captureCommandRunner{}
	ctrl := NewEthtoolSpeedControllerWith(runner)

	if err := ctrl.Restore("eth2"); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	want := []string{"ethtool", "-s", "eth2", "autoneg", "on", "advertise", advertiseX540}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if !reflect.DeepEqual(runner.calls[0], want) {
		t.Errorf("Restore() args = %v, want %v", runner.calls[0], want)
	}
}

func TestEthtoolSpeedController_Restore_Error(t *testing.T) {
	errExec := errors.New("ethtool: command failed")
	ctrl := NewEthtoolSpeedControllerWith(&failCommandRunner{err: errExec})

	err := ctrl.Restore("eth2")
	if !errors.Is(err, errExec) {
		t.Errorf("Restore() error = %v, want %v", err, errExec)
	}
}

func TestEthtoolSpeedController_DefaultAdvertise(t *testing.T) {
	runner := &captureCommandRunner{}
	ctrl := NewEthtoolSpeedControllerWith(runner)

	if ctrl.advertise != advertiseX540 {
		t.Errorf("default advertise = %q, want %q", ctrl.advertise, advertiseX540)
	}
}

func TestEthtoolSpeedController_Restore_UsesAdvertiseField(t *testing.T) {
	runner := &captureCommandRunner{}
	ctrl := NewEthtoolSpeedControllerWith(runner)
	ctrl.advertise = "0x0020"

	if err := ctrl.Restore("eth2"); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	want := []string{"ethtool", "-s", "eth2", "autoneg", "on", "advertise", "0x0020"}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if !reflect.DeepEqual(runner.calls[0], want) {
		t.Errorf("Restore() args = %v, want %v", runner.calls[0], want)
	}
}
