package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"github.com/eraiza0816/llm-discord/loader" // loader.OllamaConfig を使うためインポートは必要
)

// getOllamaResponse は Ollama API にリクエストを送信し、ストリーミングレスポンスを処理します。
// 引数に ollamaCfg を追加
func (c *Chat) getOllamaResponse(userID, message, fullInput string, ollamaCfg loader.OllamaConfig) (string, float64, error) {
	// Ollama 向けの履歴処理が必要な場合は、ここで userHistory を取得・加工する (今回は fullInput をそのまま使う)

	start := time.Now()
	// 引数で渡された ollamaCfg を使用
	url := ollamaCfg.APIEndpoint
	modelName := ollamaCfg.ModelName
	if url == "" || modelName == "" {
		return "", 0, fmt.Errorf("Ollama APIエンドポイントまたはモデル名が設定されていません")
	}

	payload := map[string]string{
		"prompt": fullInput,
		"model":  modelName,
		// "stream": "false", // ストリームしない場合は false を指定できるが、現状はストリーム前提
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("OllamaリクエストペイロードのJSON作成に失敗: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", 0, fmt.Errorf("Ollamaリクエストの作成に失敗: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// TODO: タイムアウト設定などを検討
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("Ollama APIへのリクエストに失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("Ollama APIエラー: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	reader := bufio.NewReader(resp.Body)
	responseText, fullResponse, err := parseOllamaStreamResponse(reader)
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		log.Printf("Ollamaレスポンス解析エラー: %v", err)
		// エラー時でも部分的なレスポンスがあればログに出力
		if len(fullResponse) > 0 {
			lastLine := ""
			lines := strings.Split(strings.TrimSuffix(fullResponse, "\n"), "\n")
			if len(lines) > 0 {
				lastLine = lines[len(lines)-1]
			}
			log.Printf("Ollama API partial response before error: %s", lastLine)
		}
		// エラーメッセージを返す
		return "", elapsed, fmt.Errorf("Ollamaレスポンスの解析に失敗しました: %w", err)
	}

	// 成功時のログ（最後の行）
	lastLine := ""
	lines := strings.Split(strings.TrimSuffix(fullResponse, "\n"), "\n")
	if len(lines) > 0 {
		lastLine = lines[len(lines)-1]
	}
	log.Printf("Ollama API response (last line): %s", lastLine)
	log.Printf("Ollama full response text: %s", responseText) // 完全なテキストもログ出力

	// Ollamaの場合も履歴に追加する
	if responseText != "" {
		c.historyMgr.Add(userID, message, responseText)
		log.Printf("Added Ollama response to history for user %s", userID)
	} else {
		log.Printf("Skipping history add for user %s because Ollama responseText is empty.", userID)
	}

	return responseText, elapsed, nil
}

// parseOllamaStreamResponse は Ollama API からのストリーミングレスポンスを解析します。
// 各行がJSONオブジェクトであることを期待します。
func parseOllamaStreamResponse(reader *bufio.Reader) (string, string, error) {
	var responseTextBuilder strings.Builder
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n') // ReadBytes で区切り文字を含むバイトスライスを取得
		if err != nil {
			if err == io.EOF {
				// EOF の前に最後の行を処理する必要があるか確認
				trimmedLine := strings.TrimSpace(string(line))
				if trimmedLine != "" {
					fullResponseBuilder.Write(line) // EOF前の最後の行も記録
					var result map[string]interface{}
					if jsonErr := json.Unmarshal([]byte(trimmedLine), &result); jsonErr == nil {
						if responsePart, ok := result["response"].(string); ok {
							responseTextBuilder.WriteString(responsePart)
						}
						// EOF なので done フラグはチェック不要かもしれないが、念のため
						if done, ok := result["done"].(bool); ok && done {
							break
						}
					} else {
						log.Printf("最後の行のJSON解析に失敗（EOF）: %v, line: %s", jsonErr, trimmedLine)
					}
				}
				break // ストリーム終了
			}
			// EOF以外のエラー
			return responseTextBuilder.String(), fullResponseBuilder.String(), fmt.Errorf("Ollamaレスポンスの読み込みに失敗: %w", err)
		}

		fullResponseBuilder.Write(line) // 読み込んだ行全体を記録

		trimmedLine := strings.TrimSpace(string(line))
		if trimmedLine == "" {
			continue // 空行はスキップ
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(trimmedLine), &result); err != nil {
			// JSONとして解析できない行はログに出力してスキップする（例: Ollamaのバージョン情報など）
			log.Printf("Ollamaレスポンス行のJSON解析に失敗（スキップ）: %v, line: %s", err, trimmedLine)
			continue
		}

		// "response" フィールドがあれば連結
		if responsePart, ok := result["response"].(string); ok {
			responseTextBuilder.WriteString(responsePart)
		}

		// "done" フィールドが true なら終了
		if done, ok := result["done"].(bool); ok && done {
			break
		}
	}
	return responseTextBuilder.String(), fullResponseBuilder.String(), nil
}
