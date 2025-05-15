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
	genaiModel       *genai.GenerativeModel
	weatherService   WeatherService
	urlReaderService *URLReaderService // URLReaderServiceを追加
	defaultPrompt    string
	historyMgr       history.HistoryManager
	tools            []*genai.Tool
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
	urlReaderService := NewURLReaderService() // URLReaderServiceを初期化

	weatherFuncDeclarations := weatherService.GetFunctionDeclarations()
	urlReaderFuncDeclaration := GetURLReaderFunctionDeclaration() // URLリーダーの関数宣言を取得

	tools := []*genai.Tool{
		{
			FunctionDeclarations: weatherFuncDeclarations,
		},
		{ // URLリーダーのツールを追加
			FunctionDeclarations: []*genai.FunctionDeclaration{urlReaderFuncDeclaration},
		},
	}

	genaiModel.Tools = tools

	return &Chat{
		genaiClient:      genaiClient,
		genaiModel:       genaiModel,
		weatherService:   weatherService,
		urlReaderService: urlReaderService, // 初期化したサービスをセット
		historyMgr:       historyMgr,
		tools:            tools,
	}, nil
}

func (c *Chat) GetResponse(userID, threadID, username, message, timestamp, prompt string) (string, float64, string, error) {
	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		errorLogger.Printf("Error loading model config in GetResponse: %v", err)
		return "", 0, "", fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// buildFullInput に threadID と timestamp を渡すように変更 (prompt.go の修正も必要)
	fullInput := buildFullInput(prompt, message, c.historyMgr, userID, threadID, timestamp)

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

					if fn.Name == "get_url_content" {
						// URLリーダー関数の処理
						urlString, ok := fn.Args["url"].(string)
						if !ok {
							toolResult = "URLが正しく指定されていません。"
							errorLogger.Printf("Function call 'get_url_content' missing or invalid 'url' argument: %v", fn.Args)
						} else {
							toolResult, toolErr = c.urlReaderService.GetURLContentAsText(urlString)
							if toolErr != nil {
								errorLogger.Printf("Error handling function call 'get_url_content' for URL %s: %v", urlString, toolErr)
								toolResult = fmt.Sprintf("URL '%s' の内容取得中にエラー: %v", urlString, toolErr)
								toolErr = nil // エラーは結果文字列に含めたので、ここではリセット
							}
						}
					} else {
						// 既存の天候情報関数の処理
						toolResult, toolErr = c.weatherService.HandleFunctionCall(fn)
						if toolErr != nil {
							errorLogger.Printf("Error handling function call %s via WeatherService: %v", fn.Name, toolErr)
							toolResult = fmt.Sprintf("関数の処理中にエラーが発生しました: %v", toolErr)
							toolErr = nil
						}
					}
				default:
					errorLogger.Printf("Part %d is an unexpected type: %T", i, v)
				}
			}

			if functionCallProcessed {
				// Function Calling の結果をLLMに再度渡して、最終的な応答を生成させる
				// この部分は、Function Callingの応答をどのように扱うかによって実装が変わる
				// ここでは、toolResultを次のLLMへの入力パートとして追加する
				// 実際には、FunctionResponseパートを作成し、再度GenerateContentを呼び出す
				log.Printf("Function call processed. Result: %s. Intro text: %s", toolResult, llmIntroText.String())

				// FunctionResponseを作成 (現在は直接使用していないが、将来的な拡張のためにコメントとして残す)
				// parts := []genai.Part{
				// 	genai.FunctionResponse{
				// 		Name: candidate.Content.Parts[0].(genai.FunctionCall).Name, // 呼び出された関数名
				// 		Response: map[string]interface{}{
				// 			"content": toolResult, // ツールからの生の出力
				// 		},
				// 	},
				// }
				// LLMからの導入テキストがあれば、それもコンテキストに含める
				// ただし、通常はFunctionResponseのみを返し、LLMがそれを解釈して自然な応答を生成する
				// if llmIntroText.Len() > 0 {
				// parts = append([]genai.Part{genai.Text(llmIntroText.String())}, parts...)
				// }


				// 再度GenerateContentを呼び出す
				// 履歴にFunctionCallとFunctionResponseを追加する必要がある
				// ここでは簡略化のため、toolResultをそのまま返す
				// 実際には、この応答を元にLLMが自然なテキストを生成する
				// c.historyMgr.Add(userID, threadID, message, fmt.Sprintf("Tool execution: %s", toolResult)) // 履歴への追加は検討

				// FunctionResponseを作成
				// fnName := candidate.Content.Parts[0].(genai.FunctionCall).Name // 呼び出された関数名
				var calledFuncName string
				for _, part := range candidate.Content.Parts {
					if fc, ok := part.(genai.FunctionCall); ok {
						calledFuncName = fc.Name
						break
					}
				}

				if calledFuncName == "" {
					errorLogger.Printf("Could not determine called function name from candidate parts.")
					// フォールバックとして、toolResultをそのまま返すか、エラーメッセージを返す
					return "関数呼び出し名の取得に失敗しました。", elapsed, modelCfg.ModelName, fmt.Errorf("関数呼び出し名の取得に失敗")
				}


				// 新しいコンテンツリストを作成して、再度GenerateContentを呼び出す
				// 1. 元のユーザー入力 (fullInput)
				// 2. LLMの最初の応答 (FunctionCallを含む candidate.Content)
				// 3. ツールの実行結果 (FunctionResponse)
				// これらを会話履歴としてモデルに渡す
				// ただし、genai.GenerativeModel.GenerateContent は []*Content ではなく、[]Part を取る。
				// セッション (ChatSession) を使うと履歴管理が容易になるが、ここでは手動で構築。

				// 履歴を構築する代わりに、現在のGenerateContent呼び出しにFunctionResponseを追加する
				// GenerateContentの入力は []Part なので、FunctionResponseをPartとして渡す
				// ただし、GenerateContentは通常、新しいユーザー入力から開始する。
				// Function Callingの正しいフローでは、ChatSessionを使い、
				// session.SendMessage(FunctionResponse) のようにして履歴を継続する。
				// ここでは、既存の fullInput (ユーザーの元のメッセージ) と、
				// LLMのFunctionCall、そしてツールのFunctionResponseを結合して新しい入力とする。
				// これは厳密には正しくないが、ChatSessionを使わない場合の次善策。

				// 正しいアプローチ: ChatSession を使うか、手動で会話履歴を構築する
				// ここでは、GenerateContentに渡すpartsを構築して再呼び出しする
				// 1. ユーザーの元の入力 (genai.Text(fullInput) として渡されたもの)
				// 2. LLMの最初の応答 (FunctionCallを含む candidate.Content)
				// 3. ツールの実行結果 (FunctionResponse)

				// ユーザーの元の入力は fullInput だが、GenerateContent は []Part を取る。
				// fullInput は既に genai.Text(fullInput) として最初の呼び出しで使われている。
				// 履歴を模倣するために、これらの要素を []Part として組み立てる。

				// var historyForSecondCall []genai.Part // 未使用のためコメントアウトまたは削除
				// 1. ユーザーの元のメッセージ (fullInput はシステムプロンプト等も含むため、message の方が適切か検討)
				//    ただし、最初の GenerateContent には genai.Text(fullInput) を渡している。
				//    一貫性のため、ここでも genai.Text(fullInput) を使うか、あるいは
				//    ChatSession のようにユーザーロールとモデルロールを区別したContentオブジェクトを使うべき。
				//    genai.Text は単一のテキストパート。
				//    より正確には、最初のユーザーメッセージ、最初のモデル応答(FC)、ツール応答(FR)の順。

				// 2回目のGenerateContentに渡すpartsを構築する。
				// 理想的には、ChatSessionを使い、履歴を適切に管理する。
				// ここでは、手動で履歴のターンを模倣する。
				// ターン1: ユーザー (fullInput に含まれるユーザーメッセージ)
				// ターン2: モデル (FunctionCall を含む candidate.Content.Parts)
				// ターン3: ツール (FunctionResponse)
				// これらを GenerateContent に渡す。
				// ただし、GenerateContent は基本的に「現在のターン」の parts を期待する。
				// 複数のターンを渡す場合は、Content{Role:"user", Parts:...}, Content{Role:"model", Parts:...} のリストを
				// ChatSession.History に設定し、SendMessage で新しいターンを開始するのが正しい。

				// 現在の GenerateContent の枠組みでの改善案:
				// 1. ユーザーの元の入力 (genai.Text(fullInput)) -> これは最初の呼び出しで使ったもの
				// 2. LLMの最初の応答 (candidate.Content.Parts)
				// 3. ツールの実行結果 (genai.FunctionResponse)
				// これらをすべて含めて GenerateContent を呼び出す。
				// ただし、genai.Text(fullInput) はシステムプロンプト等も含むため、
				// ユーザーメッセージ部分だけを抽出して genai.Text として渡す方が良いかもしれない。
				// ここでは、最初の呼び出しと同じ genai.Text(fullInput) は含めず、
				// LLMの最初の応答 (FunctionCall) とツールの結果 (FunctionResponse) のみを渡す現在の方法を維持しつつ、
				// ログ出力を強化して、何が渡されているかを確認できるようにする。
				// もしこれでもダメなら、ChatSessionへの移行を強く推奨する。

				var partsForNextTurn []genai.Part
				// LLMの最初の応答 (FunctionCallを含む)
				partsForNextTurn = append(partsForNextTurn, candidate.Content.Parts...)
				// ツールの実行結果
				partsForNextTurn = append(partsForNextTurn, genai.FunctionResponse{
					Name: calledFuncName,
					Response: map[string]interface{}{
						"content": toolResult,
					},
				})

				log.Printf("Re-calling GenerateContent with %d parts for function %s. Parts: %+v", len(partsForNextTurn), calledFuncName, partsForNextTurn)
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

				// 導入テキストと結合する場合 (オプション)
				// combinedResponse := llmIntroText.String()
				// if combinedResponse != "" && !strings.HasSuffix(combinedResponse, "\n") {
				// 	combinedResponse += "\n"
				// }
				// combinedResponse += finalResponseText
				// finalResponseText = combinedResponse

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
	responseText := "すみません、応答を取得できませんでした。"
	return responseText, elapsed, modelCfg.ModelName, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}

func GetErrorLogger() *log.Logger {
	return errorLogger
}
