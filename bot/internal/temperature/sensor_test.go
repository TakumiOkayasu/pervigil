package temperature

import (
	"fmt"
	"testing"
)

// mapSensorDeps returns command outputs, file contents, and glob results from maps
type mapSensorDeps struct {
	cmdOutput  map[string]string
	cmdErr     map[string]error
	files      map[string]string
	globResult []string
}

func (d *mapSensorDeps) RunCommand(name string, args ...string) ([]byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if err, ok := d.cmdErr[key]; ok && err != nil {
		return nil, err
	}
	if out, ok := d.cmdOutput[key]; ok {
		return []byte(out), nil
	}
	return nil, fmt.Errorf("command not found: %s", key)
}

func (d *mapSensorDeps) ReadFile(path string) ([]byte, error) {
	if content, ok := d.files[path]; ok {
		return []byte(content), nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (d *mapSensorDeps) Glob(pattern string) ([]string, error) {
	return d.globResult, nil
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestGetCPUTemps_FromSensors(t *testing.T) {
	deps := &mapSensorDeps{
		cmdOutput: map[string]string{
			"sensors -u": `coretemp-isa-0000
Core 0:
  temp2_input: 45.000
Core 1:
  temp3_input: 47.000
`,
		},
	}

	temps, err := GetCPUTempsWith(deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(temps) != 2 {
		t.Fatalf("expected 2 temps, got %d", len(temps))
	}

	if temps[0].Value != 45.0 {
		t.Errorf("expected Core 0 temp 45.0, got %f", temps[0].Value)
	}
}

func TestGetNICTemp_FromEthtool(t *testing.T) {
	deps := &mapSensorDeps{
		cmdOutput: map[string]string{
			"ethtool -m eth1": `Module temperature : 55.5`,
		},
	}

	temp, err := GetNICTempWith("eth1", deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if temp.Value != 55.5 {
		t.Errorf("expected NIC temp 55.5, got %f", temp.Value)
	}
	if temp.Label != "eth1" {
		t.Errorf("expected label 'eth1', got '%s'", temp.Label)
	}
}

func TestGetNICTemp_DefaultInterface(t *testing.T) {
	deps := &mapSensorDeps{
		cmdOutput: map[string]string{
			"ethtool -m eth1": `Module temperature : 60.0`,
		},
	}

	temp, err := GetNICTempWith("", deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if temp.Label != "eth1" {
		t.Errorf("expected default interface 'eth1', got '%s'", temp.Label)
	}
}

// フォールバックテスト

func TestGetCPUTemps_FallbackToHwmon(t *testing.T) {
	// sensorsコマンド失敗→hwmonフォールバック
	deps := &mapSensorDeps{
		cmdErr: map[string]error{
			"sensors -u": &testError{msg: "sensors not found"},
		},
		globResult: []string{"/sys/class/hwmon/hwmon0/temp1_input"},
		files: map[string]string{
			"/sys/class/hwmon/hwmon0/name":        "coretemp",
			"/sys/class/hwmon/hwmon0/temp1_label": "Core 0",
			"/sys/class/hwmon/hwmon0/temp1_input": "52000", // 52°C in millidegrees
		},
	}

	temps, err := GetCPUTempsWith(deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(temps) != 1 {
		t.Fatalf("expected 1 temp from hwmon fallback, got %d", len(temps))
	}

	if temps[0].Value != 52.0 {
		t.Errorf("expected temp 52.0, got %f", temps[0].Value)
	}
}

func TestGetNICTemp_FallbackToEthtoolStats(t *testing.T) {
	// ethtool -m 失敗→ethtool -S フォールバック
	deps := &mapSensorDeps{
		cmdErr: map[string]error{
			"ethtool -m eth1": &testError{msg: "no EEPROM"},
		},
		cmdOutput: map[string]string{
			"ethtool -S eth1": `temp: 65.0`,
		},
	}

	temp, err := GetNICTempWith("eth1", deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if temp.Value != 65.0 {
		t.Errorf("expected NIC temp 65.0 from stats fallback, got %f", temp.Value)
	}
}

func TestGetNICTemp_FallbackToHwmon(t *testing.T) {
	// ethtool全失敗→hwmonフォールバック
	deps := &mapSensorDeps{
		cmdErr: map[string]error{
			"ethtool -m eth1": &testError{msg: "no EEPROM"},
			"ethtool -S eth1": &testError{msg: "no stats"},
		},
		globResult: []string{"/sys/class/net/eth1/device/hwmon/hwmon0/temp1_input"},
		files: map[string]string{
			"/sys/class/net/eth1/device/hwmon/hwmon0/temp1_input": "70000", // 70°C
		},
	}

	temp, err := GetNICTempWith("eth1", deps)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if temp.Value != 70.0 {
		t.Errorf("expected NIC temp 70.0 from hwmon fallback, got %f", temp.Value)
	}
}
