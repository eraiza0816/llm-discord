package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/loader"
)

type CustomModelConfig struct {
	Prompts map[string]string `json:"prompts"`
}

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
		case "edit":
			editCommandHandler(s, i)
		}
	})
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func chatCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, modelCfg *loader.ModelConfig) {
	username := i.Member.User.Username
	message := i.ApplicationCommandData().Options[0].StringValue()
	timestamp := time.Now().Format(time.RFC3339)
	userID := i.Member.User.ID

	var userPrompt string
	if _, err := os.Stat("json/custom_model.json"); err == nil {
		file, err := os.ReadFile("json/custom_model.json")
		if err != nil {
			log.Printf("LoadModelConfig error: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("エラーが発生しました（設定ファイル読み込み失敗）: %v", err),
				},
			})
			return
		}

		customModelCfg := CustomModelConfig{}
		if err := json.Unmarshal(file, &customModelCfg); err == nil {
			if prompt, exists := customModelCfg.Prompts[username]; exists {
				userPrompt = prompt
			} else {
				log.Printf("Prompt not found for user %s", username)
				modelCfg, err := loader.LoadModelConfig("json/model.json")
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("エラーが発生しました（設定ファイル読み込み失敗）: %v", err),
						},
					})
					return
				}
				userPrompt = modelCfg.GetPromptByUser(username)
			}
		} else {
			log.Printf("json.Unmarshal error: %v", err)
		}
	} else {
		modelCfg, err := loader.LoadModelConfig("json/model.json")
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("エラーが発生しました（設定ファイル読み込み失敗）: %v", err),
				},
			})
			return
		}
		userPrompt = modelCfg.GetPromptByUser(username)
	}

	log.Printf("User %s sent message: %s ", username, message)

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

	embedUser := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    username,
			IconURL: i.Member.User.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{Value: message},
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
	log.Printf("User %s performed a reset operation.", resetUsername)

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
}

func aboutCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, modelCfg *loader.ModelConfig) {
	username := i.Member.User.Username
	log.Printf("User %s performed an about operation.", username)

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
}

func editCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("editCommandHandler called")
	username := i.Member.User.Username
	prompt := i.ApplicationCommandData().Options[0].StringValue()

	var customModelCfg CustomModelConfig
	if data, err := os.ReadFile("json/custom_model.json"); err == nil {
		json.Unmarshal(data, &customModelCfg)
	} else {
		customModelCfg = CustomModelConfig{Prompts: make(map[string]string)}
	}

	customModelCfg.Prompts[username] = prompt

	jsonBytes, err := json.MarshalIndent(customModelCfg, "", "  ")
	if err != nil {
		log.Printf("json.Marshal error: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("エラーが発生しました: %v", err),
			},
		})
		return
	}

	err = os.WriteFile("json/custom_model.json", jsonBytes, 0644)
	if err != nil {
		log.Printf("os.WriteFile error: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("エラーが発生しました: %v", err),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "プロンプトテンプレートを更新しました！",
		},
	})
}