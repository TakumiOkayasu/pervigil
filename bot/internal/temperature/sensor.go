package temperature

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Single-responsibility interfaces (ISP)

// commandRunnable abstracts command execution
type commandRunnable interface {
	RunCommand(name string, args ...string) ([]byte, error)
}

// fileReadable abstracts file reading
type fileReadable interface {
	ReadFile(path string) ([]byte, error)
}

// globbable abstracts glob pattern matching
type globbable interface {
	Glob(pattern string) ([]string, error)
}

// sensorDeps combines interfaces needed for sensor operations
type sensorDeps interface {
	commandRunnable
	fileReadable
	globbable
}

// osDeps is the production implementation
type osDeps struct{}

func (o *osDeps) RunCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (o *osDeps) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (o *osDeps) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

type TempReading struct {
	Label string
	Value float64
}

// GetCPUTemps returns CPU core temperatures
func GetCPUTemps() ([]TempReading, error) {
	return GetCPUTempsWith(&osDeps{})
}

// GetCPUTempsWith returns CPU core temperatures using provided deps (for testing)
func GetCPUTempsWith(d sensorDeps) ([]TempReading, error) {
	// Try lm-sensors first
	if temps, err := getCPUFromSensors(d); err == nil && len(temps) > 0 {
		return temps, nil
	}

	// Fallback to hwmon
	return getCPUFromHwmon(d)
}

// GetNICTemp returns NIC temperature
func GetNICTemp(iface string) (*TempReading, error) {
	return GetNICTempWith(iface, &osDeps{})
}

// GetNICTempWith returns NIC temperature using provided deps (for testing)
func GetNICTempWith(iface string, d sensorDeps) (*TempReading, error) {
	if iface == "" {
		iface = "eth1"
	}

	// Try ethtool first (most reliable for Intel NICs)
	if temp, err := getNICFromEthtool(iface, d); err == nil {
		return temp, nil
	}

	// Fallback to hwmon
	return getNICFromHwmon(iface, d)
}

func getCPUFromSensors(d commandRunnable) ([]TempReading, error) {
	out, err := d.RunCommand("sensors", "-u")
	if err != nil {
		return nil, err
	}

	var temps []TempReading
	re := regexp.MustCompile(`(Core \d+|Tctl|CPU).*\n\s+temp\d+_input:\s+([\d.]+)`)
	matches := re.FindAllStringSubmatch(string(out), -1)

	for _, m := range matches {
		val, _ := strconv.ParseFloat(m[2], 64)
		temps = append(temps, TempReading{Label: m[1], Value: val})
	}

	return temps, nil
}

// hwmonDeps is the minimal interface for hwmon operations
type hwmonDeps interface {
	fileReadable
	globbable
}

func getCPUFromHwmon(d hwmonDeps) ([]TempReading, error) {
	var temps []TempReading

	hwmonDirs, err := d.Glob("/sys/class/hwmon/hwmon*/temp*_input")
	if err != nil {
		return nil, err
	}

	for _, path := range hwmonDirs {
		dir := filepath.Dir(path)
		name, _ := d.ReadFile(filepath.Join(dir, "name"))
		nameStr := strings.TrimSpace(string(name))

		if nameStr != "coretemp" && nameStr != "k10temp" {
			continue
		}

		labelPath := strings.Replace(path, "_input", "_label", 1)
		label, _ := d.ReadFile(labelPath)
		labelStr := strings.TrimSpace(string(label))
		if labelStr == "" {
			labelStr = nameStr
		}

		data, err := d.ReadFile(path)
		if err != nil {
			continue
		}

		val, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		temps = append(temps, TempReading{Label: labelStr, Value: val / 1000})
	}

	return temps, nil
}

// ethtoolDeps is the minimal interface for ethtool operations
type ethtoolDeps interface {
	commandRunnable
	fileReadable
	globbable
}

func getNICFromEthtool(iface string, d ethtoolDeps) (*TempReading, error) {
	out, err := d.RunCommand("ethtool", "-m", iface)
	if err != nil {
		// Try alternative: ethtool -S for ixgbe driver
		return getNICFromEthtoolStats(iface, d)
	}

	re := regexp.MustCompile(`Module temperature\s*:\s*([\d.]+)`)
	match := re.FindStringSubmatch(string(out))
	if match == nil {
		return getNICFromEthtoolStats(iface, d)
	}

	val, _ := strconv.ParseFloat(match[1], 64)
	return &TempReading{Label: iface, Value: val}, nil
}

func getNICFromEthtoolStats(iface string, d commandRunnable) (*TempReading, error) {
	out, err := d.RunCommand("ethtool", "-S", iface)
	if err != nil {
		return nil, err
	}

	// ixgbe driver reports temperature in stats
	re := regexp.MustCompile(`temp:\s*([\d.]+)`)
	match := re.FindStringSubmatch(string(out))
	if match == nil || len(match) < 2 {
		return nil, fmt.Errorf("temperature not found in ethtool stats")
	}

	val, _ := strconv.ParseFloat(match[1], 64)
	return &TempReading{Label: iface, Value: val}, nil
}

func getNICFromHwmon(iface string, d hwmonDeps) (*TempReading, error) {
	// Look for network device hwmon
	hwmonDirs, err := d.Glob("/sys/class/net/" + iface + "/device/hwmon/hwmon*/temp*_input")
	if err != nil {
		return nil, err
	}
	if len(hwmonDirs) == 0 {
		return nil, fmt.Errorf("no hwmon for %s", iface)
	}

	data, err := d.ReadFile(hwmonDirs[0])
	if err != nil {
		return nil, err
	}

	val, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	return &TempReading{Label: iface, Value: val / 1000}, nil
}

// GetAllTemps returns all available temperature readings
func GetAllTemps(nicIface string) (cpu []TempReading, nic *TempReading) {
	cpu, _ = GetCPUTemps()
	nic, _ = GetNICTemp(nicIface)
	return
}
