package discord

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/history"
)

func setupHandlers(s *discordgo.Session, cfg *config.Config) (history.HistoryManager, chat.Service, error) {
	historyMgr, err := history.NewDuckDBHistoryManager()
	if err != nil {
		log.Printf("DuckDBHistoryManager の初期化に失敗しました: %v", err)
		return nil, nil, fmt.Errorf("DuckDBHistoryManager の初期化に失敗しました: %w", err)
	}

	chatSvc, err := chat.NewChat(cfg, historyMgr)
	if err != nil {
		if cerr, ok := err.(interface{ Unwrap() error }); ok && cerr.Unwrap() != nil {
			log.Printf("Chat サービスの初期化に失敗しました: %v (underlying: %v)", err, cerr.Unwrap())
		} else {
			log.Printf("Chat サービスの初期化に失敗しました: %v", err)
		}
		return nil, nil, fmt.Errorf("Chat サービスの初期化に失敗しました: %w", err)
	}

	// chat パッケージと discord パッケージでエラーロガーを共有
	errorLogger := chat.GetErrorLogger()
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
			chatCommandHandler(s, i, chatSvc, threadID, cfg) // cfgを渡す
		case "reset":
			resetCommandHandler(s, i, historyMgr, threadID)
		case "about":
			aboutCommandHandler(s, i, cfg) // cfgを渡す
		case "edit":
			editCommandHandler(s, i, chatSvc)
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

	if m.EditedTimestamp == nil {
		log.Printf("EditedTimestamp is nil, skipping update log for MessageID: %s", m.ID)
		return
	}

	updateErr := history.LogMessageUpdate(
		m.ID,
		m.Content,
		*m.EditedTimestamp,
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
