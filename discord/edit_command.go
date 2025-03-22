package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
)

func editCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service) {
	log.Printf("editCommandHandler called")
	username := i.Member.User.Username

	options := i.ApplicationCommandData().Options

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "カスタムテンプレートを更新しています...",
		},
	})

	prompt := options[0].StringValue()

	if strings.ToLower(prompt) == "delete" {
		// /edit コマンドで "delete" と送信された場合、custom_model.json から該当ユーザーの行を削除する
		var customModelCfg CustomModelConfig
		data, err := os.ReadFile("json/custom_model.json")
		if err != nil {
			log.Printf("os.ReadFile error: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("エラーが発生しました: %v", err),
				},
			})
			return
		}

		err = json.Unmarshal(data, &customModelCfg)
		if err != nil {
			log.Printf("json.Unmarshal error: %v", err)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("エラーが発生しました: %v", err),
				},
			})
			return
		}

		if _, ok := customModelCfg.Prompts[username]; ok {
			delete(customModelCfg.Prompts, username)

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

			embed := &discordgo.MessageEmbed{
				Description: "あなたのカスタムテンプレートを削除しました！",
				Color:       0xa8ffee,
			}
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}
			return
		} else {
			embed := &discordgo.MessageEmbed{
				Description: "カスタムテンプレートが見つかりませんでした！",
				Color:       0xa8ffee,
			}
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				log.Printf("InteractionResponseEdit error: %v", err)
			}
			return
		}
	}

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

	embed := &discordgo.MessageEmbed{
		Description: "カスタムテンプレートを更新しました！",
		Color:       0xa8ffee,
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("InteractionResponseEdit error: %v", err)
	}
}
