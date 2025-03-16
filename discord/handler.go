package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
)

func setupHandlers(s *discordgo.Session, chatSvc chat.Service) {
	s.AddHandler(onReady)
	s.AddHandler(interactionCreate(chatSvc))
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func interactionCreate(chatSvc chat.Service) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "chat":
			userID := i.Member.User.ID
			username := i.Member.User.Username
			message := i.ApplicationCommandData().Options[0].StringValue()
			timestamp := time.Now().Format(time.RFC3339)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ちょっと待ってね！",
				},
			})

			resp, elapsed, err := chatSvc.GetResponse(userID, username, message, timestamp)
			if err != nil {
				log.Printf("GetResponse error: %v", err)
				resp = "エラーが発生しました。"
			}

			content := fmt.Sprintf("%s\n%.2fms", resp, elapsed)
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}
		}
	}
}
