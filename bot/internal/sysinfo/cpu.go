package sysinfo

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// fileReadable abstracts file reading (defined at usage site per Go idiom)
type fileReadable interface {
	ReadFile(path string) ([]byte, error)
}

// sleeper abstracts time.Sleep for testing
type sleeper interface {
	Sleep(d time.Duration)
}

// cpuDeps combines interfaces for CPU info operations
type cpuDeps interface {
	fileReadable
	sleeper
}

// osCpuDeps is the production implementation
type osCpuDeps struct{}

func (o *osCpuDeps) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (o *osCpuDeps) Sleep(d time.Duration) {
	time.Sleep(d)
}

// CPUInfo contains CPU usage and load information.
type CPUInfo struct {
	Usage   float64    // Overall CPU usage (%)
	LoadAvg [3]float64 // 1/5/15 min load averages
}

// GetCPUInfo returns CPU usage and load averages
func GetCPUInfo() (*CPUInfo, error) {
	return GetCPUInfoWith(&osCpuDeps{})
}

// GetCPUInfoWith returns CPU usage using the provided cpuDeps (for testing)
func GetCPUInfoWith(d cpuDeps) (*CPUInfo, error) {
	info := &CPUInfo{}

	// Get load averages
	data, err := d.ReadFile("/proc/loadavg")
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			info.LoadAvg[0], _ = strconv.ParseFloat(fields[0], 64)
			info.LoadAvg[1], _ = strconv.ParseFloat(fields[1], 64)
			info.LoadAvg[2], _ = strconv.ParseFloat(fields[2], 64)
		}
	}

	// Get CPU usage (compare two /proc/stat snapshots)
	usage, err := getCPUUsageWith(d)
	if err == nil {
		info.Usage = usage
	}

	return info, nil
}

func getCPUUsageWith(d cpuDeps) (float64, error) {
	stat1, err := readCPUStatWith(d)
	if err != nil {
		return 0, err
	}

	d.Sleep(100 * time.Millisecond)

	stat2, err := readCPUStatWith(d)
	if err != nil {
		return 0, err
	}

	idle := stat2.idle - stat1.idle
	total := stat2.total - stat1.total

	if total == 0 {
		return 0, nil
	}

	return 100 * float64(total-idle) / float64(total), nil
}

type cpuStat struct {
	idle  uint64
	total uint64
}

func readCPUStatWith(r fileReadable) (*cpuStat, error) {
	data, err := r.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}

			var total uint64
			var idle uint64

			for i, f := range fields[1:] {
				val, _ := strconv.ParseUint(f, 10, 64)
				total += val
				if i == 3 { // idle is 4th field (index 3)
					idle = val
				}
			}

			return &cpuStat{idle: idle, total: total}, nil
		}
	}

	return &cpuStat{}, nil
}
