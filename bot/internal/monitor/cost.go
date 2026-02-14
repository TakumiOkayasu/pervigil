package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/murata-lab/pervigil/bot/internal/anthropic"
	"github.com/murata-lab/pervigil/bot/internal/notifier"
)

// CostState represents cost monitor state.
type CostState string

const (
	CostNormal   CostState = "normal"
	CostWarning  CostState = "warning"
	CostCritical CostState = "critical"
)

// CostThresholds defines cost alert thresholds in USD.
type CostThresholds struct {
	DailyWarning  float64
	DailyCritical float64
}

// DefaultCostThresholds returns sensible default thresholds.
func DefaultCostThresholds() CostThresholds {
	return CostThresholds{
		DailyWarning:  5.0,
		DailyCritical: 10.0,
	}
}

// UsageFetcher abstracts cost data retrieval.
type UsageFetcher interface {
	GetCost(ctx context.Context, start, end time.Time) (*anthropic.CostReport, error)
}

// CostStateData holds persisted cost monitor state.
type CostStateData struct {
	State CostState `json:"state"`
	Date  string    `json:"date"`
}

// CostStateStore persists cost monitor state.
type CostStateStore interface {
	LoadCost() (CostStateData, error)
	SaveCost(CostStateData) error
}

// FileCostStateStore persists cost state to a file.
type FileCostStateStore struct {
	path string
}

// NewFileCostStateStore creates a new file-based cost state store.
func NewFileCostStateStore(path string) *FileCostStateStore {
	return &FileCostStateStore{path: path}
}

// LoadCost reads cost state from file.
func (s *FileCostStateStore) LoadCost() (CostStateData, error) {
	def := CostStateData{State: CostNormal, Date: ""}

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return def, nil
	}
	if err != nil {
		return def, err
	}

	var state CostStateData
	if err := json.Unmarshal(data, &state); err != nil {
		return def, nil
	}

	// Validate State value
	if state.State != CostNormal && state.State != CostWarning && state.State != CostCritical {
		state.State = CostNormal
	}

	return state, nil
}

// SaveCost writes cost state to file.
func (s *FileCostStateStore) SaveCost(state CostStateData) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// CostMonitor monitors Claude API costs and alerts on threshold breaches.
type CostMonitor struct {
	fetcher    UsageFetcher
	notifier   notifier.Notifier
	stateStore CostStateStore
	thresholds CostThresholds
	hostname   string
	nowFunc    func() time.Time
}

// CostOption configures CostMonitor.
type CostOption func(*CostMonitor)

// WithCostFetcher sets the usage fetcher.
func WithCostFetcher(f UsageFetcher) CostOption {
	return func(m *CostMonitor) {
		m.fetcher = f
	}
}

// WithCostNotifier sets the notifier.
func WithCostNotifier(n notifier.Notifier) CostOption {
	return func(m *CostMonitor) {
		m.notifier = n
	}
}

// WithCostStateStore sets the state store.
func WithCostStateStore(s CostStateStore) CostOption {
	return func(m *CostMonitor) {
		m.stateStore = s
	}
}

// WithCostThresholds sets custom thresholds.
func WithCostThresholds(t CostThresholds) CostOption {
	return func(m *CostMonitor) {
		m.thresholds = t
	}
}

// WithCostNowFunc sets a custom time source (for testing).
func WithCostNowFunc(f func() time.Time) CostOption {
	return func(m *CostMonitor) {
		m.nowFunc = f
	}
}

// NewCostMonitor creates a new cost monitor.
func NewCostMonitor(opts ...CostOption) *CostMonitor {
	hostname, _ := os.Hostname()
	m := &CostMonitor{
		thresholds: DefaultCostThresholds(),
		hostname:   hostname,
		nowFunc:    time.Now,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Check fetches current cost and sends notifications on state transitions.
func (m *CostMonitor) Check(ctx context.Context) error {
	now := m.nowFunc()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	todayStr := today.Format("2006-01-02")

	report, err := m.fetcher.GetCost(ctx, today, tomorrow)
	if err != nil {
		return fmt.Errorf("fetch cost: %w", err)
	}

	var dailyCost float64
	for _, b := range report.Data {
		dailyCost += b.CostUSD
	}

	prev, err := m.stateStore.LoadCost()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	// Reset state on new day
	if prev.Date != todayStr {
		prev = CostStateData{State: CostNormal, Date: todayStr}
	}

	newState := m.determineState(dailyCost)

	if newState != prev.State {
		if err := m.sendTransition(prev.State, newState, dailyCost); err != nil {
			return fmt.Errorf("send notification: %w", err)
		}
	}

	if err := m.stateStore.SaveCost(CostStateData{State: newState, Date: todayStr}); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	return nil
}

func (m *CostMonitor) determineState(cost float64) CostState {
	switch {
	case cost >= m.thresholds.DailyCritical:
		return CostCritical
	case cost >= m.thresholds.DailyWarning:
		return CostWarning
	default:
		return CostNormal
	}
}

func (m *CostMonitor) sendTransition(from, to CostState, cost float64) error {
	fields := []notifier.Field{
		{Name: "Daily Cost", Value: fmt.Sprintf("$%.2f", cost), Inline: true},
		{Name: "Warning", Value: fmt.Sprintf("$%.2f", m.thresholds.DailyWarning), Inline: true},
		{Name: "Critical", Value: fmt.Sprintf("$%.2f", m.thresholds.DailyCritical), Inline: true},
	}

	switch to {
	case CostCritical:
		return m.notifier.Send(
			fmt.Sprintf("🔴 Claude API コスト危険 - %s", m.hostname),
			"日次コストが危険閾値を超過しました。",
			notifier.ColorRed,
			fields,
		)
	case CostWarning:
		return m.notifier.Send(
			fmt.Sprintf("🟡 Claude API コスト警告 - %s", m.hostname),
			"日次コストが警告閾値を超過しました。",
			notifier.ColorYellow,
			fields,
		)
	case CostNormal:
		if from != CostNormal {
			return m.notifier.Send(
				fmt.Sprintf("🟢 Claude API コスト正常化 - %s", m.hostname),
				"日次コストが正常範囲に戻りました。",
				notifier.ColorGreen,
				fields,
			)
		}
	}
	return nil
}
