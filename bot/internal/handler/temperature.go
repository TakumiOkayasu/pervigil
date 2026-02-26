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
	_, nics, board := temperature.GetAllTemps(ifaces)

	var sb strings.Builder
	sb.WriteString("**NIC温度**\n```\n")

	if len(nics) == 0 {
		sb.WriteString("N/A (センサー未対応)\n")
		if len(board) > 0 {
			sb.WriteString("\nBoard:\n")
			for _, b := range board {
				sb.WriteString(fmt.Sprintf("  %-16s: %5.1f°C\n", b.Label, b.Value))
			}
		}
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
	cpu, nics, board := temperature.GetAllTemps(ifaces)

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
		sb.WriteString("  N/A (センサー未対応)\n")
	}

	if len(board) > 0 {
		sb.WriteString("\nBoard:\n")
		for _, b := range board {
			sb.WriteString(fmt.Sprintf("  %-16s: %5.1f°C\n", b.Label, b.Value))
		}
	}

	sb.WriteString("```")
	followup(s, i, sb.String())
}
