package sysinfo

import (
	"fmt"
	"testing"
	"time"
)

// mapFileCpuDeps returns file contents from a map
type mapFileCpuDeps struct {
	files map[string]string
}

func (d *mapFileCpuDeps) ReadFile(path string) ([]byte, error) {
	if content, ok := d.files[path]; ok {
		return []byte(content), nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (d *mapFileCpuDeps) Sleep(time.Duration) {}

// sequentialStatCpuDeps returns different /proc/stat values on each call
type sequentialStatCpuDeps struct {
	statValues []string
	loadavg    string
	callIndex  int
}

func (d *sequentialStatCpuDeps) ReadFile(path string) ([]byte, error) {
	if path == "/proc/loadavg" {
		return []byte(d.loadavg), nil
	}
	if path == "/proc/stat" {
		idx := d.callIndex
		if idx >= len(d.statValues) {
			idx = len(d.statValues) - 1
		}
		d.callIndex++
		return []byte(d.statValues[idx]), nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (d *sequentialStatCpuDeps) Sleep(time.Duration) {}

func TestGetCPUInfo_LoadAvg(t *testing.T) {
	deps := &mapFileCpuDeps{
		files: map[string]string{
			"/proc/loadavg": "1.50 2.00 2.50 1/100 12345",
			"/proc/stat":    "cpu  100 0 50 200 0 0 0 0 0 0\n",
		},
	}

	info, err := GetCPUInfoWith(deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if info.LoadAvg[0] != 1.50 {
		t.Errorf("expected LoadAvg[0]=1.50, got %f", info.LoadAvg[0])
	}
	if info.LoadAvg[1] != 2.00 {
		t.Errorf("expected LoadAvg[1]=2.00, got %f", info.LoadAvg[1])
	}
	if info.LoadAvg[2] != 2.50 {
		t.Errorf("expected LoadAvg[2]=2.50, got %f", info.LoadAvg[2])
	}
}

func TestGetCPUInfo_MissingLoadAvg(t *testing.T) {
	deps := &mapFileCpuDeps{
		files: map[string]string{
			"/proc/stat": "cpu  100 0 50 200 0 0 0 0 0 0\n",
		},
	}

	info, err := GetCPUInfoWith(deps)
	if err != nil {
		t.Fatalf("expected no error even without loadavg, got %v", err)
	}

	if info.LoadAvg[0] != 0 {
		t.Errorf("expected LoadAvg[0]=0, got %f", info.LoadAvg[0])
	}
}

func TestGetCPUInfo_Usage(t *testing.T) {
	deps := &sequentialStatCpuDeps{
		statValues: []string{
			"cpu  100 0 50 200 0 0 0 0 0 0\n", // total=350, idle=200
			"cpu  150 0 75 250 0 0 0 0 0 0\n", // total=475, idle=250
		},
		loadavg: "1.00 1.00 1.00 1/100 12345",
	}

	info, err := GetCPUInfoWith(deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Usage = 100 * (total_diff - idle_diff) / total_diff
	// total_diff = 475 - 350 = 125, idle_diff = 250 - 200 = 50
	// Usage = 100 * 75 / 125 = 60%
	expected := 60.0
	if info.Usage != expected {
		t.Errorf("expected Usage=%f, got %f", expected, info.Usage)
	}
}
