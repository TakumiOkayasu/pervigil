package sysinfo

import (
	"os"
	"strconv"
	"strings"
)

// MemInfo contains memory usage information.
type MemInfo struct {
	Total        uint64
	Used         uint64
	Available    uint64
	UsagePercent float64
}

// GetMemoryInfo returns memory usage information.
func GetMemoryInfo() (*MemInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	info := &MemInfo{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Values are in kB
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		val *= 1024 // Convert to bytes

		switch fields[0] {
		case "MemTotal:":
			info.Total = val
		case "MemAvailable:":
			info.Available = val
		}
	}

	if info.Total > 0 {
		info.Used = info.Total - info.Available
		info.UsagePercent = 100 * float64(info.Used) / float64(info.Total)
	}

	return info, nil
}

// FormatBytes formats bytes to human readable string.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatUint(b, 10) + " B"
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KiB", "MiB", "GiB", "TiB"}
	return strconv.FormatFloat(float64(b)/float64(div), 'f', 1, 64) + " " + units[exp]
}
