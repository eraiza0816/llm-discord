package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/loader"
)

func chatCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, modelCfg *loader.ModelConfig) {
	username := i.Member.User.Username
	message := i.ApplicationCommandData().Options[0].StringValue()
	timestamp := time.Now().Format(time.RFC3339)
	userID := i.Member.User.ID

	var userPrompt string

	customPrompt, exists, err := GetCustomPromptForUser(username)
	if err != nil {
		// エラーが発生したらログを出力し、デフォルトプロンプトを使用
		log.Printf("カスタムプロンプトの取得中にエラーが発生しました: %v。デフォルトプロンプトを使用します。", err)
		// ユーザーにエラーを通知せず、処理を続行
		userPrompt = modelCfg.GetPromptByUser(username) // デフォルトを使用
	} else if exists {
		// カスタムプロンプトが存在すればそれを使用
		log.Printf("ユーザー %s のカスタムプロンプトを使用します。", username)
		userPrompt = customPrompt
	} else {
		// カスタムプロンプトが存在しなければデフォルトを使用
		userPrompt = modelCfg.GetPromptByUser(username)
	}

	log.Printf("User %s sent message: %s ", username, message)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "ちょっと待ってね！",
		},
	})

	response, elapsed, modelName, err := chatSvc.GetResponse(userID, username, message, timestamp, userPrompt)
	if err != nil {
		sendErrorResponse(s, i, fmt.Errorf("LLMからの応答取得中にエラーが発生しました: %w", err))
		return
	}

	embedUser := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    username,
			IconURL: i.Member.User.AvatarURL(""),
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
		Fields: splitToEmbedFields(response),
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
