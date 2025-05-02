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
	"google.golang.org/api/googleapi" // googleapi エラーを判定するためにインポート
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
	// modelCfg       *loader.ModelConfig // 削除
	tools          []*genai.Tool
	// defaultPrompt string // 削除
}

// model, defaultPrompt, modelCfg を引数から削除
func NewChat(token string, historyMgr history.HistoryManager) (Service, error) {
	// NewChat 時点で model.json を一時的に読み込み、初期 Gemini モデル名を取得
	initialModelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		// 起動時に設定ファイルが読めないのは致命的なのでエラーにする
		return nil, fmt.Errorf("初期 model.json の読み込みに失敗しました: %w", err)
	}
	// Ollama が有効な場合でも、Gemini クライアントとモデルの初期化は行っておく
	// GetResponse でモデル名は毎回上書きされる想定
	initialGeminiModelName := initialModelCfg.ModelName

	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	// 初期モデル名で genaiModel を初期化
	genaiModel := genaiClient.GenerativeModel(initialGeminiModelName)

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

	genaiModel.Tools = tools // tools は初期設定を保持

	return &Chat{
		genaiClient:    genaiClient,
		genaiModel:     genaiModel, // 初期モデルで初期化
		weatherService: weatherService,
		historyMgr:     historyMgr,
		// defaultPrompt と modelCfg は削除
		tools:          tools,
	}, nil
}

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error) {
	// GetResponse が呼ばれるたびに model.json を読み込む
	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		// エラー時はデフォルトのモデル名（空文字）を返し、エラーをラップする
		log.Printf("Error loading model config in GetResponse: %v", err)
		return "", 0, "", fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// userPrompt (引数 prompt) はカスタムプロンプト or デフォルトプロンプトが渡ってくる想定
	// buildFullInput は渡された prompt を使うので、c.defaultPrompt は不要
	fullInput := buildFullInput(prompt, message, c.historyMgr, userID)

	// Ollamaが有効かチェック (読み込んだ modelCfg を使用)
	if modelCfg.Ollama.Enabled {
		log.Printf("Using Ollama (%s) for user %s", modelCfg.Ollama.ModelName, userID)
		// getOllamaResponse に modelCfg.Ollama の情報を渡すように変更 (ollama.go 側の修正も必要)
		responseText, elapsed, err := c.getOllamaResponse(userID, message, fullInput, modelCfg.Ollama) // modelCfg.Ollama を渡す
		if err != nil {
			// エラー時は Ollama のモデル名を返す (読み込んだ modelCfg から)
			return "", elapsed, modelCfg.Ollama.ModelName, fmt.Errorf("Ollama APIからのエラー: %w", err)
		}
		// 成功時は Ollama のモデル名を返す (読み込んだ modelCfg から)
		return responseText, elapsed, modelCfg.Ollama.ModelName, nil
	}

	// --- 以下、Gemini 処理 ---
	log.Printf("Using Gemini (%s) for user %s", modelCfg.ModelName, userID)
	// 読み込んだ設定に基づき Gemini モデルを更新 (モデル名が変わった場合に対応)
	// Note: genai.Client はスレッドセーフだが、GenerativeModel の設定変更が安全かは要確認。
	//       毎回 GenerativeModel を生成する方が安全かもしれないが、パフォーマンス影響を考慮。
	//       ここでは既存の genaiModel の設定を上書きする方針を試す。
	//       ただし、Tools など他の設定も毎回上書きする必要があるか注意。ToolsはNewChatで設定済み。
	c.genaiModel = c.genaiClient.GenerativeModel(modelCfg.ModelName) // モデル名を更新
	c.genaiModel.Tools = c.tools // Tools は NewChat で設定したものを再設定

	ctx := context.Background()
	start := time.Now()
	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())

	// --- エラーハンドリング開始 ---
	if err != nil {
		log.Printf("Initial Gemini API call failed for model %s: %v", modelCfg.ModelName, err)

		// 429 エラーかどうかを判定
		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == 429 {
			log.Printf("Quota exceeded for model %s. Attempting fallback...", modelCfg.ModelName)

			// 1. Secondary Model で再試行
			if modelCfg.SecondaryModelName != "" {
				log.Printf("Attempting retry with secondary model: %s", modelCfg.SecondaryModelName)
				secondaryModel := c.genaiClient.GenerativeModel(modelCfg.SecondaryModelName)
				secondaryModel.Tools = c.tools // Tools を設定

				startSecondary := time.Now()
				resp, err = secondaryModel.GenerateContent(ctx, genai.Text(fullInput))
				elapsed = float64(time.Since(startSecondary).Milliseconds()) // elapsed を更新

				if err == nil {
					// Secondary Model で成功した場合、以降の処理に進む (モデル名は Secondary を使う)
					log.Printf("Successfully generated content with secondary model: %s", modelCfg.SecondaryModelName)
					// モデル名を Secondary に差し替えて処理を続行
					modelCfg.ModelName = modelCfg.SecondaryModelName // 以降の処理で使うモデル名を更新
					goto HandleResponse // エラーがなかったのでレスポンス処理へジャンプ
				}
				// Secondary Model でもエラーが発生した場合
				log.Printf("Secondary Gemini API call failed for model %s: %v", modelCfg.SecondaryModelName, err)
				// Secondary での 429 エラーも考慮するなら、ここで再度チェックするが、一旦 Ollama フォールバックへ
			} else {
				log.Println("Secondary model name not configured.")
			}

			// 2. Ollama にフォールバック
			if modelCfg.Ollama.Enabled {
				log.Printf("Falling back to Ollama (%s)", modelCfg.Ollama.ModelName)
				// getOllamaResponse を呼び出す
				responseText, ollamaElapsed, ollamaErr := c.getOllamaResponse(userID, message, fullInput, modelCfg.Ollama)
				if ollamaErr != nil {
					// Ollama も失敗した場合、元の 429 エラーと Ollama エラーを合わせて返すか検討
					// ここでは元の 429 エラーを返すことにする
					log.Printf("Ollama fallback failed: %v", ollamaErr)
					return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIクォータ超過後、Ollamaフォールバックも失敗: (Gemini: %w), (Ollama: %v)", err, ollamaErr) // 元の Gemini エラーを返す
				}
				// Ollama で成功した場合
				log.Printf("Successfully generated content with Ollama fallback: %s", modelCfg.Ollama.ModelName)
				return responseText, ollamaElapsed, modelCfg.Ollama.ModelName, nil
			} else {
				log.Println("Ollama is not enabled, cannot fallback.")
			}

			// Secondary Model もなく、Ollama も無効な場合は、元の 429 エラーを返す
			return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIクォータ超過、フォールバック先なし: %w", err)

		} else {
			// 429 以外の Gemini API エラーの場合
			return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIからのエラー: %w", err)
		}
	}
	// --- エラーハンドリング終了 ---

HandleResponse: // Secondary Model で成功した場合のジャンプ先
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
			// log.Printf("Added normal text response to history for user %s", userID)
		} else {
			log.Printf("Skipping history add for user %s because responseText is empty.", userID)
		}
		// Gemini のモデル名を返す (読み込んだ modelCfg から)
		return responseText, elapsed, modelCfg.ModelName, nil

	} else {
		log.Println("Gemini response candidate content or parts are empty.")
		}
	} else {
		log.Println("Gemini response candidates are empty.")
	}

	log.Println("No valid candidates found in Gemini response.")
	responseText := "すみません、応答を取得できませんでした。"
	// エラー時もモデル名を返す (読み込んだ modelCfg から)
	return responseText, elapsed, modelCfg.ModelName, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}
