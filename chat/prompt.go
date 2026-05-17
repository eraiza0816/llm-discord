package chat

import (
	"fmt"
	"log"
	"strings"

	"github.com/eraiza0816/llm-discord/history"
)

func buildFullInput(systemPrompt, userMessage string, historyMgr history.HistoryManager, userID string, threadID string, timestamp string) string {
	dateTimeInfo := fmt.Sprintf("Today is  %s .\n", timestamp)
	toolInstructions := ""
	historyText := ""
	if historyMgr != nil {
		messages, err := historyMgr.Get(userID, threadID)
		if err != nil {
			log.Printf("ユーザー %s のスレッド %s の履歴取得に失敗しました: %v", userID, threadID, err)
		} else {
			var historyParts []string
			for _, msg := range messages {
				role := msg.Role
				if role == "model" {
					role = "assistant"
				}
				historyParts = append(historyParts, fmt.Sprintf("%s: %s", role, msg.Content))
			}
			if len(historyParts) > 0 {
				historyText = "Chat history:\n" + strings.Join(historyParts, "\n") + "\n\n"
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n")
	sb.WriteString(dateTimeInfo)
	sb.WriteString("\n")
	sb.WriteString(toolInstructions)
	sb.WriteString("\n\n")
	sb.WriteString(historyText)
	sb.WriteString("User message:\n")
	sb.WriteString(userMessage)

	return sb.String()
}
