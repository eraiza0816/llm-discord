package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

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
