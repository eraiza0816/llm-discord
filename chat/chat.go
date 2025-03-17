package chat

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Service interface {
	GetResponse(userID, username, message, timestamp, prompt string) (string, float64, error)
	ClearHistory(userID string)
	Close()
}

type Chat struct {
	genaiClient       *genai.Client
	genaiModel        *genai.GenerativeModel
	defaultPrompt     string
	userHistories     map[string][]string
	userHistoriesMutex sync.Mutex
}

func NewChat(token, model, defaultPrompt string) (Service, error) {
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
	}, nil
}

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, error) {
	c.userHistoriesMutex.Lock()
	defer c.userHistoriesMutex.Unlock()

	if strings.ToLower(strings.TrimSpace(message)) == "/reset" {
		c.ClearHistory(userID)
		return "履歴をリセットしました！", 0, nil
	}

	history := strings.Join(c.userHistories[userID], "\n")
	fullInput := prompt + "\n" + history + "\n" + message

	ctx := context.Background()
	start := time.Now()

	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		return "エラーが発生しました。", elapsed, err
	}

	responseText := getResponseText(resp)

	// 20件のメッセージ履歴を保持
	c.userHistories[userID] = append(c.userHistories[userID], message, responseText)
	if len(c.userHistories[userID]) > 20 {
		c.userHistories[userID] = c.userHistories[userID][len(c.userHistories[userID])-20:]
	}

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
