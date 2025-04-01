package chat

import (
	"github.com/google/generative-ai-go/genai"
	"strings"
)

// getResponseText は GenerateContentResponse からテキスト部分を抽出する。
func getResponseText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		// ログ出力は呼び出し元で行うか、ここで追加するか検討する。
		// log.Println("Warning: Empty or nil response from Gemini API.")
		return "Gemini APIからの応答がありませんでした。" // ユーザー向けメッセージ
	}

	var responseText strings.Builder // strings.Builder を使用する。
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText.WriteString(string(text)) // WriteString を使用する。
		}
		// 他の Part タイプ (FunctionCall など) は無視する。
	}

	finalText := responseText.String() // 最後に文字列に変換する。
	if finalText == "" {
		// テキスト部分が空の場合のログや代替メッセージ
		// log.Println("Warning: Extracted response text is empty.")
		// return "応答からテキストを抽出できませんでした。"
	}

	return finalText
}
