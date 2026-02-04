package sysinfo

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/murata-lab/pervigil/bot/internal/temp"
)

// RouterInfo contains comprehensive router system information.
type RouterInfo struct {
	Hostname string
	Uptime   string
	CPU      *CPUInfo
	Memory   *MemInfo
	NICs     []NICInfo
	Disk     *DiskInfo
	CPUTemps []temp.TempReading
}

// GetAllRouterInfo returns all router system information.
func GetAllRouterInfo() *RouterInfo {
	info := &RouterInfo{}

	// Hostname
	var err error
	info.Hostname, err = os.Hostname()
	if err != nil {
		log.Printf("[sysinfo] hostname: %v", err)
	}

	// Uptime
	info.Uptime = GetUptime()

	// CPU info
	info.CPU, err = GetCPUInfo()
	if err != nil {
		log.Printf("[sysinfo] cpu: %v", err)
	}

	// Memory info
	info.Memory, err = GetMemoryInfo()
	if err != nil {
		log.Printf("[sysinfo] memory: %v", err)
	}

	// NIC info
	info.NICs = GetAllNICs()

	// Disk info (root partition)
	info.Disk, err = GetDiskInfo("/")
	if err != nil {
		log.Printf("[sysinfo] disk: %v", err)
	}

	// CPU temps
	info.CPUTemps, err = temp.GetCPUTemps()
	if err != nil {
		log.Printf("[sysinfo] cpu temps: %v", err)
	}

	return info
}

// GetUptime returns the system uptime as a string.
func GetUptime() string {
	out, err := exec.Command("uptime", "-p").Output()
	if err != nil {
		// Fallback for systems without -p flag
		data, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return "unknown"
		}
		var secs float64
		fmt.Sscanf(string(data), "%f", &secs)
		d := time.Duration(secs) * time.Second
		return d.Round(time.Minute).String()
	}
	return strings.TrimSpace(strings.TrimPrefix(string(out), "up "))
}
