package chat

import (
	"strings"

	"github.com/eraiza0816/llm-discord/history"
)

// buildFullInput は、システムプロンプト、ツール指示、履歴、ユーザーメッセージを結合して
// LLMへの完全な入力文字列を構築します。
func buildFullInput(systemPrompt, userMessage string, historyMgr history.HistoryManager, userID string) string {
	// Function Calling のルール説明
	toolInstructions := `
【Function Calling Rules】
あなたは以下のツール（関数）を利用できます。ユーザーのリクエストに応じて適切な関数を選択し、FunctionCallを返してください。
- getWeather: 「天気」について、地名（例：「東京」、「大阪市」）で質問された場合に使います。地点コードでは使いません。
- getPainStatus: 「頭痛」や「ずつう」について、地名（例：「横浜」）で質問された場合に使います。地点コードでは使いません。
- searchWeatherPoint: 「地点コード」を知りたい、または地名を検索したい場合に使います。キーワード（地名など）が必要です。
- getOtenkiAspInfo: 「Otenki ASP」の情報について、地点コード（例：「13112」）で質問された場合に使います。地名では使いません。
`
	// ユーザー履歴の取得
	userHistory := historyMgr.Get(userID)
	historyText := ""
	if userHistory != "" {
		// 履歴が空でない場合のみ "会話履歴:" プレフィックスを追加
		historyText = "会話履歴:\n" + userHistory + "\n\n"
	}

	// 全体の組み立て
	// 順序: システムプロンプト -> ツール指示 -> 履歴 -> ユーザーメッセージ
	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString(toolInstructions)
	sb.WriteString("\n\n") // ツール指示と履歴の間の空行
	sb.WriteString(historyText) // 履歴がない場合は空文字列が追加される
	sb.WriteString("ユーザーのメッセージ:\n")
	sb.WriteString(userMessage)

	return sb.String()
}
