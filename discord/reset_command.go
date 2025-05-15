package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/history"
)

func resetCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, historyMgr history.HistoryManager, threadID string) {
	// userID := i.Member.User.ID // スレッド全体の履歴を消すので、特定のユーザーIDは不要
	resetUsername := i.Member.User.Username + "#" + i.Member.User.Discriminator
	log.Printf("User %s performed a reset operation for thread/channel ID: %s.", resetUsername, threadID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "このスレッド/チャンネルの履歴をリセットしています...",
		},
	})

	historyMgr.ClearAllByThreadID(threadID)

	embed := &discordgo.MessageEmbed{
		Description: "このスレッド/チャンネルのチャット履歴をリセットしました！",
		Color:       0xa8ffee,
	}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("InteractionResponseEdit error: %v", err)
	}
}
