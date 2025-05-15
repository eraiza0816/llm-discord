package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var errorLogger *log.Logger

type Service interface {
	GetResponse(userID, threadID, username, message, timestamp, prompt string) (string, float64, string, error)
	Close()
}

type Chat struct {
	genaiClient    *genai.Client
	genaiModel     *genai.GenerativeModel
	weatherService WeatherService
	defaultPrompt  string
	historyMgr     history.HistoryManager
	tools          []*genai.Tool
}

func NewChat(token string, historyMgr history.HistoryManager) (Service, error) {
	errorLogFile, err := os.OpenFile("log/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open error log file: %v", err)
		errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		errorLogger = log.New(errorLogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	initialModelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		errorLogger.Printf("初期 model.json の読み込みに失敗しました: %v", err)
		return nil, fmt.Errorf("初期 model.json の読み込みに失敗しました: %w", err)
	}
	initialGeminiModelName := initialModelCfg.ModelName

	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	genaiModel := genaiClient.GenerativeModel(initialGeminiModelName)

	weatherService := NewWeatherService()

	weatherFuncDeclarations := weatherService.GetFunctionDeclarations()

	tools := []*genai.Tool{
		{
			FunctionDeclarations: weatherFuncDeclarations,
		},
	}

	genaiModel.Tools = tools

	return &Chat{
		genaiClient:    genaiClient,
		genaiModel:     genaiModel,
		weatherService: weatherService,
		historyMgr:     historyMgr,
		tools:          tools,
	}, nil
}

func (c *Chat) GetResponse(userID, threadID, username, message, timestamp, prompt string) (string, float64, string, error) {
	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		errorLogger.Printf("Error loading model config in GetResponse: %v", err)
		return "", 0, "", fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// buildFullInput に threadID を渡すように変更 (prompt.go の修正も必要)
	fullInput := buildFullInput(prompt, message, c.historyMgr, userID, threadID)

	if modelCfg.Ollama.Enabled {
		log.Printf("Using Ollama (%s) for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
		// getOllamaResponse に threadID を渡すように変更 (ollama.go の修正も必要)
		responseText, elapsed, err := c.getOllamaResponse(userID, threadID, message, fullInput, modelCfg.Ollama)
		if err != nil {
			errorLogger.Printf("Ollama API call failed for user %s in thread %s: %v", userID, threadID, err)
			return "", elapsed, modelCfg.Ollama.ModelName, fmt.Errorf("Ollama APIからのエラー: %w", err)
		}
		return responseText, elapsed, modelCfg.Ollama.ModelName, nil
	}

	log.Printf("Using Gemini (%s) for user %s", modelCfg.ModelName, userID)
	c.genaiModel = c.genaiClient.GenerativeModel(modelCfg.ModelName)
	c.genaiModel.Tools = c.tools

	ctx := context.Background()
	start := time.Now()
	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		errorLogger.Printf("Initial Gemini API call failed for model %s: %v", modelCfg.ModelName, err)

		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == 429 {
			log.Printf("Quota exceeded for model %s. Attempting fallback...", modelCfg.ModelName)

			if modelCfg.SecondaryModelName != "" {
				log.Printf("Attempting retry with secondary model: %s", modelCfg.SecondaryModelName)
				secondaryModel := c.genaiClient.GenerativeModel(modelCfg.SecondaryModelName)
				secondaryModel.Tools = c.tools

				startSecondary := time.Now()
				resp, err = secondaryModel.GenerateContent(ctx, genai.Text(fullInput))
				elapsed = float64(time.Since(startSecondary).Milliseconds())

				if err == nil {
					log.Printf("Successfully generated content with secondary model: %s", modelCfg.SecondaryModelName)
					modelCfg.ModelName = modelCfg.SecondaryModelName
					goto HandleResponse
				}
				errorLogger.Printf("Secondary Gemini API call failed for model %s: %v", modelCfg.SecondaryModelName, err)
			} else {
				log.Println("Secondary model name not configured.")
			}

			if modelCfg.Ollama.Enabled {
				log.Printf("Falling back to Ollama (%s) for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
				// getOllamaResponse に threadID を渡すように変更 (ollama.go の修正も必要)
				responseText, ollamaElapsed, ollamaErr := c.getOllamaResponse(userID, threadID, message, fullInput, modelCfg.Ollama)
				if ollamaErr != nil {
					errorLogger.Printf("Ollama fallback failed for user %s in thread %s: %v", userID, threadID, ollamaErr)
					return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIクォータ超過後、Ollamaフォールバックも失敗: (Gemini: %w), (Ollama: %v)", err, ollamaErr)
				}
				log.Printf("Successfully generated content with Ollama fallback: %s for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
				return responseText, ollamaElapsed, modelCfg.Ollama.ModelName, nil
			} else {
				log.Println("Ollama is not enabled, cannot fallback.")
			}

			return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIクォータ超過、フォールバック先なし: %w", err)

		} else {
			errorLogger.Printf("Gemini API error for model %s: %v", modelCfg.ModelName, err)
			return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIからのエラー: %w", err)
		}
	}

HandleResponse:
	if resp.Candidates != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			var functionCallProcessed bool
			var llmIntroText strings.Builder
			var toolResult string
			var toolErr error

			for i, part := range candidate.Content.Parts {
				switch v := part.(type) {
				case genai.Text:
					llmIntroText.WriteString(string(v))
				case genai.FunctionCall:
					fn := v
					functionCallProcessed = true

					toolResult, toolErr = c.weatherService.HandleFunctionCall(fn)
					if toolErr != nil {
						errorLogger.Printf("Error handling function call %s via WeatherService: %v", fn.Name, toolErr)
						toolResult = fmt.Sprintf("関数の処理中にエラーが発生しました: %v", toolErr)
						toolErr = nil
					}
				default:
					errorLogger.Printf("Part %d is an unexpected type: %T", i, v)
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

				if finalResponse != "" {
					c.historyMgr.Add(userID, threadID, message, finalResponse)
				} else {
					errorLogger.Printf("Skipping history add for user %s in thread %s because combined response is empty.", userID, threadID)
				}
				return finalResponse, 0, "zutool", nil
			}

			responseText := llmIntroText.String()
			if responseText == "" {
				responseText = getResponseText(resp)
			}

			if responseText != "" {
				c.historyMgr.Add(userID, threadID, message, responseText)
		} else {
			errorLogger.Printf("Skipping history add for user %s in thread %s because responseText is empty.", userID, threadID)
		}
		return responseText, elapsed, modelCfg.ModelName, nil

	} else {
		errorLogger.Println("Gemini response candidate content or parts are empty.")
		}
	} else {
		errorLogger.Println("Gemini response candidates are empty.")
	}

	errorLogger.Println("No valid candidates found in Gemini response.")
	responseText := "すみません、応答を取得できませんでした。"
	return responseText, elapsed, modelCfg.ModelName, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}

func GetErrorLogger() *log.Logger {
	return errorLogger
}
