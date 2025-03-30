package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	log.Printf("Error occurred: %v", err)

	content := fmt.Sprintf("エラーが発生しました: %v", err)
	_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &[]*discordgo.MessageEmbed{},
	})
	if editErr != nil {
		log.Printf("Failed to send error response via InteractionResponseEdit: %v (original error: %v)", editErr, err)
	}
}

func sendEphemeralErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	log.Printf("Error occurred: %v", err)
	content := fmt.Sprintf("エラーが発生しました: %v", err)
	errResp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
	if respErr := s.InteractionRespond(i.Interaction, errResp); respErr != nil {
		log.Printf("Failed to send ephemeral error response: %v (original error: %v)", respErr, err)
	}
}