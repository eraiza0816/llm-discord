package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Service interface {
	GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error)
	Close()
}

type Chat struct {
	genaiClient    *genai.Client
	genaiModel     *genai.GenerativeModel
	weatherService WeatherService
	defaultPrompt  string
	historyMgr     history.HistoryManager
	modelCfg       *loader.ModelConfig
	tools          []*genai.Tool
}

func NewChat(token string, model string, defaultPrompt string, modelCfg *loader.ModelConfig, historyMgr history.HistoryManager) (Service, error) {
	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	genaiModel := genaiClient.GenerativeModel(model)

	weatherService := NewWeatherService()

	weatherFuncDeclarations := weatherService.GetFunctionDeclarations()

	// 他ツール追加箇所
	// 例: otherTools := []*genai.Tool{...}
	// allFuncDeclarations := append(weatherFuncDeclarations, otherTools...)

	// 現在は天気ツールのみ
	tools := []*genai.Tool{
		{
			FunctionDeclarations: weatherFuncDeclarations,
		},
	}

	genaiModel.Tools = tools

	return &Chat{
		genaiClient:    genaiClient,
		genaiModel:     genaiModel,
		weatherService: weatherService, // 初期化済み WeatherService を設定
		defaultPrompt:  defaultPrompt,
		historyMgr:     historyMgr,
		modelCfg:       modelCfg,
		tools:          tools,
	}, nil
}

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error) {
	fullInput := buildFullInput(prompt, message, c.historyMgr, userID)

	ctx := context.Background()
	start := time.Now()
	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())
	if err != nil {
		return "", elapsed, "", fmt.Errorf("Gemini APIからのエラー: %w", err)
	}

	if resp.Candidates != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			// log.Printf("Gemini response parts: %+v", candidate.Content.Parts) // デバッグ用ログコメントアウト

			var functionCallProcessed bool
			var llmIntroText strings.Builder
			var toolResult string
			var toolErr error

			for i, part := range candidate.Content.Parts {
				// log.Printf("Processing part %d", i) // デバッグ用ログコメントアウト
				switch v := part.(type) {
				case genai.Text:
					// log.Printf("Part %d is genai.Text: %s", i, string(v)) // デバッグ用ログコメントアウト
					llmIntroText.WriteString(string(v))
				case genai.FunctionCall:
					// log.Printf("Part %d IS a genai.FunctionCall!", i) // デバッグ用ログコメントアウト
					fn := v
					// log.Printf("Function Call triggered: %s, Args: %v", fn.Name, fn.Args) // デバッグ用ログコメントアウト
					functionCallProcessed = true

					toolResult, toolErr = c.weatherService.HandleFunctionCall(fn)
					if toolErr != nil {
						log.Printf("Error handling function call %s via WeatherService: %v", fn.Name, toolErr) // エラーログは残す
						toolResult = fmt.Sprintf("関数の処理中にエラーが発生しました: %v", toolErr)
						toolErr = nil // エラーは処理済みとしてnilにする
					}
				default:
					log.Printf("Part %d is an unexpected type: %T", i, v) // 未知の型はログに残す
				}
			}

			if functionCallProcessed {
				finalResponse := llmIntroText.String()
				if finalResponse != "" && toolResult != "" {
					if !strings.HasSuffix(finalResponse, "\n") {
						finalResponse += "\n"
					}
				}
				finalResponse += toolResult

				// log.Printf("Combined LLM intro and tool result: %s", finalResponse) // デバッグ用ログコメントアウト

				if finalResponse != "" {
					c.historyMgr.Add(userID, message, finalResponse)
					// log.Printf("Added combined response to history for user %s", userID) // デバッグ用ログコメントアウト
				} else {
					log.Printf("Skipping history add for user %s because combined response is empty.", userID) // これは残す
				}
				return finalResponse, 0, "zutool", nil
			}

			// log.Println("No FunctionCall was processed in response parts.") // デバッグ用ログコメントアウト
			responseText := llmIntroText.String()
			if responseText == "" {
				responseText = getResponseText(resp)
				// log.Printf("Falling back to getResponseText: %s", responseText) // デバッグ用ログコメントアウト
			}

			// log.Printf("Final response text (no function call): %s", responseText) // デバッグ用ログコメントアウト
			if responseText != "" {
				c.historyMgr.Add(userID, message, responseText)
				// log.Printf("Added normal text response to history for user %s", userID) // デバッグ用ログコメントアウト
			} else {
				log.Printf("Skipping history add for user %s because responseText is empty.", userID) // これは残す
			}
			return responseText, elapsed, c.modelCfg.ModelName, nil

		} else {
			log.Println("Gemini response candidate content or parts are empty.") // これは残す
		}
	} else {
		log.Println("Gemini response candidates are empty.") // これは残す
	}

	log.Println("No valid candidates found in Gemini response.") // これは残す
	responseText := "すみません、応答を取得できませんでした。"
	return responseText, elapsed, c.modelCfg.ModelName, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}
