package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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
	GetResponse(userID, username, message, timestamp, prompt string) (string, float64, error)
	ClearHistory(userID string)
	Close()
}

type Chat struct {
	genaiClient        *genai.Client
	genaiModel         *genai.GenerativeModel
	defaultPrompt      string
	userHistories      map[string][]string
	userHistoriesMutex sync.Mutex
	modelCfg         *loader.ModelConfig
}

func NewChat(token string, model string, defaultPrompt string, modelCfg *loader.ModelConfig) (Service, error) {
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, err
	}

	genaiModel := client.GenerativeModel(model)

	return &Chat{
		genaiClient:   client,
		genaiModel:    genaiModel,
		defaultPrompt: defaultPrompt,
		userHistories: make(map[string][]string),
		modelCfg:         modelCfg,
	}, nil
}

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, error) {
	var history string
	if strings.ToLower(strings.TrimSpace(message)) == "/reset" {
		c.ClearHistory(userID)
		return "履歴をリセットしました！", 0, nil
	}

	c.userHistoriesMutex.Lock()
	history = strings.Join(c.userHistories[userID], "\n")
	c.userHistoriesMutex.Unlock()

	currentTime := time.Now().Format("2006-01-02 15:04:05")
	fullInput := prompt + "\n" + history + "\ndate time：" + currentTime + "\n" + message

	var responseText string
	var elapsed float64
	var err error

	if c.modelCfg.Ollama.Enabled {
		responseText, elapsed, err = c.getOllamaResponse(fullInput)
		if err != nil {
			log.Printf("Ollamaとの通信に失敗したため、Geminiにフォールバックします: %v", err)
			responseText, elapsed, err = c.getGeminiResponse(fullInput)
			if err != nil {
				return "", elapsed, fmt.Errorf("Gemini APIからのエラー: %v", err)
			}
		}
	} else {
		responseText, elapsed, err = c.getGeminiResponse(fullInput)
		if err != nil {
			return "", elapsed, fmt.Errorf("Gemini APIからのエラー: %v", err)
		}
	}

	c.userHistoriesMutex.Lock()
	c.userHistories[userID] = append(c.userHistories[userID], message, responseText)
	if len(c.userHistories[userID]) > c.modelCfg.MaxHistorySize {
		c.userHistories[userID] = c.userHistories[userID][len(c.userHistories[userID])-c.modelCfg.MaxHistorySize:]
	}
	c.userHistoriesMutex.Unlock()

	return responseText, elapsed, nil
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
	var responseText string
	var fullResponse string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", 0, fmt.Errorf("レスポンスの読み込みに失敗: %v", err)
		}
		fullResponse += line
		var result map[string]interface{}
		err = json.Unmarshal([]byte(line), &result)
		if err != nil {
			return "", 0, fmt.Errorf("レスポンスのJSON解析に失敗: %v", err)
		}

		response, ok := result["response"].(string)
		if ok {
			responseText += response
		}

		done, ok := result["done"].(bool)
		if ok && done {
			break
		}
	}

	elapsed := float64(time.Since(start).Milliseconds())
	lastLine := ""
	lines := strings.Split(strings.TrimSuffix(fullResponse, "\n"), "\n")
	if len(lines) > 0 {
		lastLine = lines[len(lines)-1]
	}
	log.Printf("Ollama API response: %s", lastLine)
	return responseText, elapsed, nil
}

func (c *Chat) ClearHistory(userID string) {
	c.userHistoriesMutex.Lock()
	defer c.userHistoriesMutex.Unlock()
	delete(c.userHistories, userID)
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
