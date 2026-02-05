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

	iface := os.Getenv("NIC_INTERFACE")
	if iface == "" {
		iface = "eth1"
	}

	nic, err := temperature.GetNICTemp(iface)

	var content string
	if err != nil {
		content = fmt.Sprintf("NIC温度取得エラー: %v", err)
	} else {
		status := statusIndicator(nic.Value, 70, 85)
		content = fmt.Sprintf("**NIC温度** (%s)\n%s %.1f°C", nic.Label, status, nic.Value)
	}

	followup(s, i, content)
}

func cmdTemp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := deferredRespond(s, i); err != nil {
		return
	}

	iface := os.Getenv("NIC_INTERFACE")
	cpu, nic := temperature.GetAllTemps(iface)

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
	if nic != nil {
		status := statusIndicator(nic.Value, 70, 85)
		sb.WriteString(fmt.Sprintf("  %-10s: %5.1f°C %s\n", nic.Label, nic.Value, status))
	} else {
		sb.WriteString("  取得不可\n")
	}

	sb.WriteString("```")
	followup(s, i, sb.String())
}
