package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/murata-lab/pervigil/bot/internal/anthropic"
	"github.com/murata-lab/pervigil/bot/internal/monitor"
	"github.com/murata-lab/pervigil/bot/internal/notifier"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	log.Printf("Starting pervigil-monitor (interval=%ds, iface=%s)", cfg.checkInterval, cfg.nicInterface)

	// Initialize notifier
	discordNotifier := notifier.NewDiscordNotifier(cfg.webhookURL)

	// Initialize NIC monitor
	nicMonitor := monitor.NewNICMonitor(
		monitor.WithTempReader(monitor.NewTempAdapter()),
		monitor.WithNotifier(discordNotifier),
		monitor.WithStateStore(monitor.NewFileStateStore(cfg.stateFile)),
		monitor.WithSpeedController(monitor.NewEthtoolSpeedController()),
		monitor.WithInterface(cfg.nicInterface),
	)

	// Initialize Log monitor
	logMonitor := monitor.NewLogMonitor(
		monitor.WithLogNotifier(discordNotifier),
		monitor.WithLogReader(monitor.NewFileLogReader(cfg.logFile, cfg.logPosFile)),
	)

	// Initialize Cost monitor (optional)
	var costMonitor *monitor.CostMonitor
	if cfg.anthropicKey != "" {
		costMonitor = monitor.NewCostMonitor(
			monitor.WithCostFetcher(anthropic.NewClient(cfg.anthropicKey)),
			monitor.WithCostNotifier(discordNotifier),
			monitor.WithCostStateStore(monitor.NewFileCostStateStore(cfg.costStateFile)),
			monitor.WithCostThresholds(monitor.CostThresholds{
				DailyWarning:  cfg.dailyBudgetWarn,
				DailyCritical: cfg.dailyBudgetCrit,
			}),
		)
		log.Printf("Cost monitor enabled (warn=$%.0f, crit=$%.0f, interval=%ds)",
			cfg.dailyBudgetWarn, cfg.dailyBudgetCrit, cfg.costCheckInterval)
	}

	// Setup signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.checkInterval) * time.Second)
	defer ticker.Stop()

	// Cost monitor has its own interval; nil channel blocks forever in select
	var costCh <-chan time.Time
	if costMonitor != nil {
		costTicker := time.NewTicker(time.Duration(cfg.costCheckInterval) * time.Second)
		defer costTicker.Stop()
		costCh = costTicker.C
	}

	// Run immediately on startup
	runChecks(nicMonitor, logMonitor, costMonitor)

	for {
		select {
		case <-ticker.C:
			runChecks(nicMonitor, logMonitor, nil)
		case <-costCh:
			runChecks(nil, nil, costMonitor)
		case sig := <-stop:
			log.Printf("Received %v, shutting down", sig)
			return nil
		}
	}
}

func runChecks(nic *monitor.NICMonitor, lg *monitor.LogMonitor, cost *monitor.CostMonitor) {
	if nic != nil {
		if err := nic.Check(); err != nil {
			log.Printf("NIC monitor error: %v", err)
		}
	}

	if lg != nil {
		result, err := lg.Process()
		if err != nil {
			log.Printf("Log monitor error: %v", err)
		} else if result != nil && (result.ErrorCount > 0 || result.WarningCount > 0) {
			log.Printf("Log monitor: %d errors, %d warnings", result.ErrorCount, result.WarningCount)
		}
	}

	if cost != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := cost.Check(ctx)
		cancel()
		if err != nil {
			log.Printf("Cost monitor error: %v", err)
		}
	}
}

type config struct {
	webhookURL        string
	nicInterface      string
	checkInterval     int
	stateFile         string
	logFile           string
	logPosFile        string
	anthropicKey      string
	costCheckInterval int
	dailyBudgetWarn   float64
	dailyBudgetCrit   float64
	costStateFile     string
}

func loadConfig() (*config, error) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, fmt.Errorf("DISCORD_WEBHOOK_URL is required")
	}

	nicInterface := os.Getenv("NIC_INTERFACE")
	if nicInterface == "" {
		nicInterface = "eth1"
	}

	checkInterval := 60
	if v := os.Getenv("CHECK_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			checkInterval = i
		}
	}

	stateFile := os.Getenv("STATE_FILE")
	if stateFile == "" {
		stateFile = "/tmp/pervigil-state"
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "/var/log/syslog"
	}

	logPosFile := os.Getenv("LOG_POS_FILE")
	if logPosFile == "" {
		logPosFile = "/tmp/pervigil-log-pos"
	}

	anthropicKey := os.Getenv("ANTHROPIC_ADMIN_KEY")

	costCheckInterval := 3600
	if v := os.Getenv("COST_CHECK_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			costCheckInterval = i
		}
	}

	dailyBudgetWarn := 5.0
	if v := os.Getenv("DAILY_BUDGET_WARN"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			dailyBudgetWarn = f
		}
	}

	dailyBudgetCrit := 10.0
	if v := os.Getenv("DAILY_BUDGET_CRIT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			dailyBudgetCrit = f
		}
	}

	costStateFile := os.Getenv("COST_STATE_FILE")
	if costStateFile == "" {
		costStateFile = "/tmp/pervigil-cost-state"
	}

	return &config{
		webhookURL:        webhookURL,
		nicInterface:      nicInterface,
		checkInterval:     checkInterval,
		stateFile:         stateFile,
		logFile:           logFile,
		logPosFile:        logPosFile,
		anthropicKey:      anthropicKey,
		costCheckInterval: costCheckInterval,
		dailyBudgetWarn:   dailyBudgetWarn,
		dailyBudgetCrit:   dailyBudgetCrit,
		costStateFile:     costStateFile,
	}, nil
}
