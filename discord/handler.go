package discord

import (
	"log"
	"os"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/history"
)

func setupHandlers(s *discordgo.Session, geminiAPIKey string) error {
	const defaultMaxHistorySize = 10
	historyMgr := history.NewInMemoryHistoryManager(defaultMaxHistorySize)

	chatSvc, err := chat.NewChat(geminiAPIKey, historyMgr) 
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
			chatCommandHandler(s, i, chatSvc)
		case "reset":
			resetCommandHandler(s, i, historyMgr)
		case "about":
			aboutCommandHandler(s, i)
		case "edit":
			editCommandHandler(s, i, chatSvc)
		}
	})
	return nil
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! %s#%s", s.State.User.Username, s.State.User.Discriminator)
}
