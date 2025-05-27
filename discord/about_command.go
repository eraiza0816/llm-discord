package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/config"
)

func aboutCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) {
	if cfg == nil {
		log.Println("Error in aboutCommandHandler: cfg is nil")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "設定情報が読み込めませんでした。管理者に連絡してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if cfg.Model == nil {
		log.Println("Error in aboutCommandHandler: cfg.Model is nil")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "モデル設定が読み込めませんでした。管理者に連絡してください。",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	modelCfg := cfg.Model

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

	_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if editErr != nil {
		log.Printf("InteractionResponseEdit error: %v", editErr)
	}
}
