package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/murata-lab/pervigil/bot/internal/temperature"
)

// NICInfo contains network interface information.
type NICInfo struct {
	Name      string
	State     string  // up/down
	Speed     string  // 10000Mb/s etc
	Temp      float64 // Temperature (if available)
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
	RxErrors  uint64
	TxErrors  uint64
}

// GetMonitoredNICs returns the list of NICs to monitor.
// Uses MONITOR_NICS env var (comma-separated) or defaults to eth0,eth1,eth2.
func GetMonitoredNICs() []string {
	if env := os.Getenv("MONITOR_NICS"); env != "" {
		return strings.Split(env, ",")
	}
	return []string{"eth0", "eth1", "eth2"}
}

// GetNICInfo returns information for a single NIC.
func GetNICInfo(iface string) (*NICInfo, error) {
	basePath := filepath.Join("/sys/class/net", iface)

	// Check if interface exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, err
	}

	info := &NICInfo{Name: iface}

	// State (operstate)
	if data, err := os.ReadFile(filepath.Join(basePath, "operstate")); err == nil {
		info.State = strings.TrimSpace(string(data))
	}

	// Speed
	if data, err := os.ReadFile(filepath.Join(basePath, "speed")); err == nil {
		speed := strings.TrimSpace(string(data))
		if speed != "" && speed != "-1" {
			info.Speed = speed + "Mb/s"
		}
	}

	// Statistics
	statsPath := filepath.Join(basePath, "statistics")
	info.RxBytes = readStatFile(filepath.Join(statsPath, "rx_bytes"))
	info.TxBytes = readStatFile(filepath.Join(statsPath, "tx_bytes"))
	info.RxPackets = readStatFile(filepath.Join(statsPath, "rx_packets"))
	info.TxPackets = readStatFile(filepath.Join(statsPath, "tx_packets"))
	info.RxErrors = readStatFile(filepath.Join(statsPath, "rx_errors"))
	info.TxErrors = readStatFile(filepath.Join(statsPath, "tx_errors"))

	// Temperature
	if t, err := temperature.GetNICTemp(iface); err == nil {
		info.Temp = t.Value
	}

	return info, nil
}

// GetAllNICs returns information for all monitored NICs.
func GetAllNICs() []NICInfo {
	var nics []NICInfo

	for _, iface := range GetMonitoredNICs() {
		if info, err := GetNICInfo(iface); err == nil {
			nics = append(nics, *info)
		}
	}

	return nics
}

func readStatFile(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return val
}
