package discord

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/loader"
)

func StartBot(cfg *config.Config) error {
	session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		return err
	}

	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		return err
	}

	defaultPrompt := modelCfg.Prompts["default"]

	chatService, err := chat.NewChat(cfg.GeminiAPIKey, modelCfg.ModelName, defaultPrompt)
	if err != nil {
		return err
	}
	defer chatService.Close()

	setupHandlers(session, chatService, modelCfg)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds

	if err := session.Open(); err != nil {
		return err
	}

	log.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	session.Close()
	return nil
}
