package discord

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/config"
)

func editCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) {
	log.Printf("editCommandHandler called")

	var username string
	if i.Member != nil && i.Member.User != nil {
		username = i.Member.User.Username
	} else if i.User != nil { // DMからの場合
		username = i.User.Username
	} else {
		log.Println("editCommandHandler: User information not found in interaction")
		sendErrorResponse(s, i, fmt.Errorf("ユーザー情報が取得できませんでした。"))
		return
	}

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
