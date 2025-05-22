package discord

import (
	"errors"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/history"
)

func StartBot(cfg *config.Config) error {
	log.Println("StartBot called")
	if cfg == nil {
		return errors.New("Config is nil")
	}

	session, err := discordgo.New("Bot " + cfg.DiscordBotToken)
	if err != nil {
		log.Printf("Error creating Discord session: %v", err)
		return err
	}
	defer func() {
		log.Println("Closing Discord session at the end of StartBot...")
		if err := session.Close(); err != nil {
			log.Printf("Error closing Discord session: %v", err)
		}
	}()

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "chat",
			Description: "おしゃべりしようよ",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "メッセージ",
					Required:    true,
				},
			},
		},
		{
			Name:        "reset",
			Description: "あなたとのチャット履歴をリセット",
		},
		{
			Name:        "about",
			Description: "このBotについて",
		},
		{
			Name:        "edit",
			Description: "カスタムテンプレートであなたの好みに編集",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "set_custom_prompt",
					Description: "カスタムテンプレートを入力(deleteで削除)",
					Required:    true,
				},
			},
		},
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds

	err = session.Open()
	if err != nil {
		log.Printf("Error opening Discord session: %v", err)
		return err
	}

	log.Printf("session.State after open: %v", session.State)
	if session.State == nil {
		log.Println("session.State is nil after open")
		return errors.New("session.State is nil after open")
	}
	log.Printf("session.State.User after open: %v", session.State.User)
	if session.State.User == nil {
		log.Println("session.State.User is nil after open")
		return errors.New("session.State.User is nil after open")
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	log.Println("Registering commands...")
	for i, v := range commands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
		if err != nil {
			log.Printf("Can not create '%v' command for UserID %s: %v", v.Name, session.State.User.ID, err)
			continue
		}
		registeredCommands[i] = cmd
	}

	// setupHandlers から history.HistoryManager と chat.Service を受け取る
	var historyMgr history.HistoryManager
	var chatSvc chat.Service // chat.Service 型の変数を宣言
	historyMgr, chatSvc, err = setupHandlers(session, cfg, chatSvc, historyMgr) // chatSvc と historyMgr を渡す
	if err != nil {
		log.Printf("Error in setupHandlers: %v", err)
		return fmt.Errorf("ハンドラの設定中にエラーが発生しました: %w", err)
	}

	// AddHandler for messageCreate to pass chatSvc and cfg
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageCreateHandler(s, m, chatSvc, cfg)
	})
	session.AddHandler(messageUpdateHandler) //  messageUpdateHandler と messageDeleteHandler は変更なし
	session.AddHandler(messageDeleteHandler)


	// HistoryManager のクローズ処理
	if historyMgr != nil {
		defer func() {
			log.Println("Closing HistoryManager via defer in StartBot...")
			if closeErr := historyMgr.Close(); closeErr != nil {
				log.Printf("Error closing HistoryManager in StartBot defer: %v", closeErr)
			}
		}()
	}

	// ChatService のクローズ処理
	if chatSvc != nil {
		defer func() {
			log.Println("Closing ChatService via defer in StartBot...")
			chatSvc.Close() // Closeメソッドを呼び出す
		}()
	}

	log.Println("Bot setup complete. Waiting for signals from main.")
	// session.Close() はこの関数の冒頭で defer されているため、ここでは不要

	// main.goからのシグナルで適切に終了処理が行われるよう、ここではブロックし続ける
	select {}
}
