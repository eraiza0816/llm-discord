package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/config"
)

func chatCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, threadID string, cfg *config.Config) {
	if cfg == nil {
		log.Println("chatCommandHandler: cfg is nil")
		sendErrorResponse(s, i, fmt.Errorf("設定が読み込まれていません。"))
		return
	}
	if cfg.Model == nil {
		log.Println("chatCommandHandler: cfg.Model is nil")
		sendErrorResponse(s, i, fmt.Errorf("モデル設定が読み込まれていません。"))
		return
	}
	modelCfg := cfg.Model

	var username string
	var userID string
	var avatarURL string

	if i.Member != nil && i.Member.User != nil {
		username = i.Member.User.Username
		userID = i.Member.User.ID
		avatarURL = i.Member.User.AvatarURL("")
	} else if i.User != nil {
		username = i.User.Username
		userID = i.User.ID
		avatarURL = i.User.AvatarURL("")
	} else {
		log.Println("chatCommandHandler: User information not found in interaction")
		sendErrorResponse(s, i, fmt.Errorf("ユーザー情報が取得できませんでした。"))
		return
	}

	message := i.ApplicationCommandData().Options[0].StringValue()
	timestamp := time.Now().Format(time.RFC3339)

	log.Printf("User %s (ID: %s, Thread: %s) sent message: %s ", username, userID, threadID, message)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ちょっと待ってね！",
		},
	})

	// GetResponse の最後の引数に cfg.Model.Prompts["default"] を渡す
	// userPrompt のロジックは chat.GetResponse 内に移動したため削除
	response, elapsed, modelName, err := chatSvc.GetResponse(userID, threadID, username, message, timestamp, cfg.Model.Prompts["default"], false)
	if err != nil {
		sendErrorResponse(s, i, fmt.Errorf("LLMからの応答取得中にエラーが発生しました: %w", err))
		return
	}

	embedUser := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    username,
			IconURL: avatarURL,
		},
		Fields: []*discordgo.MessageEmbedField{
			{Value: message},
		},
		Color: 0xfff9b7,
	}

	embedBot := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    modelCfg.Name,
			IconURL: modelCfg.Icon,
		},
		Fields: SplitToEmbedFields(response),
		Color:  0xa8ffee,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%vms %s", elapsed, modelName),
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embedUser, embedBot},
	})
	if err != nil {
		log.Printf("InteractionResponseEdit error: %v", err)
	}
}
