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

func splitToEmbedFields(text string) []*discordgo.MessageEmbedField {
	const maxFieldLength = 1024
	var fields []*discordgo.MessageEmbedField

	for i := 0; i < len(text); i += maxFieldLength {
		end := i + maxFieldLength
		if end > len(text) {
			end = len(text)
		}
		chunk := text[i:end]
		field := &discordgo.MessageEmbedField{
			Name:   "",
			Value:  chunk,
			Inline: false,
		}
		fields = append(fields, field)
	}
	return fields
}

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
			interactionHandler(s, i, chatSvc, modelCfg)
		case "reset":
			interactionHandler(s, i, chatSvc, modelCfg)
		case "about":
			interactionHandler(s, i, chatSvc, modelCfg)
		}
	}
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, modelCfg *loader.ModelConfig) {
	switch i.ApplicationCommandData().Name {
	case "chat":
		username := i.Member.User.Username
		message := i.ApplicationCommandData().Options[0].StringValue()
		timestamp := time.Now().Format(time.RFC3339)

		userPrompt := modelCfg.GetPromptByUser(username)

		logMessage := fmt.Sprintf("User %s sent message: %s ", username, message)
		log.Printf(logMessage)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ちょっと待ってね！",
			},
		})

		userID := i.Member.User.ID
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

		embed_user := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    username,
				IconURL: i.Member.User.AvatarURL(""),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Value: message,
				},
			},
			Color: 0xfff9b7,
		}

		modelName := modelCfg.Name
		modelIconURL := modelCfg.Icon

		embed_bot := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    modelName,
				IconURL: modelIconURL,
			},
			Fields: splitToEmbedFields(response),
			Color:  0xa8ffee,
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed_user, embed_bot},
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

		embed := &discordgo.MessageEmbed{
			Description: "チャット履歴をリセットしました！",
			Color:       0xa8ffee,
		}
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			log.Printf("InteractionResponseEdit error: %v", err)
		}

		log.Printf("User %s reset the chat history.", resetUsername)
	case "about":
		username := i.Member.User.Username

		logMessage := fmt.Sprintf("User %s performed an about operation.", username)
		log.Printf(logMessage)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "少々お待ちください...",
			},
		})

		modelCfg, err := loader.LoadModelConfig("json/model.json")
		if err != nil {
			log.Printf("LoadModelConfig error: %v", err)
			content := fmt.Sprintf("エラーが発生しました: %v", err)
			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       modelCfg.About.Title,
			Description: modelCfg.About.Description,
			URL:         modelCfg.About.URL,
			Color:       0xa8ffee,
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			log.Printf("InteractionResponseEdit error: %v", err)
		}

		log.Printf("User %s performed an about operation.", username)
	}
}
