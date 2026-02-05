package handler

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/murata-lab/pervigil/bot/internal/temperature"
)

func cmdNIC(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := deferredRespond(s, i); err != nil {
		return
	}

	ifaces := os.Getenv("NIC_INTERFACE")
	_, nics := temperature.GetAllTemps(ifaces)

	var sb strings.Builder
	sb.WriteString("**NIC温度**\n```\n")

	if len(nics) == 0 {
		sb.WriteString("取得不可\n")
	} else {
		for _, nic := range nics {
			status := statusIndicator(nic.Value, 70, 85)
			sb.WriteString(fmt.Sprintf("%-10s: %5.1f°C %s\n", nic.Label, nic.Value, status))
		}
	}

	sb.WriteString("```")
	followup(s, i, sb.String())
}

func cmdTemp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := deferredRespond(s, i); err != nil {
		return
	}

	ifaces := os.Getenv("NIC_INTERFACE")
	cpu, nics := temperature.GetAllTemps(ifaces)

	var sb strings.Builder
	sb.WriteString("**温度情報**\n```\n")

	if len(cpu) > 0 {
		sb.WriteString("CPU:\n")
		for _, t := range cpu {
			sb.WriteString(fmt.Sprintf("  %-10s: %5.1f°C\n", t.Label, t.Value))
		}
	} else {
		sb.WriteString("CPU: 取得不可\n")
	}

	sb.WriteString("\nNIC:\n")
	if len(nics) > 0 {
		for _, nic := range nics {
			status := statusIndicator(nic.Value, 70, 85)
			sb.WriteString(fmt.Sprintf("  %-10s: %5.1f°C %s\n", nic.Label, nic.Value, status))
		}
	} else {
		sb.WriteString("  取得不可\n")
	}

	sb.WriteString("```")
	followup(s, i, sb.String())
}
