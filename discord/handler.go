package discord

import (
	"errors"
	"fmt"
	// "io" // 不要になったので削除
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/history"
)

func setupHandlers(s *discordgo.Session, geminiAPIKey string) (history.HistoryManager, error) { // 戻り値を history.HistoryManager に変更
	const defaultMaxHistorySize = 10 // SQLite側でもこの値を参照するようにする
	const dbPath = "data"             // データベースファイルの保存場所

	// historyMgr を DuckDBHistoryManager で初期化
	historyMgr, err := history.NewDuckDBHistoryManager()
	if err != nil {
		// エラーロガーが利用可能になる前にエラーが発生する可能性があるため、標準ログにも出力
		log.Printf("DuckDBHistoryManager の初期化に失敗しました: %v", err)
		return nil, fmt.Errorf("DuckDBHistoryManager の初期化に失敗しました: %w", err)
	}
	// TODO: アプリケーション終了時に historyMgr.Close() を呼び出す処理を追加する必要がある -> discord.go で対応済み

	chatSvc, err := chat.NewChat(geminiAPIKey, historyMgr)
	if err != nil {
		// エラーロガーが利用可能になる前にエラーが発生する可能性があるため、標準ログにも出力
		if cerr, ok := err.(interface{ Unwrap() error }); ok && cerr.Unwrap() != nil {
			log.Printf("Chat サービスの初期化に失敗しました: %v (underlying: %v)", err, cerr.Unwrap())
		} else {
			log.Printf("Chat サービスの初期化に失敗しました: %v", err)
		}
		return nil, fmt.Errorf("Chat サービスの初期化に失敗しました: %w", err)
	}

	// chat パッケージからエラーロガーを取得し、discord パッケージに設定
	errorLogger := chat.GetErrorLogger()
	SetErrorLogger(errorLogger)

	err = os.MkdirAll("log", 0755)
	if err != nil {
		// エラーロガーが設定されていればそちらにも出力
		if errorLogger != nil {
			errorLogger.Printf("log ディレクトリの作成に失敗しました: %v", err)
		}
		return nil, fmt.Errorf("log ディレクトリの作成に失敗しました: %w", err)
	}

	logFile, err := os.OpenFile("log/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		// エラーロガーが設定されていればそちらにも出力
		if errorLogger != nil {
			errorLogger.Printf("ログファイル 'log/app.log' のオープンに失敗しました: %v", err)
		}
		return nil, fmt.Errorf("ログファイル 'log/app.log' のオープンに失敗しました: %w", err)
	}
	log.SetOutput(logFile) // 標準ロガーは app.log に出力

	s.AddHandler(onReady)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		// スレッドIDを取得。スレッドでない場合はチャンネルIDを使用。
		var threadID string
		if i.ChannelID != "" { // 通常のインタラクションでは ChannelID があるはず
			ch, err := s.State.Channel(i.ChannelID) // キャッシュからチャンネル情報を取得
			if err != nil {
				// キャッシュにない場合はAPIから取得
				ch, err = s.Channel(i.ChannelID)
				if err != nil {
					sendEphemeralErrorResponse(s, i, fmt.Errorf("チャンネル情報の取得に失敗しました: %w", err))
					return
				}
			}

			if ch.IsThread() {
				threadID = ch.ID
			} else {
				// スレッドでない場合は、チャンネルIDをスレッドIDとして扱う
				// もしチャンネルごとの履歴を管理しない場合は、ここで空文字列にするか、
				// chatCommandHandler などで threadID がチャンネルIDである場合の分岐処理を入れる。
				// 今回はチャンネルIDをそのまま渡す。
				threadID = i.ChannelID
			}
		} else if i.Message != nil && i.Message.ChannelID != "" { // メッセージコンポーネントのインタラクションの場合
			// メッセージコンポーネントのインタラクションの場合、i.ChannelID が空になることがある。
			// その場合は i.Message.ChannelID を参照する。
			ch, err := s.State.Channel(i.Message.ChannelID)
			if err != nil {
				ch, err = s.Channel(i.Message.ChannelID)
				if err != nil {
					sendEphemeralErrorResponse(s, i, fmt.Errorf("メッセージチャンネル情報の取得に失敗しました: %w", err))
					return
				}
			}
			if ch.IsThread() {
				threadID = ch.ID
			} else {
				threadID = i.Message.ChannelID
			}
		} else {
			// 稀なケースだが、どちらも取得できない場合はエラーレスポンス
			sendEphemeralErrorResponse(s, i, errors.New("スレッドIDまたはチャンネルIDの取得に失敗しました"))
			return
		}


		switch i.ApplicationCommandData().Name {
		case "chat":
			chatCommandHandler(s, i, chatSvc, threadID)
		case "reset":
			resetCommandHandler(s, i, historyMgr, threadID)
		case "about":
			aboutCommandHandler(s, i) // about はスレッドに依存しない
		case "edit":
			editCommandHandler(s, i, chatSvc) // edit も現状スレッドに依存しない
		}
	})
	return historyMgr, nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}
