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

// chatSvc と historyMgr を引数に追加
func setupHandlers(s *discordgo.Session, cfg *config.Config, chatSvc chat.Service, historyMgr history.HistoryManager) (history.HistoryManager, chat.Service, error) {
	var err error
	if historyMgr == nil {
		historyMgr, err = history.NewDuckDBHistoryManager()
		if err != nil {
			log.Printf("DuckDBHistoryManager の初期化に失敗しました: %v", err)
			return nil, nil, fmt.Errorf("DuckDBHistoryManager の初期化に失敗しました: %w", err)
		}
	}

	if chatSvc == nil {
		chatSvc, err = chat.NewChat(cfg, historyMgr)
		if err != nil {
			if cerr, ok := err.(interface{ Unwrap() error }); ok && cerr.Unwrap() != nil {
				log.Printf("Chat サービスの初期化に失敗しました: %v (underlying: %v)", err, cerr.Unwrap())
			} else {
				log.Printf("Chat サービスの初期化に失敗しました: %v", err)
			}
			return nil, nil, fmt.Errorf("Chat サービスの初期化に失敗しました: %w", err)
		}
	}

	// chat パッケージと discord パッケージでエラーロガーを共有
	// errorLogger の取得は chatSvc が nil でないことを保証してから行う
	var errorLogger *log.Logger
	if chatSvc != nil {
		errorLogger = chat.GetErrorLogger()
	}
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
			chatCommandHandler(s, i, chatSvc, threadID, cfg)
		case "reset":
			resetCommandHandler(s, i, historyMgr, threadID)
		case "about":
			aboutCommandHandler(s, i, cfg)
		case "edit":
			editCommandHandler(s, i, cfg)
		}
	})
	return historyMgr, chatSvc, nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

// MessageType defines the type of a received message.
type MessageType int

const (
	// MessageTypeNormal is a standard message in a channel.
	MessageTypeNormal MessageType = iota
	// MessageTypeDM is a direct message to the bot.
	MessageTypeDM
	// MessageTypeReply is a reply to the bot.
	MessageTypeReply
	// MessageTypeSelf is a message from the bot itself.
	MessageTypeSelf
)

// classifyMessageType determines the type of a message.
func classifyMessageType(s *discordgo.Session, m *discordgo.MessageCreate) MessageType {
	if m.Author.ID == s.State.User.ID {
		return MessageTypeSelf
	}
	if m.GuildID == "" {
		return MessageTypeDM
	}
	if m.ReferencedMessage != nil && m.ReferencedMessage.Author != nil && m.ReferencedMessage.Author.ID == s.State.User.ID {
		return MessageTypeReply
	}
	return MessageTypeNormal
}

// messageCreateHandler is the raw handler for discordgo's MessageCreate event.
// It classifies the message, resolves the thread ID, and delegates to the testable handleMessageEvent.
func messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate, chatSvc chat.Service, cfg *config.Config) {
	messageType := classifyMessageType(s, m)

	// Wrap the session for the interface
	wrappedSession := &discordgoSession{s}

	// Resolve thread ID only when necessary
	var threadID string
	if messageType == MessageTypeReply {
		threadID = resolveThreadID(wrappedSession, m.ChannelID)
	} else {
		threadID = m.ChannelID // For DMs or other cases
	}

	handleMessageEvent(wrappedSession, m, chatSvc, cfg, messageType, threadID)
}

// handleMessageEvent is the testable core logic for handling message events.
func handleMessageEvent(s DiscordSession, m *discordgo.MessageCreate, chatSvc chat.Service, cfg *config.Config, messageType MessageType, threadID string) {
	switch messageType {
	case MessageTypeSelf:
		return // Do nothing
	case MessageTypeDM:
		log.Printf("DM受信: UserID=%s, Username=%s, Content=%s", m.Author.ID, m.Author.Username, m.Content)
		handleDirectMessage(s, m, chatSvc, cfg)
	case MessageTypeReply:
		log.Printf("Botへの返信を受信: UserID=%s, Username=%s, Content=%s, ReferencedMessageID=%s", m.Author.ID, m.Author.Username, m.Content, m.ReferencedMessage.ID)
		handleReplyToBot(s, m, chatSvc, cfg, threadID)
	case MessageTypeNormal:
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

// handleDirectMessage はDMに対する応答を処理します
func handleDirectMessage(s DiscordSession, m *discordgo.MessageCreate, chatSvc chat.Service, cfg *config.Config) {
	if chatSvc == nil {
		log.Println("DM処理エラー: chatSvcがnilです")
		s.ChannelMessageSend(m.ChannelID, "内部エラーにより応答できませんでした。")
		return
	}
	if cfg == nil {
		log.Println("DM処理エラー: cfgがnilです")
		s.ChannelMessageSend(m.ChannelID, "内部エラーにより応答できませんでした。")
		return
	}

	responseText, _, _, err := chatSvc.GetResponse(m.Author.ID, m.ChannelID, m.Author.Username, m.Content, m.Timestamp.Format(time.RFC3339), cfg.Model.Prompts["default"])
	if err != nil {
		log.Printf("DM応答生成エラー: %v", err)
		s.ChannelMessageSend(m.ChannelID, "応答の生成中にエラーが発生しました。")
		return
	}
	if responseText == "" {
		log.Printf("DM応答が空です。")
		s.ChannelMessageSend(m.ChannelID, "応答がありませんでした。")
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, responseText)
	if err != nil {
		log.Printf("DM返信エラー: %v", err)
	}

	jst := m.Timestamp
	err = history.LogMessageCreate(
		m.ID,
		m.ChannelID,
		m.GuildID, // DMの場合は空文字列
		m.Author.ID,
		m.Author.Username,
		m.Content,
		jst,
	)
	if err != nil {
		log.Printf("Failed to log DM create event: %v", err)
	}
}

// resolveThreadID attempts to find the thread ID for a given channel ID using the session interface.
func resolveThreadID(s DiscordSession, channelID string) string {
	ch, err := s.StateChannel(channelID)
	if err != nil {
		// Fallback to fetching from API if not in state
		ch, err = s.Channel(channelID)
		if err != nil {
			log.Printf("Could not resolve channel %s: %v", channelID, err)
			return channelID // return original channelID as a fallback
		}
	}
	if ch.IsThread() {
		return ch.ID
	}
	return channelID
}

// handleReplyToBot はBotへの返信に対する応答を処理します
func handleReplyToBot(s DiscordSession, m *discordgo.MessageCreate, chatSvc chat.Service, cfg *config.Config, threadID string) {
	if chatSvc == nil {
		log.Println("Botへの返信処理エラー: chatSvcがnilです")
		s.ChannelMessageSend(m.ChannelID, "内部エラーにより応答できませんでした。")
		return
	}
	if cfg == nil {
		log.Println("Botへの返信処理エラー: cfgがnilです")
		s.ChannelMessageSend(m.ChannelID, "内部エラーにより応答できませんでした。")
		return
	}

	// 応答を生成
	responseText, _, _, err := chatSvc.GetResponse(m.Author.ID, threadID, m.Author.Username, m.Content, m.Timestamp.Format(time.RFC3339), cfg.Model.Prompts["default"])
	if err != nil {
		log.Printf("Botへの返信応答生成エラー: %v", err)
		s.ChannelMessageSend(m.ChannelID, "応答の生成中にエラーが発生しました。")
		return
	}
	if responseText == "" {
		log.Printf("Botへの返信応答が空です。")
		s.ChannelMessageSend(m.ChannelID, "応答がありませんでした。")
		return
	}

	// 返信としてメッセージを送信
	_, err = s.ChannelMessageSendReply(m.ChannelID, responseText, m.Reference())
	if err != nil {
		log.Printf("Botへの返信送信エラー: %v", err)
	}

	// 履歴に記録
	jst := m.Timestamp
	err = history.LogMessageCreate(
		m.ID,
		m.ChannelID,
		m.GuildID,
		m.Author.ID,
		m.Author.Username,
		m.Content,
		jst,
	)
	if err != nil {
		log.Printf("Failed to log reply message create event: %v", err)
	}
}
