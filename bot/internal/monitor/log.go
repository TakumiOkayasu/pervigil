package monitor

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/murata-lab/pervigil/bot/internal/notifier"
)

// LogReader reads new log lines
type LogReader interface {
	ReadNewLines() ([]string, error)
}

// ProcessResult contains the results of log processing
type ProcessResult struct {
	ErrorCount   int
	WarningCount int
	ErrorLines   []string
	WarningLines []string
}

// LogMonitor monitors log files for errors and warnings
type LogMonitor struct {
	errorPatterns    []*regexp.Regexp
	warningPatterns  []*regexp.Regexp
	excludePatterns  []*regexp.Regexp
	notifier         notifier.Notifier
	reader           LogReader
	hostname         string
	warningThreshold int
}

// LogOption configures LogMonitor
type LogOption func(*LogMonitor)

// WithLogNotifier sets the notifier
func WithLogNotifier(n notifier.Notifier) LogOption {
	return func(m *LogMonitor) {
		m.notifier = n
	}
}

// WithLogReader sets the log reader
func WithLogReader(r LogReader) LogOption {
	return func(m *LogMonitor) {
		m.reader = r
	}
}

// WithWarningThreshold sets the warning notification threshold
func WithWarningThreshold(n int) LogOption {
	return func(m *LogMonitor) {
		m.warningThreshold = n
	}
}

// NewLogMonitor creates a new log monitor with default patterns
func NewLogMonitor(opts ...LogOption) *LogMonitor {
	hostname, _ := os.Hostname()
	m := &LogMonitor{
		errorPatterns: compilePatterns([]string{
			`(?i)error`,
			`(?i)failed`,
			`(?i)critical`,
			`(?i)panic`,
		}),
		warningPatterns: compilePatterns([]string{
			`(?i)warning`,
			`(?i)\bwarn\b`,
		}),
		excludePatterns: compilePatterns([]string{
			`DHCP4_BUFFER_RECEIVE_FAIL.*Truncated`,
			`netlink-dp.*Network is down`,
			`pam_unix.*authentication failure`,
		}),
		hostname:         hostname,
		warningThreshold: 5,
	}

	for _, opt := range opts {
		opt(m)
	}
	return m
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err == nil {
			result = append(result, re)
		}
	}
	return result
}

func (m *LogMonitor) matchesError(line string) bool {
	return matchesAny(line, m.errorPatterns)
}

func (m *LogMonitor) matchesWarning(line string) bool {
	return matchesAny(line, m.warningPatterns)
}

func (m *LogMonitor) shouldExclude(line string) bool {
	return matchesAny(line, m.excludePatterns)
}

func matchesAny(line string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// Process reads and processes new log lines
func (m *LogMonitor) Process() (*ProcessResult, error) {
	lines, err := m.reader.ReadNewLines()
	if err != nil {
		return nil, fmt.Errorf("read lines: %w", err)
	}

	result := &ProcessResult{}

	for _, line := range lines {
		if line == "" {
			continue
		}
		if m.shouldExclude(line) {
			continue
		}

		if m.matchesError(line) {
			result.ErrorCount++
			result.ErrorLines = append(result.ErrorLines, line)
			continue
		}

		if m.matchesWarning(line) {
			result.WarningCount++
			result.WarningLines = append(result.WarningLines, line)
		}
	}

	if err := m.sendNotifications(result); err != nil {
		return result, err
	}

	return result, nil
}

func (m *LogMonitor) sendNotifications(result *ProcessResult) error {
	if result.ErrorCount > 0 {
		truncated := truncateLines(result.ErrorLines, 1000)
		fields := []notifier.Field{
			{Name: "Error Count", Value: fmt.Sprintf("%d", result.ErrorCount), Inline: true},
		}
		if err := m.notifier.Send(
			fmt.Sprintf("🚨 ログエラー検出 - %s", m.hostname),
			fmt.Sprintf("```\n%s\n```", truncated),
			notifier.ColorRed,
			fields,
		); err != nil {
			return fmt.Errorf("send error notification: %w", err)
		}
	}

	// Only notify for warnings if no errors and threshold exceeded
	if result.WarningCount >= m.warningThreshold && result.ErrorCount == 0 {
		fields := []notifier.Field{
			{Name: "Warning Count", Value: fmt.Sprintf("%d", result.WarningCount), Inline: true},
		}
		if err := m.notifier.Send(
			fmt.Sprintf("⚠️ ログ警告 - %s", m.hostname),
			fmt.Sprintf("過去の監視期間に%d件の警告が検出されました。", result.WarningCount),
			notifier.ColorYellow,
			fields,
		); err != nil {
			return fmt.Errorf("send warning notification: %w", err)
		}
	}

	return nil
}

func truncateLines(lines []string, maxLen int) string {
	joined := strings.Join(lines, "\n")
	if len(joined) > maxLen {
		return joined[:maxLen] + "..."
	}
	return joined
}
