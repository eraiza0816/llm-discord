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
	log.SetOutput(os.Stdout)
	log.Println("StartBot called")
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
		{
			Name:        "about",
			Description: "このBotについて",
		},
		{
			Name:        "edit",
			Description: "プロンプトテンプレートをあなたの好みに編集",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "set_prompt",
					Description: "プロンプトテンプレートを入力",
					Required:    true,
				},
			},
		},
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))

	err = session.Open()
	if err != nil {
		log.Println("Error opening Discord session: ", err)
		return err
	}
	defer session.Close()

	log.Printf("session.State: %v", session.State)
	log.Printf("session.State.User: %v", session.State.User)
	if session.State == nil || session.State.User == nil {
		log.Println("session.State or session.State.User is nil")
		return errors.New("session.State or session.State.User is nil")
	}
	if session.State != nil && session.State.User != nil {
		for i, v := range commands {
			cmd, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
			if err != nil {
				log.Printf("Can not create '%v' command: %v", v.Name, err)
				continue
			}
			registeredCommands[i] = cmd
		}
		for _, v := range registeredCommands {
			if v != nil {
				log.Printf("Successfully created '%v' command.", v.Name)
			} else {
				log.Printf("Failed to register one of the commands (was nil)")
			}
		}
		log.Printf("Registered commands: %v", registeredCommands)
	} else {
		log.Println("session.State or session.State.User is nil, skipping command registration")
	}

	setupHandlers(session, chatService, modelCfg)

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("session.AddHandler called")
		fmt.Printf("Bot is ready! %s#%s\n", r.User.Username, r.User.Discriminator)
	})

	log.Println("Bot is running.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return nil
}
