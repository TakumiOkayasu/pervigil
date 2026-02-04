package monitor

import "github.com/murata-lab/pervigil/bot/internal/temperature"

// TempAdapter adapts the temperature package for monitor use
type TempAdapter struct{}

// NewTempAdapter creates a new temperature adapter
func NewTempAdapter() *TempAdapter {
	return &TempAdapter{}
}

// GetNICTemp returns the NIC temperature
func (a *TempAdapter) GetNICTemp(iface string) (*temperature.TempReading, error) {
	return temperature.GetNICTemp(iface)
}
