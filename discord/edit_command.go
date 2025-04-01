package discord

import (
	"fmt"
	"log"
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
		err := DeleteCustomPromptForUser(username)
		if err != nil {
			sendErrorResponse(s, i, fmt.Errorf("カスタムテンプレートの削除中にエラーが発生しました: %w", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Description: "あなたのカスタムテンプレートを削除しました！",
			Color:       0xa8ffee,
		}
		_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		if editErr != nil {
			log.Printf("InteractionResponseEdit error after deleting prompt: %v", editErr)
		}
		return
	}

	err := SetCustomPromptForUser(username, prompt)
	if err != nil {
		sendErrorResponse(s, i, fmt.Errorf("カスタムテンプレートの設定中にエラーが発生しました: %w", err))
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
