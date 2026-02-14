package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/murata-lab/pervigil/bot/internal/anthropic"
)

var (
	claudeClientOnce sync.Once
	claudeClientInst *anthropic.Client

	thresholdOnce           sync.Once
	cachedWarnThreshold     float64
	cachedCriticalThreshold float64
)

func getClaudeClient(apiKey string) *anthropic.Client {
	claudeClientOnce.Do(func() {
		claudeClientInst = anthropic.NewClient(apiKey)
	})
	return claudeClientInst
}

func cmdClaude(s *discordgo.Session, i *discordgo.InteractionCreate) {
	apiKey := os.Getenv("ANTHROPIC_ADMIN_KEY")
	if apiKey == "" {
		respond(s, i, "ANTHROPIC_ADMIN_KEY が未設定です")
		return
	}

	if err := deferredRespond(s, i); err != nil {
		return
	}

	client := getClaudeClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)

	warnThreshold, critThreshold := costThresholds()

	type costResult struct {
		report *anthropic.CostReport
		err    error
	}
	type usageResult struct {
		report *anthropic.UsageReport
		err    error
	}

	dailyCh := make(chan costResult, 1)
	monthlyCh := make(chan costResult, 1)
	usageCh := make(chan usageResult, 1)

	go func() {
		r, err := client.GetCost(ctx, today, tomorrow)
		dailyCh <- costResult{r, err}
	}()
	go func() {
		r, err := client.GetCost(ctx, monthStart, tomorrow)
		monthlyCh <- costResult{r, err}
	}()
	go func() {
		r, err := client.GetUsage(ctx, monthStart, tomorrow, "model")
		usageCh <- usageResult{r, err}
	}()

	daily := <-dailyCh
	monthly := <-monthlyCh
	usage := <-usageCh

	var sb strings.Builder
	sb.WriteString("**Claude API 利用状況**\n```\n")

	// Daily cost
	if daily.err != nil {
		log.Printf("daily cost fetch error: %v", daily.err)
		fmt.Fprintf(&sb, "本日コスト: 取得エラー\n")
	} else {
		cost := sumCost(daily.report)
		fmt.Fprintf(&sb, "本日コスト: $%.2f %s\n", cost, statusIndicator(cost, warnThreshold, critThreshold))
	}

	// Monthly cost
	if monthly.err != nil {
		log.Printf("monthly cost fetch error: %v", monthly.err)
		fmt.Fprintf(&sb, "今月コスト: 取得エラー\n")
	} else {
		cost := sumCost(monthly.report)
		fmt.Fprintf(&sb, "今月コスト: $%.2f\n", cost)
	}

	// Usage by model
	if usage.err != nil {
		log.Printf("usage fetch error: %v", usage.err)
		fmt.Fprintf(&sb, "\nモデル別使用量: 取得エラー\n")
	} else if len(usage.report.Data) > 0 {
		sb.WriteString("\nモデル別使用量 (今月):\n")
		type modelUsage struct {
			input  int64
			output int64
		}
		models := make(map[string]*modelUsage)
		var order []string
		for _, b := range usage.report.Data {
			if _, ok := models[b.Model]; !ok {
				models[b.Model] = &modelUsage{}
				order = append(order, b.Model)
			}
			models[b.Model].input += b.InputTokens
			models[b.Model].output += b.OutputTokens
		}
		maxLen := 0
		for _, name := range order {
			if len(name) > maxLen {
				maxLen = len(name)
			}
		}
		for _, name := range order {
			u := models[name]
			fmt.Fprintf(&sb, "  %-*s In: %s / Out: %s\n", maxLen, name, formatTokens(u.input), formatTokens(u.output))
		}
	}

	sb.WriteString("```")
	followup(s, i, sb.String())
}

func costThresholds() (warn, crit float64) {
	thresholdOnce.Do(func() {
		cachedWarnThreshold = 5.0
		if v := os.Getenv("DAILY_BUDGET_WARN"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				cachedWarnThreshold = f
			}
		}
		cachedCriticalThreshold = 10.0
		if v := os.Getenv("DAILY_BUDGET_CRIT"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				cachedCriticalThreshold = f
			}
		}
	})
	return cachedWarnThreshold, cachedCriticalThreshold
}

func sumCost(report *anthropic.CostReport) float64 {
	var total float64
	for _, b := range report.Data {
		total += b.CostUSD
	}
	return total
}

func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
