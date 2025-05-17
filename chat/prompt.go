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
- getWeather: 「天気」について、地名（例：「東京」、「大阪市」）で質問された場合に使います。地点コードでは使いません。
- getPainStatus: 「頭痛」や「ずつう」について、地名（例：「横浜」）で質問された場合に使います。地点コードでは使いません。
- searchWeatherPoint: 「地点コード」を知りたい、または地名を検索したい場合に使います。キーワード（地名など）が必要です。
- getOtenkiAspInfo: 「Otenki ASP」の情報について、地点コード（例：「13112」）で質問された場合に使います。地名では使いません。
- get_url_content: 指定されたURLの主要なテキストコンテンツを取得します。ユーザーがURLについて言及したり、ウェブページの内容を知りたい場合に利用します。
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
