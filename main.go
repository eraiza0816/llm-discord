package main

import (
	"log"
	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/discord"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
		return
	}

	if cfg == nil {
		log.Fatalf("Config is nil")
		return
	}

	if err := discord.StartBot(cfg); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
