package chat

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/eraiza0816/llm-discord/history"
	"github.com/google/generative-ai-go/genai"
)

type mockHistoryManager struct {
	history.HistoryManager
	getFunc func(userID, threadID string) ([]history.HistoryMessage, error)
}

func (m *mockHistoryManager) Get(userID, threadID string) ([]history.HistoryMessage, error) {
	if m.getFunc != nil {
		return m.getFunc(userID, threadID)
	}
	return []history.HistoryMessage{}, nil
}

func TestBuildFullInput(t *testing.T) {
	t.Run("basic input with system prompt", func(t *testing.T) {
		result := buildFullInput("You are a bot.", "Hello", nil, "user1", "thread1", "2024-01-01T00:00:00Z")
		if !strings.Contains(result, "You are a bot.") {
			t.Error("Expected system prompt in output")
		}
		if !strings.Contains(result, "Hello") {
			t.Error("Expected user message in output")
		}
		if !strings.Contains(result, "User message:") {
			t.Error("Expected 'User message:' label in output")
		}
	})

	t.Run("input with history", func(t *testing.T) {
		mgr := &mockHistoryManager{
			getFunc: func(userID, threadID string) ([]history.HistoryMessage, error) {
				return []history.HistoryMessage{
					{Role: "user", Content: "previous question"},
					{Role: "model", Content: "previous answer"},
				}, nil
			},
		}
		result := buildFullInput("System prompt", "new message", mgr, "user1", "thread1", "2024-01-01T00:00:00Z")
		if !strings.Contains(result, "previous question") {
			t.Error("Expected history content in output")
		}
		if !strings.Contains(result, "assistant: previous answer") {
			t.Error("Expected 'model' role converted to 'assistant' in history")
		}
		if !strings.Contains(result, "Chat history:") {
			t.Error("Expected 'Chat history:' label in output")
		}
	})

	t.Run("input with empty history", func(t *testing.T) {
		mgr := &mockHistoryManager{
			getFunc: func(userID, threadID string) ([]history.HistoryMessage, error) {
				return []history.HistoryMessage{}, nil
			},
		}
		result := buildFullInput("System prompt", "new message", mgr, "user1", "thread1", "2024-01-01T00:00:00Z")
		if strings.Contains(result, "Chat history:") {
			t.Error("Expected no 'Chat history:' label for empty history")
		}
	})

	t.Run("history with error falls back gracefully", func(t *testing.T) {
		mgr := &mockHistoryManager{
			getFunc: func(userID, threadID string) ([]history.HistoryMessage, error) {
				return nil, errors.New("db error")
			},
		}
		result := buildFullInput("System prompt", "message", mgr, "user1", "thread1", "2024-01-01T00:00:00Z")
		if !strings.Contains(result, "message") {
			t.Error("Expected message even when history fetch fails")
		}
	})
}

func TestGetResponseText(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		result := getResponseText(nil)
		if result != "Gemini APIからの応答がありませんでした。" {
			t.Errorf("Expected default message for nil response, got %q", result)
		}
	})

	t.Run("empty candidates", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}
		result := getResponseText(resp)
		if result != "Gemini APIからの応答がありませんでした。" {
			t.Errorf("Expected default message for empty candidates, got %q", result)
		}
	})

	t.Run("candidate with nil content", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{Content: nil},
			},
		}
		result := getResponseText(resp)
		if result != "Gemini APIからの応答がありませんでした。" {
			t.Errorf("Expected default message for nil content, got %q", result)
		}
	})

	t.Run("candidate with text parts", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []genai.Part{
							genai.Text("Hello "),
							genai.Text("World"),
						},
					},
				},
			},
		}
		result := getResponseText(resp)
		if result != "Hello World" {
			t.Errorf("Expected 'Hello World', got %q", result)
		}
	})

	t.Run("candidate with non-text parts are ignored", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []genai.Part{
							genai.Text("Only text"),
							genai.Blob{MIMEType: "image/png", Data: []byte{1, 2, 3}},
						},
					},
				},
			},
		}
		result := getResponseText(resp)
		if result != "Only text" {
			t.Errorf("Expected 'Only text', got %q", result)
		}
	})
}

func TestParseOllamaStreamResponse(t *testing.T) {
	t.Run("single response line", func(t *testing.T) {
		input := `{"response":"Hello","done":false}
{"response":" World","done":true}
`
		reader := bufio.NewReader(strings.NewReader(input))
		text, full, err := parseOllamaStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "Hello World" {
			t.Errorf("Expected 'Hello World', got %q", text)
		}
		if !strings.Contains(full, "Hello") || !strings.Contains(full, "World") {
			t.Errorf("Full response should contain all lines")
		}
	})

	t.Run("empty response", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader(""))
		text, full, err := parseOllamaStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "" {
			t.Errorf("Expected empty text, got %q", text)
		}
		if full != "" {
			t.Errorf("Expected empty full, got %q", full)
		}
	})

	t.Run("invalid JSON lines are skipped", func(t *testing.T) {
		input := `{"response":"valid","done":false}
not json
{"response":"done","done":true}
`
		reader := bufio.NewReader(strings.NewReader(input))
		text, _, err := parseOllamaStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "validdone" {
			t.Errorf("Expected 'validdone', got %q", text)
		}
	})

	t.Run("done flag stops parsing", func(t *testing.T) {
		input := `{"response":"first","done":false}
{"response":"second","done":true}
{"response":"should not appear","done":false}
`
		reader := bufio.NewReader(strings.NewReader(input))
		text, _, err := parseOllamaStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "firstsecond" {
			t.Errorf("Expected 'firstsecond', got %q", text)
		}
	})
}

func TestParseOpenAIStreamResponse(t *testing.T) {
	t.Run("single chunk", func(t *testing.T) {
		input := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\" World\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n"
		reader := bufio.NewReader(strings.NewReader(input))
		text, err := parseOpenAIStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "Hello World" {
			t.Errorf("Expected 'Hello World', got %q", text)
		}
	})

	t.Run("empty response", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader(""))
		text, err := parseOpenAIStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "" {
			t.Errorf("Expected empty string, got %q", text)
		}
	})

	t.Run("DONE signal stops parsing", func(t *testing.T) {
		input := "data: {\"choices\":[{\"delta\":{\"content\":\"first\"},\"finish_reason\":null}]}\n\ndata: [DONE]\ndata: {\"choices\":[{\"delta\":{\"content\":\"ignored\"},\"finish_reason\":null}]}\n"
		reader := bufio.NewReader(strings.NewReader(input))
		text, err := parseOpenAIStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "first" {
			t.Errorf("Expected 'first', got %q", text)
		}
	})

	t.Run("non-data lines are skipped", func(t *testing.T) {
		input := ": heartbeat\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"content\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n"
		reader := bufio.NewReader(strings.NewReader(input))
		text, err := parseOpenAIStreamResponse(reader)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if text != "content" {
			t.Errorf("Expected 'content', got %q", text)
		}
	})
}
