package monitor

import (
	"fmt"
	"os"

	"github.com/murata-lab/pervigil/bot/internal/notifier"
	"github.com/murata-lab/pervigil/bot/internal/temperature"
)

// NICState represents the monitor state
type NICState string

const (
	StateNormal   NICState = "normal"
	StateWarning  NICState = "warning"
	StateCritical NICState = "critical"
)

// NICThresholds defines temperature thresholds
type NICThresholds struct {
	Warning  float64
	Critical float64
	Recovery float64
}

// DefaultThresholds returns the default NIC temperature thresholds
func DefaultThresholds() NICThresholds {
	return NICThresholds{
		Warning:  70.0,
		Critical: 85.0,
		Recovery: 65.0,
	}
}

// tempReader abstracts temperature reading
type tempReader interface {
	GetNICTemp(iface string) (*temperature.TempReading, error)
}

// MonitorState holds both temperature state and speed limit status
type MonitorState struct {
	TempState    NICState `json:"temp_state"`
	SpeedLimited bool     `json:"speed_limited"`
}

// StateStore persists monitor state
type StateStore interface {
	Load() (MonitorState, error)
	Save(MonitorState) error
}

// SpeedController controls NIC speed
type SpeedController interface {
	Limit(iface string) error
	Restore(iface string) error
}

// NICMonitor monitors NIC temperature and takes action
type NICMonitor struct {
	tempReader tempReader
	notifier   notifier.Notifier
	stateStore StateStore
	speedCtrl  SpeedController
	thresholds NICThresholds
	iface      string
	hostname   string
}

// NICOption configures NICMonitor
type NICOption func(*NICMonitor)

// WithTempReader sets the temperature reader
func WithTempReader(r tempReader) NICOption {
	return func(m *NICMonitor) {
		m.tempReader = r
	}
}

// WithNotifier sets the notifier
func WithNotifier(n notifier.Notifier) NICOption {
	return func(m *NICMonitor) {
		m.notifier = n
	}
}

// WithStateStore sets the state store
func WithStateStore(s StateStore) NICOption {
	return func(m *NICMonitor) {
		m.stateStore = s
	}
}

// WithSpeedController sets the speed controller
func WithSpeedController(c SpeedController) NICOption {
	return func(m *NICMonitor) {
		m.speedCtrl = c
	}
}

// WithInterface sets the NIC interface
func WithInterface(iface string) NICOption {
	return func(m *NICMonitor) {
		m.iface = iface
	}
}

// WithThresholds sets custom thresholds
func WithThresholds(t NICThresholds) NICOption {
	return func(m *NICMonitor) {
		m.thresholds = t
	}
}

// NewNICMonitor creates a new NIC monitor
func NewNICMonitor(opts ...NICOption) *NICMonitor {
	hostname, _ := os.Hostname()
	m := &NICMonitor{
		thresholds: DefaultThresholds(),
		iface:      "eth1",
		hostname:   hostname,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Check performs a temperature check and takes appropriate action
func (m *NICMonitor) Check() error {
	reading, err := m.tempReader.GetNICTemp(m.iface)
	if err != nil {
		return fmt.Errorf("read temperature: %w", err)
	}

	state, err := m.stateStore.Load()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	newTempState := m.determineState(reading.Value)
	newState, err := m.handleTransition(state, newTempState, reading.Value)
	if err != nil {
		return err
	}

	return m.stateStore.Save(newState)
}

func (m *NICMonitor) determineState(temp float64) NICState {
	if temp >= m.thresholds.Critical {
		return StateCritical
	}
	if temp >= m.thresholds.Warning {
		return StateWarning
	}
	return StateNormal
}

func (m *NICMonitor) handleTransition(current MonitorState, newTempState NICState, temp float64) (MonitorState, error) {
	newState := MonitorState{TempState: newTempState, SpeedLimited: current.SpeedLimited}

	switch {
	case newTempState == StateCritical && current.TempState != StateCritical:
		if err := m.notifier.Send(
			fmt.Sprintf("🔥 NIC過熱警報 - %s", m.hostname),
			"NIC温度が危険域に達しました。速度を1Gbpsに制限します。",
			notifier.ColorRed,
			m.makeFields(temp, "Threshold", m.thresholds.Critical, "Action", "Speed limited to 1Gbps"),
		); err != nil {
			return newState, fmt.Errorf("send notification: %w", err)
		}
		if err := m.speedCtrl.Limit(m.iface); err != nil {
			return newState, err
		}
		newState.SpeedLimited = true

	case newTempState == StateWarning && current.TempState == StateNormal:
		if err := m.notifier.Send(
			fmt.Sprintf("⚠️ NIC温度警告 - %s", m.hostname),
			"NIC温度が警告域に達しました。監視を継続します。",
			notifier.ColorYellow,
			m.makeFields(temp, "Warning Threshold", m.thresholds.Warning, "Critical Threshold", m.thresholds.Critical),
		); err != nil {
			return newState, err
		}

	case current.SpeedLimited && temp <= m.thresholds.Recovery:
		// Restore speed when below recovery threshold
		if err := m.notifier.Send(
			fmt.Sprintf("✅ NIC温度正常化 - %s", m.hostname),
			"NIC温度が正常範囲に戻りました。速度制限を解除します。",
			notifier.ColorGreen,
			m.makeFields(temp, "Action", "Speed restored to auto"),
		); err != nil {
			return newState, fmt.Errorf("send notification: %w", err)
		}
		if err := m.speedCtrl.Restore(m.iface); err != nil {
			return newState, err
		}
		newState.SpeedLimited = false

	case newTempState == StateNormal && current.TempState == StateWarning:
		if err := m.notifier.Send(
			fmt.Sprintf("✅ NIC温度正常化 - %s", m.hostname),
			"NIC温度が正常範囲に戻りました。",
			notifier.ColorGreen,
			m.makeFields(temp),
		); err != nil {
			return newState, err
		}
	}

	return newState, nil
}

func (m *NICMonitor) makeFields(temp float64, extra ...any) []notifier.Field {
	fields := []notifier.Field{
		{Name: "Temperature", Value: fmt.Sprintf("%.1f°C", temp), Inline: true},
	}
	for i := 0; i+1 < len(extra); i += 2 {
		name, _ := extra[i].(string)
		var value string
		switch v := extra[i+1].(type) {
		case string:
			value = v
		case float64:
			value = fmt.Sprintf("%.0f°C", v)
		}
		fields = append(fields, notifier.Field{Name: name, Value: value, Inline: true})
	}
	return fields
}
