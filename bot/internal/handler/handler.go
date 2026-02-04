package handler

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// Command represents a Discord slash command with its handler.
type Command struct {
	Name        string
	Description string
	Execute     func(*discordgo.Session, *discordgo.InteractionCreate)
}

var commands []Command

func init() {
	commands = []Command{
		// temperature.go
		{"nic", "NIC温度を表示", cmdNIC},
		{"temp", "全温度情報を表示 (CPU + NIC)", cmdTemp},
		// system.go
		{"status", "システム状態サマリー", cmdStatus},
		{"cpu", "CPU使用率とロードアベレージを表示", cmdCPU},
		{"memory", "メモリ使用状況を表示", cmdMemory},
		{"disk", "ディスク使用状況を表示", cmdDisk},
		{"info", "ルーター全情報を表示", cmdInfo},
		// network.go
		{"network", "全NIC情報を表示", cmdNetwork},
	}
}

// Commands returns Discord application commands for registration.
func Commands() []*discordgo.ApplicationCommand {
	result := make([]*discordgo.ApplicationCommand, len(commands))
	for i, cmd := range commands {
		result[i] = &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
		}
	}
	return result
}

// Handlers returns a map of command handlers.
func Handlers() map[string]func(*discordgo.Session, *discordgo.InteractionCreate) {
	result := make(map[string]func(*discordgo.Session, *discordgo.InteractionCreate))
	for _, cmd := range commands {
		result[cmd.Name] = cmd.Execute
	}
	return result
}

// respond sends a response to a Discord interaction.
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	if err != nil {
		log.Printf("[handler] respond error: %v", err)
	}
}

// statusIndicator returns an emoji based on value thresholds.
func statusIndicator(val, warn, crit float64) string {
	switch {
	case val >= crit:
		return "🔴"
	case val >= warn:
		return "🟡"
	default:
		return "🟢"
	}
}
