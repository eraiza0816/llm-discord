package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var errorLogger *log.Logger

type Service interface {
	GetResponse(userID, threadID, username, message, timestamp, prompt string, isBot bool) (string, float64, string, error)
	Close()
}

type Chat struct {
	genaiClient      *genai.Client
	genaiModel       *genai.GenerativeModel
	weatherService   WeatherService
	urlReaderService URLReaderService
	defaultPrompt    string
	historyMgr       history.HistoryManager
	tools            []*genai.Tool
	modelConfig      *loader.ModelConfig
	config           *config.Config
}

func NewChat(cfg *config.Config, historyMgr history.HistoryManager) (Service, error) {
	errorLogFile, err := os.OpenFile("log/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open error log file: %v", err)
		errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		errorLogger = log.New(errorLogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	initialModelCfg := cfg.Model
	initialGeminiModelName := initialModelCfg.ModelName

	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	genaiModel := genaiClient.GenerativeModel(initialGeminiModelName)

	weatherService := NewWeatherService()
	urlReaderService := NewURLReaderService()

	weatherFuncDeclarations := weatherService.GetFunctionDeclarations()
	urlReaderFuncDeclaration := urlReaderService.GetURLReaderFunctionDeclaration()

	var allDeclarations []*genai.FunctionDeclaration
	// weatherFuncDeclarations が nil でなく、要素を持つ場合のみ追加
	if len(weatherFuncDeclarations) > 0 {
		allDeclarations = append(allDeclarations, weatherFuncDeclarations...)
	}
	if urlReaderFuncDeclaration != nil {
		allDeclarations = append(allDeclarations, urlReaderFuncDeclaration)
	}

	var tools []*genai.Tool
	if len(allDeclarations) > 0 {
		tools = []*genai.Tool{
			{
				FunctionDeclarations: allDeclarations,
			},
		}
	}

	genaiModel.Tools = tools

	return &Chat{
		genaiClient:      genaiClient,
		genaiModel:       genaiModel,
		weatherService:   weatherService,
		urlReaderService: urlReaderService,
		historyMgr:       historyMgr,
		tools:            tools,
		modelConfig:      initialModelCfg,
		config:           cfg,
	}, nil
}

func (c *Chat) GetResponse(userID, threadID, username, message, timestamp, defaultSystemPrompt string, isBot bool) (string, float64, string, error) {
	if isBot {
		// Botとの会話履歴を取得
		count, err := c.historyMgr.GetBotConversationCount(threadID, userID)
		if err != nil {
			errorLogger.Printf("Failed to get bot conversation count: %v", err)
			// エラーが発生しても処理は続行するが、ログには残す
		}

		// 3回以上会話している場合は応答しない
		if count >= 3 {
			log.Printf("Botとの会話が3回に達したため、応答を中断します。UserID: %s, ThreadID: %s", userID, threadID)
			return "", 0, "", nil
		}
	}

	modelCfg := c.modelConfig
	if isBot {
		// Botの場合はOllamaを強制的に使用する
		log.Printf("Botとの対話のため、Ollamaモデルを強制的に使用します。UserID: %s", userID)
		modelCfg.Ollama.Enabled = true
	}

	currentSystemPrompt := modelCfg.GetPromptByUser(username)
	// log.Printf("System prompt for user %s: %s", username, currentSystemPrompt)


	// prompt.go の buildFullInput を使用して、LLMへの完全な入力文字列を構築
	fullInput := buildFullInput(currentSystemPrompt, message, c.historyMgr, userID, threadID, timestamp)

	if modelCfg.Ollama.Enabled {
		log.Printf("Using Ollama (%s) for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
		// ollama.go の getOllamaResponse を使用してOllamaからの応答を取得
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
					goto HandleResponse
				}
				errorLogger.Printf("Secondary Gemini API call failed for model %s: %v", modelCfg.SecondaryModelName, err)
			} else {
				log.Println("Secondary model name not configured.")
			}

			if modelCfg.Ollama.Enabled {
				log.Printf("Falling back to Ollama (%s) for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
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
			logMessage := fmt.Sprintf("Gemini API error for model %s. Input: %s, Error: %v", modelCfg.ModelName, fullInput, err)
			if gapiErr, ok := err.(*googleapi.Error); ok {
				logMessage += fmt.Sprintf(", API Error Body: %s, API Error Headers: %v", gapiErr.Body, gapiErr.Header)
			}
			errorLogger.Printf(logMessage)
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

					if fn.Name == "get_url_content" {
						url, ok := fn.Args["url"].(string)
						if !ok {
							toolErr = fmt.Errorf("get_url_content の引数 'url' がstring型ではありません")
							errorLogger.Printf("Invalid argument type for get_url_content: %v", fn.Args["url"])
							toolResult = fmt.Sprintf("関数の引数エラー: %v", toolErr)
						} else {
							toolResult, toolErr = c.urlReaderService.GetURLContentAsText(url)
						}
					} else {
						// 既存の天候情報関数の処理
						toolResult, toolErr = c.weatherService.HandleFunctionCall(fn)
					}

					if toolErr != nil {
						errorLogger.Printf("Error handling function call %s: %v", fn.Name, toolErr)
						toolResult = fmt.Sprintf("関数の処理中にエラーが発生しました: %v", toolErr)
						toolErr = nil
					}
				default:
					errorLogger.Printf("Part %d is an unexpected type: %T", i, v)
				}
			}

			if functionCallProcessed {
				// Function Calling が実行された場合、その結果 (toolResult) を用いて
				// LLMに再度問い合わせを行い、最終的な自然言語の応答を生成する。

				// 呼び出された関数名を取得
				var calledFuncName string
				for _, part := range candidate.Content.Parts {
					if fc, ok := part.(genai.FunctionCall); ok {
						calledFuncName = fc.Name
						break
					}
				}

				if calledFuncName == "" {
					errorLogger.Printf("Could not determine called function name from candidate parts.")
					return "関数呼び出し名の取得に失敗しました。", elapsed, modelCfg.ModelName, fmt.Errorf("関数呼び出し名の取得に失敗")
				}

				// 次のLLM呼び出しのためのpartsを構築する。
				// 構成:
				// 1. ユーザーの最新のメッセージ
				// 2. LLMの最初の応答から FunctionCall の部分のみ
				// 3. ツールの実行結果 (genai.FunctionResponse)
				var partsForNextTurn []genai.Part
				partsForNextTurn = append(partsForNextTurn, genai.Text(message)) // ユーザーの最新メッセージ

				var functionCallPart genai.FunctionCall
				for _, part := range candidate.Content.Parts {
					if fc, ok := part.(genai.FunctionCall); ok {
						functionCallPart = fc
						break
					}
				}

				if functionCallPart.Name == "" {
					errorLogger.Printf("Could not extract FunctionCall from candidate.Content.Parts for second call. Using full candidate.Content.Parts.")
					// FunctionCallが見つからなかった場合は、元のpartsForNextTurnのロジックにフォールバック（あるいはエラー）
					// ここでは元のロジック（candidate.Content.Parts全体）を使う
					partsForNextTurn = append(partsForNextTurn, candidate.Content.Parts...)
				} else {
					// LLMの最初の応答に含まれる導入テキスト（もしあれば）も追加した方が自然かもしれない。
					// 一旦、FunctionCallのみに絞ってみる。
					// 導入テキストも必要な場合は、candidate.Content.Parts全体を使うか、Text部分も抽出する。
					// llmIntroText を使う手もある。
					if llmIntroText.Len() > 0 {
						partsForNextTurn = append(partsForNextTurn, genai.Text(llmIntroText.String()))
					}
					partsForNextTurn = append(partsForNextTurn, functionCallPart)
				}


				// LLMに渡すtoolResultの長さを制限
				const maxToolResultForLLM = 1800
				toolResultForLLM := toolResult
				if len(toolResultForLLM) > maxToolResultForLLM {
					toolResultForLLM = toolResultForLLM[:maxToolResultForLLM] + "..."
				}

				// ツールの実行結果をFunctionResponseとして追加
				partsForNextTurn = append(partsForNextTurn, genai.FunctionResponse{
					Name: calledFuncName,
					Response: map[string]interface{}{
						"content": toolResultForLLM,
					},
				})

				// Function Callingの結果を踏まえて、再度LLMにコンテンツ生成を要求
				secondResp, err := c.genaiModel.GenerateContent(ctx, partsForNextTurn...)
				elapsed += float64(time.Since(start).Milliseconds()) // 時間を加算

				if err != nil {
					errorLogger.Printf("Error in second GenerateContent call after function execution: %v", err)
					return fmt.Sprintf("ツールの実行結果: %s (LLMによる最終応答生成に失敗: %v)", toolResult, err), elapsed, modelCfg.ModelName, nil
				}

				finalResponseText := getResponseText(secondResp)
				if finalResponseText == "" {
					finalResponseText = "ツールは実行されましたが、LLMからの追加の応答はありませんでした。"
					if toolResult != "" {
						finalResponseText += fmt.Sprintf(" ツールの結果: %s", toolResult)
					}
				}

				// (オプション) LLMからの導入テキストと最終応答を結合することも可能だが、現在はしていない。

				// 最終応答があれば履歴に追加
				if finalResponseText != "" {
					c.historyMgr.Add(userID, threadID, message, finalResponseText)
				} else {
					errorLogger.Printf("Skipping history add for user %s in thread %s because finalResponseText is empty after function call.", userID, threadID)
				}
				return finalResponseText, elapsed, modelCfg.ModelName, nil
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
	responseText := "応答を取得できませんでした。"
	return responseText, elapsed, modelCfg.ModelName, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}

func GetErrorLogger() *log.Logger {
	return errorLogger
}
