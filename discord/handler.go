package discord

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/loader"
)

func setupHandlers(s *discordgo.Session, chatSvc chat.Service, modelCfg *loader.ModelConfig) {
	logFile, err := os.OpenFile("log/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)

	s.AddHandler(onReady)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "chat":
			chatCommandHandler(s, i, chatSvc, modelCfg)
		case "reset":
			resetCommandHandler(s, i, chatSvc)
		case "about":
			aboutCommandHandler(s, i, modelCfg)
		case "edit":
			editCommandHandler(s, i, chatSvc)
		}
	})
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}
