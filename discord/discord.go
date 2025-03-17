package discord

import (
	"errors"
	"fmt"
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
	if cfg == nil {
		return errors.New("Config is nil")
	}

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

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "chat",
			Description: "おしゃべりしようよ",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "メッセージ",
					Required:    true,
				},
			},
		},
		{
			Name:        "reset",
			Description: "あなたとのチャット履歴をリセット",
		},
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	if err := session.Open(); err != nil {
		log.Println("Error opening Discord session: ", err)
		return err
	}
	defer session.Close()

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		if s.State == nil || s.State.User == nil {
			log.Println("s.State or s.State.User is nil")
			return
		}
		for i, v := range commands {
			cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
			if err != nil {
				log.Printf("Can not create '%v' command: %v", v.Name, err)
				continue
			}
			registeredCommands[i] = cmd
		}
		for _, v := range registeredCommands {
			log.Printf("Successfully created '%v' command.", v.Name)
		}
		fmt.Printf("Bot is ready! %s#%s\n", s.State.User.Username, s.State.User.Discriminator)
	})

	log.Println("Bot is running.")
	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return nil
}
