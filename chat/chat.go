package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"net/http"
	"bytes"
	"encoding/json"
	"log"
	"bufio"
	"io"
)

type Service interface {
	GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error)
	Close()
}

type Chat struct {
	genaiClient   *genai.Client
	genaiModel    *genai.GenerativeModel
	defaultPrompt string
	historyMgr    history.HistoryManager
	modelCfg      *loader.ModelConfig
}

func NewChat(token string, model string, defaultPrompt string, modelCfg *loader.ModelConfig, historyMgr history.HistoryManager) (Service, error) {
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, err
	}

	genaiModel := client.GenerativeModel(model)

	return &Chat{
		genaiClient:   client,
		genaiModel:    genaiModel,
		defaultPrompt: defaultPrompt,
		historyMgr:    historyMgr,
		modelCfg:      modelCfg,
	}, nil
}

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error) {

	if strings.ToLower(strings.TrimSpace(message)) == "/reset" {
		c.historyMgr.Clear(userID)
		return "履歴をリセットしました！", 0, "", nil
	}

	historyStr := c.historyMgr.Get(userID)

	currentTime := time.Now().Format("2006-01-02 15:04:05")
	fullInput := prompt + "\n" + historyStr + "\ndate time：" + currentTime + "\n" + message

	var responseText string
	var elapsed float64
	var err error

	if c.modelCfg.Ollama.Enabled {
		responseText, elapsed, err = c.getOllamaResponse(fullInput)
		if err != nil {
			log.Printf("Ollamaとの通信に失敗したため、Geminiにフォールバックします: %v", err)
			responseText, elapsed, err = c.getGeminiResponse(fullInput)
			if err != nil {
				return "", elapsed, "", fmt.Errorf("Gemini APIからのエラー: %v", err)
			}
		}
	} else {
		responseText, elapsed, err = c.getGeminiResponse(fullInput)
		if err != nil {
			return "", elapsed, "", fmt.Errorf("Gemini APIからのエラー: %v", err)
		}
	}

	c.historyMgr.Add(userID, message, responseText)

	modelName := c.modelCfg.ModelName
	if c.modelCfg.Ollama.Enabled {
		modelName = c.modelCfg.Ollama.ModelName
	}

	return responseText, elapsed, modelName, nil
}

func (c *Chat) getGeminiResponse(fullInput string) (string, float64, error) {
	ctx := context.Background()
	start := time.Now()

	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		return "", elapsed, fmt.Errorf("Gemini APIからのエラー: %v", err)
	}

	responseText := getResponseText(resp)
	return responseText, elapsed, nil
}

func (c *Chat) getOllamaResponse(fullInput string) (string, float64, error) {
	start := time.Now()
	url := c.modelCfg.Ollama.APIEndpoint

	payload := map[string]string{
		"prompt": fullInput,
		"model":  c.modelCfg.Ollama.ModelName,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("JSONの作成に失敗: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", 0, fmt.Errorf("リクエストの作成に失敗: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("Ollama APIへのリクエストに失敗: %v", err)
	}
	defer resp.Body.Close()

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
	log.Printf("Ollama API response: %s", lastLine)

	return responseText, elapsed, nil
}

func parseOllamaStreamResponse(reader *bufio.Reader) (string, string, error) {
	var responseText string
	var fullResponse string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fullResponse, fmt.Errorf("レスポンスの読み込みに失敗: %w", err)
		}
		fullResponse += line

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(trimmedLine), &result); err != nil {
			log.Printf("レスポンス行のJSON解析に失敗（処理は続行）: %v, line: %s", err, trimmedLine)
			continue

		responsePart, ok := result["response"].(string)
		if ok {
			responseText += responsePart
		}
	}


		done, ok := result["done"].(bool)
		if ok && done {
			break
		}
	}
	// ループ終了後、最終的なテキストと全レスポンス、エラーなしを返す
	return responseText, fullResponse, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}

func getResponseText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "Gemini APIからの応答がありませんでした。"
	}

	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			responseText += string(text)
		}
	}
	return responseText
}
