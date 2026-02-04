package handler

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/murata-lab/pervigil/bot/internal/sysinfo"
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
	{
		Name:        "cpu",
		Description: "CPU使用率とロードアベレージを表示",
	},
	{
		Name:        "memory",
		Description: "メモリ使用状況を表示",
	},
	{
		Name:        "network",
		Description: "全NIC情報を表示",
	},
	{
		Name:        "disk",
		Description: "ディスク使用状況を表示",
	},
	{
		Name:        "info",
		Description: "ルーター全情報を表示",
	},
}

var Handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"nic":     handleNIC,
	"temp":    handleTemp,
	"status":  handleStatus,
	"cpu":     handleCPU,
	"memory":  handleMemory,
	"network": handleNetwork,
	"disk":    handleDisk,
	"info":    handleInfo,
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
	uptime := sysinfo.GetUptime()
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

func handleCPU(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info, err := sysinfo.GetCPUInfo()
	if err != nil {
		respond(s, i, fmt.Sprintf("CPU情報取得エラー: %v", err))
		return
	}

	var sb strings.Builder
	sb.WriteString("**CPU情報**\n```\n")
	sb.WriteString(fmt.Sprintf("使用率:         %.1f%%\n", info.Usage))
	sb.WriteString(fmt.Sprintf("ロードアベレージ: %.2f / %.2f / %.2f\n",
		info.LoadAvg[0], info.LoadAvg[1], info.LoadAvg[2]))
	sb.WriteString("```")

	respond(s, i, sb.String())
}

func handleMemory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info, err := sysinfo.GetMemoryInfo()
	if err != nil {
		respond(s, i, fmt.Sprintf("メモリ情報取得エラー: %v", err))
		return
	}

	status := usageStatus(info.UsagePercent, 70, 90)

	var sb strings.Builder
	sb.WriteString("**メモリ情報**\n```\n")
	sb.WriteString(fmt.Sprintf("合計:   %s\n", sysinfo.FormatBytes(info.Total)))
	sb.WriteString(fmt.Sprintf("使用:   %s\n", sysinfo.FormatBytes(info.Used)))
	sb.WriteString(fmt.Sprintf("空き:   %s\n", sysinfo.FormatBytes(info.Available)))
	sb.WriteString(fmt.Sprintf("使用率: %.1f%% %s\n", info.UsagePercent, status))
	sb.WriteString("```")

	respond(s, i, sb.String())
}

func handleNetwork(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			tStatus := tempStatus(nic.Temp, 70, 85)
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

func handleDisk(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info, err := sysinfo.GetDiskInfo("/")
	if err != nil {
		respond(s, i, fmt.Sprintf("ディスク情報取得エラー: %v", err))
		return
	}

	status := usageStatus(info.UsagePercent, 70, 90)

	var sb strings.Builder
	sb.WriteString("**ディスク情報** (/)\n```\n")
	sb.WriteString(fmt.Sprintf("合計:   %s\n", sysinfo.FormatBytes(info.Total)))
	sb.WriteString(fmt.Sprintf("使用:   %s\n", sysinfo.FormatBytes(info.Used)))
	sb.WriteString(fmt.Sprintf("空き:   %s\n", sysinfo.FormatBytes(info.Available)))
	sb.WriteString(fmt.Sprintf("使用率: %.1f%% %s\n", info.UsagePercent, status))
	sb.WriteString("```")

	respond(s, i, sb.String())
}

func handleInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info := sysinfo.GetAllRouterInfo()

	var sb strings.Builder
	sb.WriteString("**ルーター情報**\n```\n")
	sb.WriteString(fmt.Sprintf("ホスト名: %s\n", info.Hostname))
	sb.WriteString(fmt.Sprintf("稼働時間: %s\n", info.Uptime))
	sb.WriteString(fmt.Sprintf("Go版:     %s\n\n", runtime.Version()))

	// CPU
	if info.CPU != nil {
		sb.WriteString(fmt.Sprintf("CPU使用率: %.1f%%\n", info.CPU.Usage))
		sb.WriteString(fmt.Sprintf("Load Avg:  %.2f / %.2f / %.2f\n",
			info.CPU.LoadAvg[0], info.CPU.LoadAvg[1], info.CPU.LoadAvg[2]))
	}

	// CPU temps
	if len(info.CPUTemps) > 0 {
		maxTemp := 0.0
		for _, t := range info.CPUTemps {
			if t.Value > maxTemp {
				maxTemp = t.Value
			}
		}
		sb.WriteString(fmt.Sprintf("CPU最高温度: %.1f°C\n", maxTemp))
	}
	sb.WriteString("\n")

	// Memory
	if info.Memory != nil {
		status := usageStatus(info.Memory.UsagePercent, 70, 90)
		sb.WriteString(fmt.Sprintf("メモリ: %s / %s (%.1f%%) %s\n",
			sysinfo.FormatBytes(info.Memory.Used),
			sysinfo.FormatBytes(info.Memory.Total),
			info.Memory.UsagePercent, status))
	}

	// Disk
	if info.Disk != nil {
		status := usageStatus(info.Disk.UsagePercent, 70, 90)
		sb.WriteString(fmt.Sprintf("ディスク: %s / %s (%.1f%%) %s\n",
			sysinfo.FormatBytes(info.Disk.Used),
			sysinfo.FormatBytes(info.Disk.Total),
			info.Disk.UsagePercent, status))
	}
	sb.WriteString("\n")

	// NICs
	sb.WriteString("NIC:\n")
	for _, nic := range info.NICs {
		state := "up"
		if nic.State != "up" {
			state = "down"
		}
		tempStr := ""
		if nic.Temp > 0 {
			tempStr = fmt.Sprintf(" %.1f°C", nic.Temp)
		}
		sb.WriteString(fmt.Sprintf("  %s: %s%s\n", nic.Name, state, tempStr))
	}

	sb.WriteString("```")
	respond(s, i, sb.String())
}

func usageStatus(val, warn, crit float64) string {
	switch {
	case val >= crit:
		return "🔴"
	case val >= warn:
		return "🟡"
	default:
		return "🟢"
	}
}
