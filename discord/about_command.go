package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/loader"
)

func aboutCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		log.Printf("Error loading model config: %v", err)
		sendEphemeralErrorResponse(s, i, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err))
		return
	}

	username := i.Member.User.Username
	log.Printf("User %s performed an about operation.", username)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "少々お待ちください...",
		},
	})

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
