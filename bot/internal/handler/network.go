package handler

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/murata-lab/pervigil/bot/internal/sysinfo"
)

func cmdNetwork(s *discordgo.Session, i *discordgo.InteractionCreate) {
	nics := sysinfo.GetAllNICs()

	if len(nics) == 0 {
		respond(s, i, "NIC情報なし")
		return
	}

	var sb strings.Builder
	sb.WriteString("**ネットワーク情報**\n```\n")

	for _, nic := range nics {
		state := "🟢"
		if nic.State != "up" {
			state = "🔴"
		}

		sb.WriteString(fmt.Sprintf("%s %s (%s)\n", state, nic.Name, nic.State))
		if nic.Speed != "" {
			sb.WriteString(fmt.Sprintf("  速度: %s\n", nic.Speed))
		}
		if nic.Temp > 0 {
			tStatus := statusIndicator(nic.Temp, 70, 85)
			sb.WriteString(fmt.Sprintf("  温度: %.1f°C %s\n", nic.Temp, tStatus))
		}
		sb.WriteString(fmt.Sprintf("  RX: %s (%d pkts, %d err)\n",
			sysinfo.FormatBytes(nic.RxBytes), nic.RxPackets, nic.RxErrors))
		sb.WriteString(fmt.Sprintf("  TX: %s (%d pkts, %d err)\n",
			sysinfo.FormatBytes(nic.TxBytes), nic.TxPackets, nic.TxErrors))
		sb.WriteString("\n")
	}

	sb.WriteString("```")
	respond(s, i, sb.String())
}
