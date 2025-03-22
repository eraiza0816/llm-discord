package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func editCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("editCommandHandler called")
	username := i.Member.User.Username

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		// /edit コマンドが空の場合、custom_model.json から該当ユーザーの行を削除する
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

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "プロンプトテンプレートを削除しました！",
				},
			})
			return
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "プロンプトテンプレートが見つかりませんでした！",
				},
			})
			return
		}
	}

	// /edit コマンドが空でない場合、プロンプトを更新または削除する
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

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "プロンプトテンプレートを削除しました！",
				},
			})
			return
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "プロンプトテンプレートが見つかりませんでした！",
				},
			})
			return
		}
	}

	// /edit コマンドが空でない場合、プロンプトを更新する
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
