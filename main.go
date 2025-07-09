package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/discord"
	"github.com/eraiza0816/llm-discord/history"
)

func main() {
	if err := history.InitAuditLog(); err != nil {
		log.Fatalf("Failed to initialize log: %v", err)
	}
	defer history.CloseAuditLog()

	downloadDir := "data/downloads/images"
	history.StartAuditLogMonitor(downloadDir)
	defer history.StopAuditLogMonitor()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := discord.StartBot(cfg); err != nil {
		log.Printf("Bot error: %v", err)
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Bot shutting down...")
}
