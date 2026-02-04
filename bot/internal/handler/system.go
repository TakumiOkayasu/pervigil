package handler

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/murata-lab/pervigil/bot/internal/sysinfo"
	"github.com/murata-lab/pervigil/bot/internal/temperature"
)

func maxCPUTemp(temps []temperature.TempReading) float64 {
	max := 0.0
	for _, t := range temps {
		if t.Value > max {
			max = t.Value
		}
	}
	return max
}

func cmdStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	hostname, _ := os.Hostname()
	uptime := sysinfo.GetUptime()
	iface := os.Getenv("NIC_INTERFACE")
	cpu, nic := temperature.GetAllTemps(iface)

	var sb strings.Builder
	sb.WriteString("**システム状態**\n```\n")
	sb.WriteString(fmt.Sprintf("ホスト名: %s\n", hostname))
	sb.WriteString(fmt.Sprintf("稼働時間: %s\n", uptime))
	sb.WriteString(fmt.Sprintf("Go版: %s\n", runtime.Version()))
	sb.WriteString("\n")

	if len(cpu) > 0 {
		sb.WriteString(fmt.Sprintf("CPU最高温度: %.1f°C\n", maxCPUTemp(cpu)))
	}

	if nic != nil {
		status := statusIndicator(nic.Value, 70, 85)
		sb.WriteString(fmt.Sprintf("NIC温度: %.1f°C %s\n", nic.Value, status))
	}

	sb.WriteString("```")
	respond(s, i, sb.String())
}

func cmdCPU(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

func cmdMemory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info, err := sysinfo.GetMemoryInfo()
	if err != nil {
		respond(s, i, fmt.Sprintf("メモリ情報取得エラー: %v", err))
		return
	}

	status := statusIndicator(info.UsagePercent, 70, 90)

	var sb strings.Builder
	sb.WriteString("**メモリ情報**\n```\n")
	sb.WriteString(fmt.Sprintf("合計:   %s\n", sysinfo.FormatBytes(info.Total)))
	sb.WriteString(fmt.Sprintf("使用:   %s\n", sysinfo.FormatBytes(info.Used)))
	sb.WriteString(fmt.Sprintf("空き:   %s\n", sysinfo.FormatBytes(info.Available)))
	sb.WriteString(fmt.Sprintf("使用率: %.1f%% %s\n", info.UsagePercent, status))
	sb.WriteString("```")

	respond(s, i, sb.String())
}

func cmdDisk(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info, err := sysinfo.GetDiskInfo("/")
	if err != nil {
		respond(s, i, fmt.Sprintf("ディスク情報取得エラー: %v", err))
		return
	}

	status := statusIndicator(info.UsagePercent, 70, 90)

	var sb strings.Builder
	sb.WriteString("**ディスク情報** (/)\n```\n")
	sb.WriteString(fmt.Sprintf("合計:   %s\n", sysinfo.FormatBytes(info.Total)))
	sb.WriteString(fmt.Sprintf("使用:   %s\n", sysinfo.FormatBytes(info.Used)))
	sb.WriteString(fmt.Sprintf("空き:   %s\n", sysinfo.FormatBytes(info.Available)))
	sb.WriteString(fmt.Sprintf("使用率: %.1f%% %s\n", info.UsagePercent, status))
	sb.WriteString("```")

	respond(s, i, sb.String())
}

func cmdInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	info := sysinfo.GetAllRouterInfo()

	var sb strings.Builder
	sb.WriteString("**ルーター情報**\n```\n")
	sb.WriteString(fmt.Sprintf("ホスト名: %s\n", info.Hostname))
	sb.WriteString(fmt.Sprintf("稼働時間: %s\n", info.Uptime))
	sb.WriteString(fmt.Sprintf("Go版:     %s\n\n", runtime.Version()))

	if info.CPU != nil {
		sb.WriteString(fmt.Sprintf("CPU使用率: %.1f%%\n", info.CPU.Usage))
		sb.WriteString(fmt.Sprintf("Load Avg:  %.2f / %.2f / %.2f\n",
			info.CPU.LoadAvg[0], info.CPU.LoadAvg[1], info.CPU.LoadAvg[2]))
	}

	if len(info.CPUTemps) > 0 {
		sb.WriteString(fmt.Sprintf("CPU最高温度: %.1f°C\n", maxCPUTemp(info.CPUTemps)))
	}
	sb.WriteString("\n")

	if info.Memory != nil {
		status := statusIndicator(info.Memory.UsagePercent, 70, 90)
		sb.WriteString(fmt.Sprintf("メモリ: %s / %s (%.1f%%) %s\n",
			sysinfo.FormatBytes(info.Memory.Used),
			sysinfo.FormatBytes(info.Memory.Total),
			info.Memory.UsagePercent, status))
	}

	if info.Disk != nil {
		status := statusIndicator(info.Disk.UsagePercent, 70, 90)
		sb.WriteString(fmt.Sprintf("ディスク: %s / %s (%.1f%%) %s\n",
			sysinfo.FormatBytes(info.Disk.Used),
			sysinfo.FormatBytes(info.Disk.Total),
			info.Disk.UsagePercent, status))
	}
	sb.WriteString("\n")

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
