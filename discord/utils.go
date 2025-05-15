package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var errorLogger *log.Logger // エラーログ用ロガー

// SetErrorLogger はエラーログ用ロガーを設定します。
func SetErrorLogger(logger *log.Logger) {
	errorLogger = logger
}

func sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	if errorLogger != nil { // ロガーが設定されているか確認
		errorLogger.Printf("Error occurred: %v", err) // エラーロガーを使用
	} else {
		log.Printf("Error occurred: %v", err) // ロガーが設定されていない場合は標準ログを使用
	}

	content := fmt.Sprintf("エラーが発生しました: %v", err)
	_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
		Embeds:  &[]*discordgo.MessageEmbed{},
	})
	if editErr != nil {
		if errorLogger != nil { // ロガーが設定されているか確認
			errorLogger.Printf("Failed to send error response via InteractionResponseEdit: %v (original error: %v)", editErr, err) // エラーロガーを使用
		} else {
			log.Printf("Failed to send error response via InteractionResponseEdit: %v (original error: %v)", editErr, err) // ロガーが設定されていない場合は標準ログを使用
		}
	}
}

func sendEphemeralErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	if errorLogger != nil { // ロガーが設定されているか確認
		errorLogger.Printf("Error occurred: %v", err) // エラーロガーを使用
	} else {
		log.Printf("Error occurred: %v", err) // ロガーが設定されていない場合は標準ログを使用
	}
	content := fmt.Sprintf("エラーが発生しました: %v", err)
	errResp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
	if respErr := s.InteractionRespond(i.Interaction, errResp); respErr != nil {
		if errorLogger != nil { // ロガーが設定されているか確認
			errorLogger.Printf("Failed to send ephemeral error response: %v (original error: %v)", respErr, err) // エラーロガーを使用
		} else {
			log.Printf("Failed to send ephemeral error response: %v (original error: %v)", respErr, err) // ロガーが設定されていない場合は標準ログを使用
		}
	}
}
