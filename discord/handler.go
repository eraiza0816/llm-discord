package discord

import (
	"fmt"
	"log"
	"os"
	"time"

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
	s.AddHandler(interactionCreate(chatSvc, modelCfg))
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func interactionCreate(chatSvc chat.Service, modelCfg *loader.ModelConfig) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

			userPrompt := modelCfg.GetPromptByUser(username)

			logMessage := fmt.Sprintf("User %s sent message: %s at %s", username, message, timestamp)
			log.Printf(logMessage)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "ちょっと待ってね！",
				},
			})

			response, _, err := chatSvc.GetResponse(userID, username, message, timestamp, userPrompt)
			if err != nil {
				log.Printf("GetResponse error: %v", err)
				content := fmt.Sprintf("エラーが発生しました: %v", err)
				_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &content,
				})
				if err != nil {
					log.Printf("InteractionResponseEdit error: %v", err)
				}
				return
			}

			content := response
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}
		case "reset":
			userID := i.Member.User.ID

			resetUsername := i.Member.User.Username + "#" + i.Member.User.Discriminator
			logMessage := fmt.Sprintf("User %s performed a reset operation.", resetUsername)
			log.Printf(logMessage)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "履歴をリセットしています...",
				},
			})

			chatSvc.ClearHistory(userID)

			content := "チャット履歴をリセットしました！"
			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}

			log.Printf("User %s reset the chat history.", resetUsername)
		}
	}
}
