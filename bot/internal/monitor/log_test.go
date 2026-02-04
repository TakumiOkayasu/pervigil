package monitor

import (
	"strings"
	"testing"

	"github.com/murata-lab/pervigil/bot/internal/notifier"
)

func TestLogMonitor_MatchesError(t *testing.T) {
	m := NewLogMonitor()

	tests := []struct {
		line    string
		isError bool
	}{
		{"ERROR: something failed", true},
		{"error: test failed", true},
		{"CRITICAL: system down", true},
		{"panic: nil pointer", true},
		{"FAILED to start", true},
		{"info: all good", false},
		{"debug: trace message", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if m.matchesError(tt.line) != tt.isError {
				t.Errorf("matchesError(%q) = %v, want %v", tt.line, !tt.isError, tt.isError)
			}
		})
	}
}

func TestLogMonitor_MatchesWarning(t *testing.T) {
	m := NewLogMonitor()

	tests := []struct {
		line      string
		isWarning bool
	}{
		{"WARNING: disk usage high", true},
		{"warn: low memory", true},
		{"WARN: connection slow", true},
		{"info: all good", false},
		{"error: failed", false}, // error, not warning
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if m.matchesWarning(tt.line) != tt.isWarning {
				t.Errorf("matchesWarning(%q) = %v, want %v", tt.line, !tt.isWarning, tt.isWarning)
			}
		})
	}
}

func TestLogMonitor_ShouldExclude(t *testing.T) {
	m := NewLogMonitor()

	tests := []struct {
		line    string
		exclude bool
	}{
		{"DHCP4_BUFFER_RECEIVE_FAIL...Truncated", true},
		{"netlink-dp error Network is down", true},
		{"pam_unix: authentication failure for user", true},
		{"ERROR: real problem here", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if m.shouldExclude(tt.line) != tt.exclude {
				t.Errorf("shouldExclude(%q) = %v, want %v", tt.line, !tt.exclude, tt.exclude)
			}
		})
	}
}

type mockLogReader struct {
	lines []string
}

func (m *mockLogReader) ReadNewLines() ([]string, error) {
	return m.lines, nil
}

func TestLogMonitor_ProcessLines(t *testing.T) {
	notif := &mockNotifier{}
	reader := &mockLogReader{
		lines: []string{
			"ERROR: critical failure",
			"info: normal message",
			"WARNING: disk almost full",
			"DHCP4_BUFFER_RECEIVE_FAIL...Truncated", // excluded
		},
	}

	m := NewLogMonitor(
		WithLogNotifier(notif),
		WithLogReader(reader),
	)

	result, err := m.Process()
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", result.ErrorCount)
	}
	if result.WarningCount != 1 {
		t.Errorf("WarningCount = %d, want 1", result.WarningCount)
	}
}

func TestLogMonitor_NotifiesOnError(t *testing.T) {
	notif := &mockNotifier{}
	reader := &mockLogReader{
		lines: []string{
			"ERROR: test error",
		},
	}

	m := NewLogMonitor(
		WithLogNotifier(notif),
		WithLogReader(reader),
	)

	_, err := m.Process()
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorRed {
		t.Errorf("color = %v, want Red", notif.calls[0].color)
	}
}

func TestLogMonitor_NoNotifyOnWarningUnderThreshold(t *testing.T) {
	notif := &mockNotifier{}
	reader := &mockLogReader{
		lines: []string{
			"WARNING: single warning",
		},
	}

	m := NewLogMonitor(
		WithLogNotifier(notif),
		WithLogReader(reader),
	)

	_, err := m.Process()
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	// No notification for < 5 warnings
	if len(notif.calls) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(notif.calls))
	}
}

func TestLogMonitor_NotifiesOnManyWarnings(t *testing.T) {
	notif := &mockNotifier{}
	lines := make([]string, 5)
	for i := range lines {
		lines[i] = "WARNING: warning message"
	}
	reader := &mockLogReader{lines: lines}

	m := NewLogMonitor(
		WithLogNotifier(notif),
		WithLogReader(reader),
	)

	_, err := m.Process()
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(notif.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notif.calls))
	}
	if notif.calls[0].color != notifier.ColorYellow {
		t.Errorf("color = %v, want Yellow", notif.calls[0].color)
	}
	if !strings.Contains(notif.calls[0].title, "警告") {
		t.Errorf("title should contain 警告")
	}
}
