package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/loader"
)

// openaiStreamingResponse は OpenAI 互換 API のストリーミングレスポンスの1行分を表します。
type openaiStreamingResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// openaiChatMessage は OpenAI 互換 API リクエストのメッセージを表します。
type openaiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiChatRequest は OpenAI 互換 API のチャット補完リクエストを表します。
type openaiChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openaiChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

// getOpenAIResponse は OpenAI 互換 API エンドポイント（v1/chat/completions）にリクエストを送信し、
// ストリーミング応答からテキストを取得します。
func (c *Chat) getOpenAIResponse(userID, threadID, message, fullInput string, openaiCfg loader.OpenAIConfig) (string, float64, error) {
	start := time.Now()

	if openaiCfg.APIEndpoint == "" || openaiCfg.ModelName == "" {
		return "", 0, fmt.Errorf("OpenAI APIエンドポイントまたはモデル名が設定されていません")
	}

	// エンドポイント末尾に /chat/completions がなければ補完
	url := strings.TrimRight(openaiCfg.APIEndpoint, "/")
	if !strings.HasSuffix(url, "/chat/completions") && !strings.HasSuffix(url, "/chat/completions/") {
		url += "/chat/completions"
	}

	reqBody := openaiChatRequest{
		Model: openaiCfg.ModelName,
		Messages: []openaiChatMessage{
			{
				Role:    "user",
				Content: fullInput,
			},
		},
		Stream:    true,
		MaxTokens: 4096,
	}

	jsonPayload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("OpenAIリクエストペイロードのJSON作成に失敗: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", 0, fmt.Errorf("OpenAIリクエストの作成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// APIKey が空でなければ Authorization ヘッダーを設定
	if openaiCfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+openaiCfg.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("OpenAI APIへのリクエストに失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("OpenAI APIエラー: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	reader := bufio.NewReader(resp.Body)
	responseText, err := parseOpenAIStreamResponse(reader)
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		return "", elapsed, fmt.Errorf("OpenAIレスポンスの解析に失敗しました: %w", err)
	}

	responseText = strings.TrimSpace(responseText)

	if responseText != "" {
		c.historyMgr.Add(userID, threadID, message, responseText)
	}

	return responseText, elapsed, nil
}

// parseOpenAIStreamResponse は OpenAI 互換 API の Server-Sent Events (SSE) ストリームを解析します。
// 各行は "data: <json>" の形式で送信され、"data: [DONE]" で終了します。
func parseOpenAIStreamResponse(reader *bufio.Reader) (string, error) {
	var responseTextBuilder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return responseTextBuilder.String(), fmt.Errorf("ストリーム読み込みエラー: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// "data: " プレフィックスを除去
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// ストリーム終了シグナル
		if data == "[DONE]" {
			break
		}

		var streamResp openaiStreamingResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// パースに失敗した行はスキップ（不完全な行の可能性）
			continue
		}

		if len(streamResp.Choices) > 0 {
			delta := streamResp.Choices[0].Delta
			if delta.Content != "" {
				responseTextBuilder.WriteString(delta.Content)
			}
		}
	}

	return responseTextBuilder.String(), nil
}
