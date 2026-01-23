package handler

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/murata-lab/pervigil/bot/internal/temp"
)

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "nic",
		Description: "NIC温度を表示",
	},
	{
		Name:        "temp",
		Description: "全温度情報を表示 (CPU + NIC)",
	},
	{
		Name:        "status",
		Description: "システム状態サマリー",
	},
}

var Handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"nic":    handleNIC,
	"temp":   handleTemp,
	"status": handleStatus,
}

func handleNIC(s *discordgo.Session, i *discordgo.InteractionCreate) {
	iface := os.Getenv("NIC_INTERFACE")
	if iface == "" {
		iface = "eth1"
	}

	nic, err := temp.GetNICTemp(iface)

	var content string
	if err != nil {
		content = fmt.Sprintf("NIC温度取得エラー: %v", err)
	} else {
		status := tempStatus(nic.Value, 70, 85)
		content = fmt.Sprintf("**NIC温度** (%s)\n%s %.1f°C", nic.Label, status, nic.Value)
	}

	respond(s, i, content)
}

func handleTemp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	iface := os.Getenv("NIC_INTERFACE")
	cpu, nic := temp.GetAllTemps(iface)

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
		status := tempStatus(nic.Value, 70, 85)
		sb.WriteString(fmt.Sprintf("  %-10s: %5.1f°C %s\n", nic.Label, nic.Value, status))
	} else {
		sb.WriteString("  取得不可\n")
	}

	sb.WriteString("```")
	respond(s, i, sb.String())
}

func handleStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	hostname, _ := os.Hostname()
	uptime := getUptime()
	iface := os.Getenv("NIC_INTERFACE")
	cpu, nic := temp.GetAllTemps(iface)

	var sb strings.Builder
	sb.WriteString("**システム状態**\n```\n")
	sb.WriteString(fmt.Sprintf("ホスト名: %s\n", hostname))
	sb.WriteString(fmt.Sprintf("稼働時間: %s\n", uptime))
	sb.WriteString(fmt.Sprintf("Go版: %s\n", runtime.Version()))
	sb.WriteString("\n")

	// CPU max temp
	if len(cpu) > 0 {
		maxTemp := 0.0
		for _, t := range cpu {
			if t.Value > maxTemp {
				maxTemp = t.Value
			}
		}
		sb.WriteString(fmt.Sprintf("CPU最高温度: %.1f°C\n", maxTemp))
	}

	// NIC temp
	if nic != nil {
		status := tempStatus(nic.Value, 70, 85)
		sb.WriteString(fmt.Sprintf("NIC温度: %.1f°C %s\n", nic.Value, status))
	}

	sb.WriteString("```")
	respond(s, i, sb.String())
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func tempStatus(val, warn, crit float64) string {
	switch {
	case val >= crit:
		return "🔴"
	case val >= warn:
		return "🟡"
	default:
		return "🟢"
	}
}

func getUptime() string {
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
