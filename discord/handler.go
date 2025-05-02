package discord

import (
	"log"
	"os"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/history"
	// loader はここでは不要になる
)

func setupHandlers(s *discordgo.Session, geminiAPIKey string) error {
	// model.json の読み込みは各コマンドハンドラで行うため、ここでは削除
	// defaultPrompt と maxHistorySize も model.json から取得するため、一旦固定値や別の方法で設定するか、
	// ChatService や HistoryManager の初期化方法を見直す必要がある。
	// ここでは一旦、HistoryManager のサイズは固定値にし、ChatService の初期化から modelCfg 関連を削除する。
	// defaultPrompt は ChatService 内で model.json から取得するように変更する想定。
	const defaultMaxHistorySize = 10 // 仮のデフォルト値
	historyMgr := history.NewInMemoryHistoryManager(defaultMaxHistorySize)

	// chat.NewChat のシグネチャ変更を想定し、modelCfg 関連を削除
	chatSvc, err := chat.NewChat(geminiAPIKey, historyMgr) // modelName, defaultPrompt, modelCfg を削除
	if err != nil {
		return fmt.Errorf("Chat サービスの初期化に失敗しました: %w", err)
	}

	err = os.MkdirAll("log", 0755)
	if err != nil {
		return fmt.Errorf("log ディレクトリの作成に失敗しました: %w", err)
	}

	logFile, err := os.OpenFile("log/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ログファイル 'log/app.log' のオープンに失敗しました: %w", err)
	}
	log.SetOutput(logFile)

	s.AddHandler(onReady)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "chat":
			// chatCommandHandler に modelCfg を渡さないように変更
			chatCommandHandler(s, i, chatSvc)
		case "reset":
			resetCommandHandler(s, i, historyMgr)
		case "about":
			// aboutCommandHandler に modelCfg を渡さないように変更
			aboutCommandHandler(s, i)
		case "edit":
			// editCommandHandler は modelCfg を使っていないので変更なし
			editCommandHandler(s, i, chatSvc)
		}
	})
	return nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}
