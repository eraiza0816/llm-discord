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
	"github.com/eraiza0816/llm-discord/loader"
)

func (c *Chat) getOllamaResponse(userID, threadID, message, fullInput string, ollamaCfg loader.OllamaConfig) (string, float64, error) {

	start := time.Now()
	url := ollamaCfg.APIEndpoint
	modelName := ollamaCfg.ModelName
	if url == "" || modelName == "" {
		return "", 0, fmt.Errorf("Ollama APIエンドポイントまたはモデル名が設定されていません")
	}

	payload := map[string]string{
		"prompt": fullInput,
		"model":  modelName,
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
		if len(fullResponse) > 0 {
			lastLine := ""
			lines := strings.Split(strings.TrimSuffix(fullResponse, "\n"), "\n")
			if len(lines) > 0 {
				lastLine = lines[len(lines)-1]
			}
			log.Printf("Ollama API partial response before error: %s", lastLine)
		}
		return "", elapsed, fmt.Errorf("Ollamaレスポンスの解析に失敗しました: %w", err)
	}

	lastLine := ""
	lines := strings.Split(strings.TrimSuffix(fullResponse, "\n"), "\n")
	if len(lines) > 0 {
		lastLine = lines[len(lines)-1]
	}
	log.Printf("Ollama API response (last line): %s", lastLine)
	log.Printf("Ollama full response text: %s", responseText)

	if responseText != "" {
		c.historyMgr.Add(userID, threadID, message, responseText)
		log.Printf("Added Ollama response to history for user %s in thread %s", userID, threadID)
	} else {
		log.Printf("Skipping history add for user %s in thread %s because Ollama responseText is empty.", userID, threadID)
	}

	return responseText, elapsed, nil
}

func parseOllamaStreamResponse(reader *bufio.Reader) (string, string, error) {
	var responseTextBuilder strings.Builder
	var fullResponseBuilder strings.Builder
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				trimmedLine := strings.TrimSpace(string(line))
				if trimmedLine != "" {
					fullResponseBuilder.Write(line)
					var result map[string]interface{}
					if jsonErr := json.Unmarshal([]byte(trimmedLine), &result); jsonErr == nil {
						if responsePart, ok := result["response"].(string); ok {
							responseTextBuilder.WriteString(responsePart)
						}
						if done, ok := result["done"].(bool); ok && done {
							break
						}
					} else {
						log.Printf("最後の行のJSON解析に失敗（EOF）: %v, line: %s", jsonErr, trimmedLine)
					}
				}
				break
			}
			return responseTextBuilder.String(), fullResponseBuilder.String(), fmt.Errorf("Ollamaレスポンスの読み込みに失敗: %w", err)
		}

		fullResponseBuilder.Write(line)

		trimmedLine := strings.TrimSpace(string(line))
		if trimmedLine == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(trimmedLine), &result); err != nil {
			log.Printf("Ollamaレスポンス行のJSON解析に失敗: %v, line: %s", err, trimmedLine)
			continue
		}

		if responsePart, ok := result["response"].(string); ok {
			responseTextBuilder.WriteString(responsePart)
		}

		if done, ok := result["done"].(bool); ok && done {
			break
		}
	}
	return responseTextBuilder.String(), fullResponseBuilder.String(), nil
}
