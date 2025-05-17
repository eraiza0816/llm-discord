package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/config"
)

func aboutCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) { // cfg パラメータを追加
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
