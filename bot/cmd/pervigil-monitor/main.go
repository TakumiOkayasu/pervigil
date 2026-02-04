package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
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
	_ = godotenv.Load()

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

	// Setup signal handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.checkInterval) * time.Second)
	defer ticker.Stop()

	// Run immediately on startup
	runChecks(nicMonitor, logMonitor)

	for {
		select {
		case <-ticker.C:
			runChecks(nicMonitor, logMonitor)
		case sig := <-stop:
			log.Printf("Received %v, shutting down", sig)
			return nil
		}
	}
}

func runChecks(nic *monitor.NICMonitor, lg *monitor.LogMonitor) {
	if err := nic.Check(); err != nil {
		log.Printf("NIC monitor error: %v", err)
	}

	result, err := lg.Process()
	if err != nil {
		log.Printf("Log monitor error: %v", err)
		return
	}
	if result != nil && (result.ErrorCount > 0 || result.WarningCount > 0) {
		log.Printf("Log monitor: %d errors, %d warnings", result.ErrorCount, result.WarningCount)
	}
}

type config struct {
	webhookURL    string
	nicInterface  string
	checkInterval int
	stateFile     string
	logFile       string
	logPosFile    string
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

	return &config{
		webhookURL:    webhookURL,
		nicInterface:  nicInterface,
		checkInterval: checkInterval,
		stateFile:     stateFile,
		logFile:       logFile,
		logPosFile:    logPosFile,
	}, nil
}
