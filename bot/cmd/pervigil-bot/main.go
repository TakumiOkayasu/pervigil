package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/murata-lab/pervigil/bot/internal/config"
	"github.com/murata-lab/pervigil/bot/internal/handler"
)

func main() {
	// Load .env from current dir, then from executable dir
	_ = godotenv.Load()
	if exe, err := os.Executable(); err == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		log.Fatalf("discord session error: %v", err)
	}

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if h, ok := handler.Handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	if err := dg.Open(); err != nil {
		log.Fatalf("open connection error: %v", err)
	}
	defer dg.Close()

	// Register commands
	registeredCmds := make([]*discordgo.ApplicationCommand, len(handler.Commands))
	for i, cmd := range handler.Commands {
		registered, err := dg.ApplicationCommandCreate(dg.State.User.ID, cfg.GuildID, cmd)
		if err != nil {
			log.Printf("command register error (%s): %v", cmd.Name, err)
			continue
		}
		registeredCmds[i] = registered
		log.Printf("Registered command: %s", cmd.Name)
	}

	log.Println("Bot is running. Press Ctrl+C to exit.")

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")

	// Cleanup commands on shutdown (optional, for dev)
	if os.Getenv("CLEANUP_COMMANDS") == "1" {
		for _, cmd := range registeredCmds {
			if cmd != nil {
				dg.ApplicationCommandDelete(dg.State.User.ID, cfg.GuildID, cmd.ID)
			}
		}
	}
}
