package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"
	zutoolapi "github.com/eraiza0816/zu2l/api"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Service interface {
	GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error)
	Close()
}

type Chat struct {
	genaiClient   *genai.Client
	genaiModel    *genai.GenerativeModel
	zutoolClient  *zutoolapi.Client
	defaultPrompt string
	historyMgr    history.HistoryManager
	modelCfg      *loader.ModelConfig
	tools         []*genai.Tool
}

func NewChat(token string, model string, defaultPrompt string, modelCfg *loader.ModelConfig, historyMgr history.HistoryManager) (Service, error) {
	// Gemini クライアントの初期化
	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	genaiModel := genaiClient.GenerativeModel(model)

	// zu2l API クライアントの初期化 (タイムアウトは適当に10秒)
	// TODO: タイムアウト値を設定可能にする
	zutoolClient := zutoolapi.NewClient("", "", 10*time.Second)

	// ✨ Tool の定義
	tools := []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "getWeather",
					Description: "指定された場所の天気を取得します。",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"location": {
								Type:        genai.TypeString,
								Description: "天気情報を取得する場所",
							},
						},
						Required: []string{"location"},
					},
				},
				{
					Name:        "getPainStatus",
					Description: "指定された場所の頭痛予報を取得します。",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"location": {
								Type:        genai.TypeString,
								Description: "頭痛予報を取得する場所",
							},
						},
						Required: []string{"location"},
					},
				},
				{
					Name:        "searchWeatherPoint",
					Description: "指定されたキーワードで場所を検索します。",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"keyword": {
								Type:        genai.TypeString,
								Description: "検索キーワード",
							},
						},
						Required: []string{"keyword"},
					},
				},
				{
					Name:        "getOtenkiAspInfo",
					Description: "指定された地点コードのASP情報を取得します。",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"cityCode": {
								Type:        genai.TypeString,
								Description: "地点コード",
							},
						},
						Required: []string{"cityCode"},
					},
				},
			},
		},
	}

	genaiModel.Tools = tools // ✨ Tool を設定

	return &Chat{
		genaiClient:   genaiClient,
		genaiModel:    genaiModel,
		zutoolClient:  zutoolClient, // ✨ 初期化したクライアントを設定
		defaultPrompt: defaultPrompt,
		historyMgr:    historyMgr,
		modelCfg:      modelCfg,
		tools:         tools, // ✨ Tool を設定
	}, nil
}

// formatHistory 関数は history.MessagePair が存在しないため削除

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error) {

	toolInstructions := `
【Function Calling Rules】
あなたは以下のツール（関数）を利用できます。ユーザーのリクエストに応じて適切な関数を選択し、FunctionCallを返してください。
- getWeather: 天気に関する質問の場合。地名が必要です。
- getPainStatus: 頭痛予報に関する質問の場合。地名が必要です。
- searchWeatherPoint: 地点検索に関する質問の場合。検索キーワード（地名）が必要です。
- getOtenkiAspInfo: ASP情報に関する質問の場合。地点コードが必要です。
`
	// 1. 履歴を取得
	userHistory := c.historyMgr.Get(userID)
	historyText := ""
	if userHistory != "" {
		historyText = "会話履歴:\n" + userHistory + "\n\n"
	}

	// 2. プロンプトとメッセージを結合 (履歴も追加)
	fullInput := prompt + toolInstructions + "\n\n" + historyText + "ユーザーのメッセージ:\n" + message

	// 3. Geminiの呼び出し (GenerateContent を使用)
	ctx := context.Background()
	start := time.Now() // 時間計測開始
	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds()) // 時間計測終了

	if err != nil {
		// elapsed を返すように修正
		return "", elapsed, "", fmt.Errorf("Gemini APIからのエラー: %w", err)
	}

	// 5. Function callの実行とログ出力強化
	if resp.Candidates != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			// ✨ Geminiからの応答パーツ全体をログに出力
			log.Printf("Gemini response parts: %+v", candidate.Content.Parts)

			// ✨ FunctionCallを探して処理する type switch を使用
			var functionCallProcessed bool // FunctionCallを処理したかどうかのフラグ
			for i, part := range candidate.Content.Parts {
				log.Printf("Processing part %d", i)
				switch v := part.(type) { // ✨ type switch を使用
				case genai.FunctionCall: // ✨ ポインタ(*)を削除！ value type でチェック
					// ✨ Function Call が見つかった！
					log.Printf("Part %d IS a genai.FunctionCall!", i)
					fn := v // 型アサーション済みなのでそのまま使える
					log.Printf("Function Call triggered: %s, Args: %v", fn.Name, fn.Args)
					functionCallProcessed = true // フラグを立てる

					// 6. Function callの実行 (既存のswitch文をここに移動)
					log.Printf("Entering switch statement for FunctionCall: %s", fn.Name)
					switch fn.Name {
					case "getWeather":
						log.Println("Matched case: getWeather")
						// 引数を取得
						location, ok := fn.Args["location"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("getWeather: location がありません")
					}

					// ✨ 地点コードを取得
					weatherPoint, err := c.zutoolClient.GetWeatherPoint(location)
					if err != nil {
						log.Printf("GetWeatherPoint failed for weather: %v", err)
						return fmt.Sprintf("「%s」の地点情報の取得に失敗しちゃった… ごめんね🙏", location), 0, "zutool", nil
					}
					if len(weatherPoint.Result.Root) == 0 {
						return fmt.Sprintf("「%s」って場所が見つからなかったみたい…🤔", location), 0, "zutool", nil
					}
					// 最初の地点コードを使用
					cityCode := weatherPoint.Result.Root[0].CityCode

					// ✨ 天気情報をAPIで取得
					weatherStatus, err := c.zutoolClient.GetWeatherStatus(cityCode)
					if err != nil {
						log.Printf("GetWeatherStatus failed: %v", err)
						return fmt.Sprintf("「%s」(%s)の天気情報の取得に失敗しちゃった… ごめんね🙏", location, cityCode), 0, "zutool", nil
					}

					// ✨ 結果を整形して返す
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("【%s (%s) の天気】\n", weatherStatus.PlaceName, location))
					if len(weatherStatus.Today) > 0 {
						sb.WriteString("今日:\n")
						for _, status := range weatherStatus.Today {
							// ✨ フィールド名と型を修正 (Tempは*string, Pressureはstring)
							tempStr := "---" // 温度がnilの場合のデフォルト表示
							if status.Temp != nil {
								tempStr = *status.Temp + "℃" // ポインタをデリファレンスして℃を付ける
							}
							sb.WriteString(fmt.Sprintf("  %s: %s, %shPa, %s\n",
								status.Time,     // DateTime -> Time (string)
								tempStr,         // Temperature -> Temp (*string)
								status.Pressure, // Pressure (string)
								status.Weather)) // Weather (WeatherEnum)
						}
					} else {
						sb.WriteString("  今日のデータはないみたい…\n")
					}
					log.Println("Returning result from getWeather case...") // ✨ Return直前のログ
					return sb.String(), 0, "zutool", nil

				case "getPainStatus":
					log.Println("Matched case: getPainStatus") // ✨ Case一致ログ
					// 引数を取得
					location, ok := fn.Args["location"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("getPainStatus: location がありません")
					}

					// ✨ 地点コードを取得
					weatherPoint, err := c.zutoolClient.GetWeatherPoint(location)
					if err != nil {
						log.Printf("GetWeatherPoint failed for headache: %v", err)
						return fmt.Sprintf("「%s」の地点情報の取得に失敗しちゃった… ごめんね🙏", location), 0, "zutool", nil
					}
					if len(weatherPoint.Result.Root) == 0 {
						return fmt.Sprintf("「%s」って場所が見つからなかったみたい…🤔", location), 0, "zutool", nil
					}
					// 最初の地点コードを使用
					cityCode := weatherPoint.Result.Root[0].CityCode
					// 地点コードの最初の2桁を地域コードとして使用 (例: "13101" -> "13")
					areaCode := ""
					if len(cityCode) >= 2 {
						areaCode = cityCode[:2]
					}
					if areaCode == "" {
						log.Printf("Failed to extract area code from city code: %s", cityCode)
						return fmt.Sprintf("「%s」の地域コードがわからなかった… ごめんね🙏", location), 0, "zutool", nil
					}

					// ✨ 頭痛情報をAPIで取得
					painStatus, err := c.zutoolClient.GetPainStatus(areaCode, &cityCode) // cityCodeをsetPointとして渡す
					if err != nil {
						log.Printf("GetPainStatus failed: %v", err)
						return fmt.Sprintf("「%s」(%s)の頭痛情報の取得に失敗しちゃった… ごめんね🙏", location, cityCode), 0, "zutool", nil
					}

					// ✨ 結果を整形して返す (CommentとLevelはエラーが出ていたため一旦削除)
					// TODO: GetPainStatus の正確な構造体を確認して修正する
					responseText := fmt.Sprintf("【%s (%s) の頭痛予報】\n(詳細情報は現在調整中です🙏)",
						painStatus.PainnoterateStatus.AreaName, // APIが返すエリア名を使う
						location) // ユーザーが指定した地名も表示
					return responseText, 0, "zutool", nil

				case "searchWeatherPoint":
					log.Println("Matched case: searchWeatherPoint") // ✨ Case一致ログ
					// 引数を取得
					keyword, ok := fn.Args["keyword"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("searchWeatherPoint: keyword がありません")
					}

					// ✨ 地点情報をAPIで取得
					weatherPoint, err := c.zutoolClient.GetWeatherPoint(keyword)
					if err != nil {
						log.Printf("GetWeatherPoint failed for search: %v", err)
						return fmt.Sprintf("「%s」の地点検索に失敗しちゃった… ごめんね🙏", keyword), 0, "zutool", nil
					}

					// ✨ 結果を整形して返す
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("【「%s」の地点検索結果】\n", keyword))
					if len(weatherPoint.Result.Root) > 0 {
						for _, point := range weatherPoint.Result.Root {
							// ✨ Kanaフィールドは存在しないため削除
							sb.WriteString(fmt.Sprintf("  - %s: %s\n", point.CityCode, point.Name))
						}
					} else {
						sb.WriteString("  該当する地点は見つからなかったみたい…\n")
					}
					return sb.String(), 0, "zutool", nil

				case "getOtenkiAspInfo":
					log.Println("Matched case: getOtenkiAspInfo") // ✨ Case一致ログ
					// 引数を取得
					cityCode, ok := fn.Args["cityCode"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("getOtenkiAspInfo: cityCode がありません")
					}

					// ✨ Otenki ASP情報をAPIで取得
					otenkiData, err := c.zutoolClient.GetOtenkiASP(cityCode)
					if err != nil {
						log.Printf("GetOtenkiASP failed: %v", err)
						return fmt.Sprintf("「%s」のASP情報の取得に失敗しちゃった… ごめんね🙏", cityCode), 0, "zutool", nil
					}

					// ✨ 結果を整形して返す (簡易版)
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("【%s のOtenki ASP情報 (%s)】\n", cityCode, otenkiData.DateTime)) // 地名がないのでコードで表示
					if len(otenkiData.Elements) > 0 {
						sb.WriteString(fmt.Sprintf("%d個の要素が見つかったよ！\n", len(otenkiData.Elements)))
						// 例として最初の数件の要素名を表示
						count := 0
						for _, elem := range otenkiData.Elements {
							if count < 5 { // 表示件数を制限
								sb.WriteString(fmt.Sprintf("  - %s (%s)\n", elem.Title, elem.ContentID))
								count++
							} else {
								sb.WriteString("  ...\n")
								break
							}
						}
					} else {
						sb.WriteString("  データが見つからなかったみたい…\n")
					}
					return sb.String(), 0, "zutool", nil

				default:
					log.Printf("Matched default case in switch: Unknown function %s", fn.Name) // ✨ Default Caseログ
					return "", 0, "", fmt.Errorf("不明な関数が呼び出されました: %s", fn.Name)
				}
					// Since all cases return, this point should not be reached if a FunctionCall was processed.

				case genai.Text:
					log.Printf("Part %d is genai.Text: %s", i, string(v))
				// 他の期待される型があればここに追加 (例: case *genai.Blob:)
				default:
					log.Printf("Part %d is an unexpected type: %T", i, v)
				}

				// FunctionCallを処理したらループを抜ける (通常、応答にFunctionCallは1つのはず)
				// ただし、テキストとFunctionCallが両方返る場合があるので、最後までループは回す
				// if functionCallProcessed { break } // ← 一旦コメントアウト

			} // End of loop through parts

			// ✨ ループ後、FunctionCallが処理されたかどうかをチェック
			if !functionCallProcessed {
				// FunctionCallが見つからなかった、または処理されなかった場合
				log.Println("No FunctionCall was processed in response parts.")
			}
			// If functionCallProcessed is true, we should have already returned from within the switch.

		} else {
			log.Println("Gemini response candidate content or parts are empty.")
		}
	} else {
		log.Println("Gemini response candidates are empty.")
	}

	// 7. Function callがなかった場合、またはFunctionCall処理後の応答取得
	//    (FunctionCallの場合はAPI実行結果がresponseTextに入る想定だが、現状はLLMの応答をそのまま取得している)
	// TODO: FunctionCall成功時はAPIの結果をresponseTextに入れるように修正する
	responseText := getResponseText(resp) // getResponseTextはそのまま使える
	log.Printf("Final response text to be returned: %s", responseText) // ✨ 最終応答のログ

	// 8. 履歴に追加 (Function Call 以外の場合)
	// Function Call の場合は、APIの結果ではなくLLM自身の応答を履歴に残すか検討が必要
	// 現状は Function Call でも LLM の応答 (responseText) を履歴に追加する
	// TODO: Function Call の場合の履歴の扱いを再検討する
	if responseText != "" { // 空の応答は追加しない
		c.historyMgr.Add(userID, message, responseText)
		log.Printf("Added to history for user %s: message='%s', response='%s'", userID, message, responseText)
	} else {
		log.Printf("Skipping history add for user %s because responseText is empty.", userID)
	}


	return responseText, elapsed, c.modelCfg.ModelName, nil
}

func (c *Chat) getOllamaResponse(userID, fullInput string) (string, float64, error) { // userID を引数に追加 (ただし未使用)
	log.Println("Warning: Ollama history processing is not implemented yet.")

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
			continue // JSON解析失敗時はこの行の処理をスキップ
		}

		// JSON解析成功時のみレスポンス内容を処理
		responsePart, ok := result["response"].(string)
		if ok {
			responseText += responsePart
		}

		// 完了チェック
		done, ok := result["done"].(bool)
		if ok && done {
			break // ストリーム完了
		}
	}
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
