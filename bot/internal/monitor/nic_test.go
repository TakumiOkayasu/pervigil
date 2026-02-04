package monitor

import (
	"errors"
	"testing"

	"github.com/murata-lab/pervigil/bot/internal/notifier"
	"github.com/murata-lab/pervigil/bot/internal/temperature"
)

type mockTempReader struct {
	temp float64
	err  error
}

func (m *mockTempReader) GetNICTemp(iface string) (*temperature.TempReading, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &temperature.TempReading{Label: iface, Value: m.temp}, nil
}

type mockNotifier struct {
	calls []notifyCall
}

type notifyCall struct {
	title   string
	message string
	color   notifier.Color
	fields  []notifier.Field
}

func (m *mockNotifier) Send(title, message string, color notifier.Color, fields []notifier.Field) error {
	m.calls = append(m.calls, notifyCall{title, message, color, fields})
	return nil
}

type mockStateStore struct {
	state MonitorState
}

func (m *mockStateStore) Load() (MonitorState, error) {
	if m.state.TempState == "" {
		return MonitorState{TempState: StateNormal}, nil
	}
	return m.state, nil
}

func (m *mockStateStore) Save(s MonitorState) error {
	m.state = s
	return nil
}

type mockSpeedController struct {
	limited  bool
	restored bool
}

func (m *mockSpeedController) Limit(iface string) error {
	m.limited = true
	return nil
}

func (m *mockSpeedController) Restore(iface string) error {
	m.restored = true
	return nil
}

func TestNICMonitor_Check_Normal(t *testing.T) {
	temp := &mockTempReader{temp: 50.0}
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateNormal}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(notif.calls) != 0 {
		t.Errorf("expected no notifications, got %d", len(notif.calls))
	}
	if store.state.TempState != StateNormal {
		t.Errorf("state = %v, want %v", store.state.TempState, StateNormal)
	}
}

func TestNICMonitor_Check_NormalToWarning(t *testing.T) {
	temp := &mockTempReader{temp: 75.0}
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateNormal}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorYellow {
		t.Errorf("color = %v, want Yellow", notif.calls[0].color)
	}
	if store.state.TempState != StateWarning {
		t.Errorf("state = %v, want %v", store.state.TempState, StateWarning)
	}
}

func TestNICMonitor_Check_NormalToCritical(t *testing.T) {
	temp := &mockTempReader{temp: 90.0}
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateNormal}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorRed {
		t.Errorf("color = %v, want Red", notif.calls[0].color)
	}
	if store.state.TempState != StateCritical {
		t.Errorf("state = %v, want %v", store.state.TempState, StateCritical)
	}
	if !speed.limited {
		t.Error("expected speed to be limited")
	}
	if !store.state.SpeedLimited {
		t.Error("expected SpeedLimited to be true")
	}
}

func TestNICMonitor_Check_CriticalToNormal(t *testing.T) {
	temp := &mockTempReader{temp: 60.0} // below recovery threshold
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateCritical, SpeedLimited: true}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorGreen {
		t.Errorf("color = %v, want Green", notif.calls[0].color)
	}
	if store.state.TempState != StateNormal {
		t.Errorf("state = %v, want %v", store.state.TempState, StateNormal)
	}
	if !speed.restored {
		t.Error("expected speed to be restored")
	}
	if store.state.SpeedLimited {
		t.Error("expected SpeedLimited to be false")
	}
}

func TestNICMonitor_Check_CriticalToNormal_AboveRecovery(t *testing.T) {
	// Temperature between recovery(65) and warning(70) - should NOT restore speed yet
	temp := &mockTempReader{temp: 67.0}
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateCritical, SpeedLimited: true}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	// No notification or speed restore when above recovery threshold
	if len(notif.calls) != 0 {
		t.Errorf("expected no notifications, got %d", len(notif.calls))
	}
	if speed.restored {
		t.Error("speed should NOT be restored above recovery threshold")
	}
	// State transitions to Normal but speed stays limited
	if store.state.TempState != StateNormal {
		t.Errorf("state = %v, want %v", store.state.TempState, StateNormal)
	}
	if !store.state.SpeedLimited {
		t.Error("SpeedLimited should remain true")
	}
}

func TestNICMonitor_Check_SpeedRestoreAfterRecovery(t *testing.T) {
	// After temp drops below recovery, speed should be restored even if TempState is already Normal
	temp := &mockTempReader{temp: 60.0}
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateNormal, SpeedLimited: true}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !speed.restored {
		t.Error("expected speed to be restored")
	}
	if store.state.SpeedLimited {
		t.Error("expected SpeedLimited to be false")
	}
	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorGreen {
		t.Errorf("color = %v, want Green", notif.calls[0].color)
	}
}

func TestNICMonitor_Check_WarningToNormal(t *testing.T) {
	temp := &mockTempReader{temp: 50.0}
	notif := &mockNotifier{}
	store := &mockStateStore{state: MonitorState{TempState: StateWarning}}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorGreen {
		t.Errorf("color = %v, want Green", notif.calls[0].color)
	}
	if store.state.TempState != StateNormal {
		t.Errorf("state = %v, want %v", store.state.TempState, StateNormal)
	}
}

func TestNICMonitor_Check_TempReadError(t *testing.T) {
	temp := &mockTempReader{err: errors.New("sensor error")}
	notif := &mockNotifier{}
	store := &mockStateStore{}
	speed := &mockSpeedController{}

	m := NewNICMonitor(
		WithTempReader(temp),
		WithNotifier(notif),
		WithStateStore(store),
		WithSpeedController(speed),
	)

	err := m.Check()
	if err == nil {
		t.Error("expected error for sensor failure")
	}
}
