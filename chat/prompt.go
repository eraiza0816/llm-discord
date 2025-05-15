package chat

import (
	"strings"

	"github.com/eraiza0816/llm-discord/history"
)

func buildFullInput(systemPrompt, userMessage string, historyMgr history.HistoryManager, userID string, threadID string) string {
	toolInstructions := `
【Function Calling Rules】
あなたは以下のツール（関数）を利用できます。ユーザーのリクエストに応じて適切な関数を選択し、FunctionCallを返してください。
- getWeather: 「天気」について、地名（例：「東京」、「大阪市」）で質問された場合に使います。地点コードでは使いません。
- getPainStatus: 「頭痛」や「ずつう」について、地名（例：「横浜」）で質問された場合に使います。地点コードでは使いません。
- searchWeatherPoint: 「地点コード」を知りたい、または地名を検索したい場合に使います。キーワード（地名など）が必要です。
- getOtenkiAspInfo: 「Otenki ASP」の情報について、地点コード（例：「13112」）で質問された場合に使います。地名では使いません。
`
	userHistory := historyMgr.Get(userID, threadID)
	historyText := ""
	if userHistory != "" {
		historyText = "会話履歴:\n" + userHistory + "\n\n"
	}

	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString(toolInstructions)
	sb.WriteString("\n\n") // ツール指示と履歴の間の空行
	sb.WriteString(historyText) // 履歴がない場合は空文字列が追加される
	sb.WriteString("ユーザーのメッセージ:\n")
	sb.WriteString(userMessage)

	return sb.String()
}
