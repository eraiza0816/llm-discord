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
		}
	})
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func chatCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, _ *loader.ModelConfig) {
	username := i.Member.User.Username
	message := i.ApplicationCommandData().Options[0].StringValue()
	timestamp := time.Now().Format(time.RFC3339)

	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		log.Printf("LoadModelConfig error: %v", err)
		content := fmt.Sprintf("エラーが発生しました（設定ファイル読み込み失敗）: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
			},
		})
		return
	}

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

	embedUser := &discordgo.MessageEmbed{
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

	embedBot := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    modelCfg.Name,
			IconURL: modelCfg.Icon,
		},
		Fields: splitToEmbedFields(response),
		Color:  0xa8ffee,
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embedUser, embedBot},
	})
	if err != nil {
		log.Printf("InteractionResponseEdit error: %v", err)
	}
}

func resetCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service) {
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
}

func aboutCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, modelCfg *loader.ModelConfig) {
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
