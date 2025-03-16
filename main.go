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
	}

	if err := discord.StartBot(cfg); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
