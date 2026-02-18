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

const (
	// advertiseX540 is the bitmask for Intel X540-T2 (ixgbe) supported link modes:
	//
	//	0x0020 = 1000baseT/Full
	//	0x1000 = 10000baseT/Full
	advertiseX540 = "0x1020"

	// speedLimitMbps is the NIC speed limit in Mbps applied during thermal throttling.
	speedLimitMbps = "1000"
)

// EthtoolSpeedController controls NIC speed using ethtool
type EthtoolSpeedController struct {
	runner    commandRunner
	advertise string
}

// NewEthtoolSpeedController creates a new ethtool-based speed controller
func NewEthtoolSpeedController() *EthtoolSpeedController {
	return &EthtoolSpeedController{runner: &osCommandRunner{}, advertise: advertiseX540}
}

// NewEthtoolSpeedControllerWith creates a controller with custom runner (for testing)
func NewEthtoolSpeedControllerWith(runner commandRunner) *EthtoolSpeedController {
	return &EthtoolSpeedController{runner: runner, advertise: advertiseX540}
}

// Limit sets the NIC speed to 1Gbps
func (c *EthtoolSpeedController) Limit(iface string) error {
	return c.runner.Run("ethtool", "-s", iface, "speed", speedLimitMbps, "duplex", "full", "autoneg", "off")
}

// Restore enables auto-negotiation with advertised link modes for the NIC.
func (c *EthtoolSpeedController) Restore(iface string) error {
	return c.runner.Run("ethtool", "-s", iface, "autoneg", "on", "advertise", c.advertise)
}
