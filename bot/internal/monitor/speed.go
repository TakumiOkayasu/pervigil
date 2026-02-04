package monitor

import (
	"os/exec"
)

// commandRunner abstracts command execution
type commandRunner interface {
	Run(name string, args ...string) error
}

// osCommandRunner is the production implementation
type osCommandRunner struct{}

func (r *osCommandRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// EthtoolSpeedController controls NIC speed using ethtool
type EthtoolSpeedController struct {
	runner commandRunner
}

// NewEthtoolSpeedController creates a new ethtool-based speed controller
func NewEthtoolSpeedController() *EthtoolSpeedController {
	return &EthtoolSpeedController{runner: &osCommandRunner{}}
}

// NewEthtoolSpeedControllerWith creates a controller with custom runner (for testing)
func NewEthtoolSpeedControllerWith(runner commandRunner) *EthtoolSpeedController {
	return &EthtoolSpeedController{runner: runner}
}

// Limit sets the NIC speed to 1Gbps
func (c *EthtoolSpeedController) Limit(iface string) error {
	return c.runner.Run("ethtool", "-s", iface, "speed", "1000", "duplex", "full", "autoneg", "off")
}

// Restore enables auto-negotiation
func (c *EthtoolSpeedController) Restore(iface string) error {
	return c.runner.Run("ethtool", "-s", iface, "autoneg", "on")
}
