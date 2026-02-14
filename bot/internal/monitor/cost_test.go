package monitor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/murata-lab/pervigil/bot/internal/anthropic"
	"github.com/murata-lab/pervigil/bot/internal/notifier"
)

type mockFetcher struct {
	cost float64
	err  error
}

func (m *mockFetcher) GetCost(_ context.Context, _, _ time.Time) (*anthropic.CostReport, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &anthropic.CostReport{
		Data: []anthropic.CostBucket{{Date: "2025-01-15", CostUSD: m.cost}},
	}, nil
}

type mockCostNotifier struct {
	calls []string
}

func (m *mockCostNotifier) Send(title, _ string, _ notifier.Color, _ []notifier.Field) error {
	m.calls = append(m.calls, title)
	return nil
}

type mockCostStateStore struct {
	state   CostStateData
	saveErr error
}

func (m *mockCostStateStore) LoadCost() (CostStateData, error) {
	return m.state, nil
}

func (m *mockCostStateStore) SaveCost(s CostStateData) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.state = s
	return nil
}

// fixedNow returns a deterministic time source for testing.
func fixedNow() time.Time {
	return time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
}

const fixedDate = "2025-01-15"

func TestCostMonitor_NormalToWarning(t *testing.T) {
	n := &mockCostNotifier{}
	ss := &mockCostStateStore{state: CostStateData{State: CostNormal, Date: fixedDate}}

	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{cost: 6.0}),
		WithCostNotifier(n),
		WithCostStateStore(ss),
		WithCostThresholds(CostThresholds{DailyWarning: 5.0, DailyCritical: 10.0}),
		WithCostNowFunc(fixedNow),
	)

	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(n.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(n.calls))
	}
	if !strings.Contains(n.calls[0], "コスト警告") {
		t.Errorf("notification title = %q, want containing %q", n.calls[0], "コスト警告")
	}
	if ss.state.State != CostWarning {
		t.Errorf("state = %s, want %s", ss.state.State, CostWarning)
	}
}

func TestCostMonitor_NormalToCritical(t *testing.T) {
	n := &mockCostNotifier{}
	ss := &mockCostStateStore{state: CostStateData{State: CostNormal, Date: fixedDate}}

	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{cost: 12.0}),
		WithCostNotifier(n),
		WithCostStateStore(ss),
		WithCostThresholds(CostThresholds{DailyWarning: 5.0, DailyCritical: 10.0}),
		WithCostNowFunc(fixedNow),
	)

	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(n.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(n.calls))
	}
	if !strings.Contains(n.calls[0], "コスト危険") {
		t.Errorf("notification title = %q, want containing %q", n.calls[0], "コスト危険")
	}
	if ss.state.State != CostCritical {
		t.Errorf("state = %s, want %s", ss.state.State, CostCritical)
	}
}

func TestCostMonitor_NoTransition(t *testing.T) {
	n := &mockCostNotifier{}
	ss := &mockCostStateStore{state: CostStateData{State: CostWarning, Date: fixedDate}}

	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{cost: 7.0}),
		WithCostNotifier(n),
		WithCostStateStore(ss),
		WithCostThresholds(CostThresholds{DailyWarning: 5.0, DailyCritical: 10.0}),
		WithCostNowFunc(fixedNow),
	)

	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(n.calls) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(n.calls))
	}
}

func TestCostMonitor_NewDayResetsState(t *testing.T) {
	n := &mockCostNotifier{}
	ss := &mockCostStateStore{state: CostStateData{State: CostCritical, Date: "2020-01-01"}}

	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{cost: 1.0}),
		WithCostNotifier(n),
		WithCostStateStore(ss),
		WithCostThresholds(CostThresholds{DailyWarning: 5.0, DailyCritical: 10.0}),
		WithCostNowFunc(fixedNow),
	)

	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ss.state.State != CostNormal {
		t.Errorf("state = %s, want %s", ss.state.State, CostNormal)
	}
	if len(n.calls) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(n.calls))
	}
}

func TestCostMonitor_FetchError(t *testing.T) {
	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{err: errors.New("api down")}),
		WithCostNotifier(&mockCostNotifier{}),
		WithCostStateStore(&mockCostStateStore{state: CostStateData{State: CostNormal}}),
		WithCostNowFunc(fixedNow),
	)

	if err := m.Check(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCostMonitor_SaveError(t *testing.T) {
	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{cost: 1.0}),
		WithCostNotifier(&mockCostNotifier{}),
		WithCostStateStore(&mockCostStateStore{
			state:   CostStateData{State: CostNormal, Date: fixedDate},
			saveErr: errors.New("disk full"),
		}),
		WithCostNowFunc(fixedNow),
	)

	err := m.Check(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "save state") {
		t.Errorf("error = %q, want containing %q", err.Error(), "save state")
	}
}

func TestCostMonitor_CriticalToNormal(t *testing.T) {
	n := &mockCostNotifier{}
	ss := &mockCostStateStore{state: CostStateData{State: CostCritical, Date: fixedDate}}

	m := NewCostMonitor(
		WithCostFetcher(&mockFetcher{cost: 2.0}),
		WithCostNotifier(n),
		WithCostStateStore(ss),
		WithCostThresholds(CostThresholds{DailyWarning: 5.0, DailyCritical: 10.0}),
		WithCostNowFunc(fixedNow),
	)

	if err := m.Check(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(n.calls) != 1 {
		t.Fatalf("expected 1 notification (recovery), got %d", len(n.calls))
	}
	if !strings.Contains(n.calls[0], "コスト正常化") {
		t.Errorf("notification title = %q, want containing %q", n.calls[0], "コスト正常化")
	}
	if ss.state.State != CostNormal {
		t.Errorf("state = %s, want %s", ss.state.State, CostNormal)
	}
}
