package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/loader" // loader を使うので残す
)

// modelCfg を引数から削除
func chatCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service) {
	// コマンド実行時に model.json を読み込む
	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		log.Printf("Error loading model config: %v", err)
		// エラー時はユーザーに通知し、処理を中断
		sendEphemeralErrorResponse(s, i, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err))
		return
	}

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

	// chatSvc.GetResponse の呼び出しは現状維持。chat.Service の修正は後で行う。
	// ただし、userPrompt の取得ロジックは modelCfg を使うので、読み込んだ modelCfg を使うようにする。
	// また、GetResponse に modelCfg を渡す必要があるかもしれないため、chat.Service のインターフェースと実装を確認・修正する必要がある。
	// ここでは一旦、既存の GetResponse を呼び出すが、後続の修正が必要。
	response, elapsed, modelName, err := chatSvc.GetResponse(userID, username, message, timestamp, userPrompt) // modelCfg が必要になる可能性
	if err != nil {
		// エラーレスポンス関数も modelCfg を使わないように修正が必要か確認 -> sendErrorResponse は使っていない
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

	// embedBot の Author 情報は読み込んだ modelCfg から取得する
	embedBot := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    modelCfg.Name, // ローカルで読み込んだ modelCfg を使用
			IconURL: modelCfg.Icon, // ローカルで読み込んだ modelCfg を使用
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
