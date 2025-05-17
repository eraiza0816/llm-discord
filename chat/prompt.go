package chat

import (
	"fmt"
	"log"
	"strings"

	"github.com/eraiza0816/llm-discord/history"
)

func buildFullInput(systemPrompt, userMessage string, historyMgr history.HistoryManager, userID string, threadID string, timestamp string) string {
	dateTimeInfo := fmt.Sprintf("Today is  %s .\n", timestamp)
	toolInstructions := `
【Function Calling Rules】
You can use the following tools (functions). Select the appropriate function based on the user's request and return a FunctionCall.
- getWeather: Use when asked about "weather" with a place name (e.g., "Tokyo", "Osaka City"). Do not use with a location code.
- getPainStatus: Use when asked about "headache" or "zutsuu" with a place name (e.g., "Yokohama"). Do not use with a location code.
- searchWeatherPoint: Use when you want to know the "location code" or search for a place name. A keyword (place name, etc.) is required.
- getOtenkiAspInfo: Use when asked about "Otenki ASP" information with a location code (e.g., "13112"). Do not use with a place name.
- get_url_content: Retrieves the main text content of the specified URL. Use when the user mentions a URL or wants to know the content of a web page.
`
	historyText := ""
	if historyMgr != nil {
		messages, err := historyMgr.Get(userID, threadID)
		if err != nil {
			// 履歴取得エラーはログに出力するが、処理は続行（履歴なしとして扱う）
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
