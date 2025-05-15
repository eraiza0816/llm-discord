package discord

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/history"
)

func setupHandlers(s *discordgo.Session, geminiAPIKey string) (history.HistoryManager, chat.Service, error) {
	const defaultMaxHistorySize = 20
	const dbPath = "data"

	historyMgr, err := history.NewDuckDBHistoryManager()
	if err != nil {
		log.Printf("DuckDBHistoryManager の初期化に失敗しました: %v", err)
		return nil, nil, fmt.Errorf("DuckDBHistoryManager の初期化に失敗しました: %w", err)
	}

	chatSvc, err := chat.NewChat(geminiAPIKey, historyMgr)
	if err != nil {
		if cerr, ok := err.(interface{ Unwrap() error }); ok && cerr.Unwrap() != nil {
			log.Printf("Chat サービスの初期化に失敗しました: %v (underlying: %v)", err, cerr.Unwrap())
		} else {
			log.Printf("Chat サービスの初期化に失敗しました: %v", err)
		}
		return nil, nil, fmt.Errorf("Chat サービスの初期化に失敗しました: %w", err)
	}

	errorLogger := chat.GetErrorLogger() // chat.GetErrorLogger() は chat.Service に関連付けられていないグローバルなものかもしれない
	SetErrorLogger(errorLogger)

	err = os.MkdirAll("log", 0755)
	if err != nil {
		if errorLogger != nil {
			errorLogger.Printf("log ディレクトリの作成に失敗しました: %v", err)
		}
		return nil, nil, fmt.Errorf("log ディレクトリの作成に失敗しました: %w", err)
	}

	logFile, err := os.OpenFile("log/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		if errorLogger != nil {
			errorLogger.Printf("ログファイル 'log/app.log' のオープンに失敗しました: %v", err)
		}
		return nil, nil, fmt.Errorf("ログファイル 'log/app.log' のオープンに失敗しました: %w", err)
	}
	log.SetOutput(logFile)

	s.AddHandler(onReady)
	s.AddHandler(messageCreateHandler)
	s.AddHandler(messageUpdateHandler)
	s.AddHandler(messageDeleteHandler)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		var threadID string
		if i.ChannelID != "" {
			ch, err := s.State.Channel(i.ChannelID)
			if err != nil {
				ch, err = s.Channel(i.ChannelID)
				if err != nil {
					sendEphemeralErrorResponse(s, i, fmt.Errorf("チャンネル情報の取得に失敗しました: %w", err))
					return
				}
			}

			if ch.IsThread() {
				threadID = ch.ID
			} else {
				threadID = i.ChannelID
			}
		} else if i.Message != nil && i.Message.ChannelID != "" {
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
			sendEphemeralErrorResponse(s, i, errors.New("スレッドIDまたはチャンネルIDの取得に失敗しました"))
			return
		}


		switch i.ApplicationCommandData().Name {
		case "chat":
			chatCommandHandler(s, i, chatSvc, threadID)
		case "reset":
			resetCommandHandler(s, i, historyMgr, threadID)
		case "about":
			aboutCommandHandler(s, i)
		case "edit":
			editCommandHandler(s, i, chatSvc) // chatSvc は chat.Service 型
		}
	})
	return historyMgr, chatSvc, nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	jst := m.Timestamp

	err := history.LogMessageCreate(
		m.ID,
		m.ChannelID,
		m.GuildID,
		m.Author.ID,
		m.Author.Username,
		m.Content,
		jst,
	)
	if err != nil {
		log.Printf("Failed to log message create event: %v", err)
	}
}

func messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	if m.Author != nil && m.Author.ID == s.State.User.ID {
		return
	}
	if m.Message == nil || m.EditedTimestamp == nil {
		log.Printf("Message update event skipped due to missing message data or edited timestamp: MessageID=%s", m.ID)
		return
	}

	editedJst := *m.EditedTimestamp

	updateErr := history.LogMessageUpdate(
		m.ID,
		m.Content,
		editedJst,
	)
	if updateErr != nil {
		log.Printf("Failed to log message update event: %v", updateErr)
	}
}

func messageDeleteHandler(s *discordgo.Session, m *discordgo.MessageDelete) {
	deletedJst := japanStandardTime()

	deleteErr := history.LogMessageDelete(
		m.ID,
		deletedJst,
	)
	if deleteErr != nil {
		log.Printf("Failed to log message delete event: %v", deleteErr)
	}
}

func japanStandardTime() time.Time {
	return time.Now().In(time.FixedZone("JST", 9*60*60))
}
