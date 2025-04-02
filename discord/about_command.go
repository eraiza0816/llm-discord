package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/loader" // loader を使うので残す
)

// modelCfg を引数から削除
func aboutCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// コマンド実行時に model.json を読み込む
	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		log.Printf("Error loading model config: %v", err)
		// エラー時はユーザーに通知し、処理を中断
		sendEphemeralErrorResponse(s, i, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err))
		return
	}

	username := i.Member.User.Username
	log.Printf("User %s performed an about operation.", username)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "少々お待ちください...",
		},
	})

	// modelCfg の nil チェックは LoadModelConfig のエラーハンドリングでカバーされるため不要

	// 読み込んだ modelCfg を使用して Embed を作成
	embed := &discordgo.MessageEmbed{
		Title:       modelCfg.About.Title,
		Description: modelCfg.About.Description,
		URL:         modelCfg.About.URL,
		Color:       0xa8ffee,
	}

	// err は LoadModelConfig で宣言済みなので、ここでは代入演算子 = を使う
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("InteractionResponseEdit error: %v", err)
	}
}
