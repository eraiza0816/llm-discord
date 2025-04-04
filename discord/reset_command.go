package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/history"
)

func resetCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, historyMgr history.HistoryManager) {
	userID := i.Member.User.ID
	resetUsername := i.Member.User.Username + "#" + i.Member.User.Discriminator
	log.Printf("User %s performed a reset operation.", resetUsername)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "履歴をリセットしています...",
		},
	})

	historyMgr.Clear(userID)

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
