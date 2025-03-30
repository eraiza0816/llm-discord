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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"
	zutoolapi "github.com/eraiza0816/zu2l/api"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var weatherEmojiMap = map[int]string{
	100: "☀", // Sunny
	200: "☁", // Cloudy
	300: "☔", // Rainy
	400: "🌨", // Snowy
}

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
	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(token))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	genaiModel := genaiClient.GenerativeModel(model)

	zutoolClient := zutoolapi.NewClient("", "", 10*time.Second)

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
					Description: "指定されたキーワード（地名など）で場所を検索し、地点コードなどの情報を取得します。", // 地点コード取得の目的を明記
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
					Description: "指定された地点コードのOtenki ASP情報を取得します。", // 「Otenki ASP」を明記
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

	genaiModel.Tools = tools

	return &Chat{
		genaiClient:   genaiClient,
		genaiModel:    genaiModel,
		zutoolClient:  zutoolClient,
		defaultPrompt: defaultPrompt,
		historyMgr:    historyMgr,
		modelCfg:      modelCfg,
		tools:         tools,
	}, nil
}

func (c *Chat) GetResponse(userID, username, message, timestamp, prompt string) (string, float64, string, error) {

	toolInstructions := `
【Function Calling Rules】
あなたは以下のツール（関数）を利用できます。ユーザーのリクエストに応じて適切な関数を選択し、FunctionCallを返してください。
- getWeather: 「天気」について、地名（例：「東京」、「大阪市」）で質問された場合に使います。地点コードでは使いません。
- getPainStatus: 「頭痛」や「ずつう」について、地名（例：「横浜」）で質問された場合に使います。地点コードでは使いません。
- searchWeatherPoint: 「地点コード」を知りたい、または地名を検索したい場合に使います。キーワード（地名など）が必要です。
- getOtenkiAspInfo: 「Otenki ASP」の情報について、地点コード（例：「13112」）で質問された場合に使います。地名では使いません。
`
// 各関数のトリガー条件をより具体的に記述
	userHistory := c.historyMgr.Get(userID)
	historyText := ""
	if userHistory != "" {
		historyText = "会話履歴:\n" + userHistory + "\n\n"
	}

	fullInput := prompt + toolInstructions + "\n\n" + historyText + "ユーザーのメッセージ:\n" + message

	ctx := context.Background()
	start := time.Now()
	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())
	if err != nil {
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

					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("【%s (%s) の天気】\n", weatherStatus.PlaceName, location))
					if len(weatherStatus.Today) > 0 {
						sb.WriteString("今日:\n")
						for _, status := range weatherStatus.Today {

							tempStr := "---"
							if status.Temp != nil {
								tempStr = *status.Temp + "℃"
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
					log.Println("Returning result from getWeather case...")
					return sb.String(), 0, "zutool", nil

				case "getPainStatus":
					log.Println("Matched case: getPainStatus")
					location, ok := fn.Args["location"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("getPainStatus: location がありません")
					}

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

					var sb strings.Builder
					status := painStatus.PainnoterateStatus

					sb.WriteString(fmt.Sprintf("【%s (%s) の頭痛予報】\n", status.AreaName, location))
					sb.WriteString(fmt.Sprintf("期間: %s 〜 %s\n", status.TimeStart, status.TimeEnd))
					sb.WriteString("割合:\n")
					sb.WriteString(fmt.Sprintf("  ほぼ心配なし: %.1f%%\n", status.RateNormal))
					sb.WriteString(fmt.Sprintf("  やや注意: %.1f%%\n", status.RateLittle))
					sb.WriteString(fmt.Sprintf("  注意: %.1f%%\n", status.RatePainful))
					sb.WriteString(fmt.Sprintf("  警戒: %.1f%%\n", status.RateBad))

					return sb.String(), 0, "zutool", nil

				case "searchWeatherPoint":
					log.Println("Matched case: searchWeatherPoint")
					keyword, ok := fn.Args["keyword"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("searchWeatherPoint: keyword がありません")
					}

					weatherPoint, err := c.zutoolClient.GetWeatherPoint(keyword)
					if err != nil {
						log.Printf("GetWeatherPoint failed for search: %v", err)
						return fmt.Sprintf("「%s」の地点検索に失敗しちゃった… ごめんね🙏", keyword), 0, "zutool", nil
					}

					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("【「%s」の地点検索結果】\n", keyword))
					if len(weatherPoint.Result.Root) > 0 {
						for _, point := range weatherPoint.Result.Root {
							sb.WriteString(fmt.Sprintf("  - %s: %s\n", point.CityCode, point.Name))
						}
					} else {
						sb.WriteString("  該当する地点は見つからなかったみたい…\n")
					}
					return sb.String(), 0, "zutool", nil

				case "getOtenkiAspInfo":
					log.Println("Matched case: getOtenkiAspInfo")
					cityCode, ok := fn.Args["cityCode"].(string)
					if !ok {
						return "", 0, "", fmt.Errorf("getOtenkiAspInfo: cityCode がありません")
					}

					apiResponse, err := c.zutoolClient.GetOtenkiASP(cityCode)
					if err != nil {
						log.Printf("GetOtenkiASP failed: %v", err)
						return fmt.Sprintf("「%s」のASP情報の取得に失敗しちゃった… ごめんね🙏", cityCode), 0, "zutool", nil
					}

					jsonData, err := json.Marshal(apiResponse)
					if err != nil {
						log.Printf("Failed to marshal API response: %v", err)
						return "APIレスポンスの処理中にエラーが起きたよ… ごめんね🙏", 0, "zutool", nil
					}

					var genericData map[string]interface{}
					decoder := json.NewDecoder(bytes.NewReader(jsonData))
					decoder.UseNumber()
					if err := decoder.Decode(&genericData); err != nil {
						log.Printf("Failed to unmarshal API response into generic map: %v", err)
						log.Printf("JSON data was: %s", string(jsonData))
						return "APIレスポンスの解析中にエラーが起きたよ… ごめんね🙏", 0, "zutool", nil
					}

					var sb strings.Builder
					dateTimeInterface, dtOk := genericData["date_time"] // キーは "date_time"
					dateTimeFormatted := "不明"
					if dtOk {
						dateTimeStr, dtStrOk := dateTimeInterface.(string)
						if dtStrOk {
							// APIDateTime の UnmarshalJSON と同様のパースを試みる
							t, err := time.Parse("2006-01-02 15", dateTimeStr)
							if err == nil {
								dateTimeFormatted = t.Format("2006-01-02 15:04")
							} else {
								// RFC3339も試す (フォールバック)
								t, err = time.Parse(time.RFC3339, dateTimeStr)
								if err == nil {
									dateTimeFormatted = t.Format("2006-01-02 15:04")
								} else {
									log.Printf("Failed to parse date_time string %q: %v", dateTimeStr, err)
								}
							}
						}
					}
					sb.WriteString(fmt.Sprintf("【%s のOtenki ASP情報 (%s)】\n", cityCode, dateTimeFormatted))

					// Elements を処理 (genericData から取得)
					elementsRawInterface, elementsOk := genericData["elements"]
					if !elementsOk {
						sb.WriteString("  天気データが見つからなかったみたい… (elements missing)\n")
						return sb.String(), 0, "zutool", nil
					}
					elementsRaw, elementsSliceOk := elementsRawInterface.([]interface{})
					if !elementsSliceOk || len(elementsRaw) == 0 {
						sb.WriteString("  天気データが見つからなかったみたい… (elements not a slice or empty)\n")
						return sb.String(), 0, "zutool", nil
					}

					// 1. データを日付ごとに整理 (キーは日付文字列 "YYYYMMDD")
					// dataByDateStr[日付文字列YYYYMMDD][要素インデックス] = 値
					dataByDateStr := make(map[string][]interface{}) // 要素の順序を保持するためスライスに変更
					allDateStrsMap := make(map[string]struct{})

					// 想定される要素の数 (天気, 降水, 最高, 最低, 風速, 風向, 気圧Lv, 湿度)
					expectedElementCount := 8
					if len(elementsRaw) < expectedElementCount {
						log.Printf("Warning: Expected %d elements, but got %d", expectedElementCount, len(elementsRaw))
						// 足りない場合でも処理を続けるが、インデックス外参照に注意
					}

					// 各要素から日付ごとのレコードを抽出
					for elemIndex, elemInterface := range elementsRaw {
						elemMap, ok := elemInterface.(map[string]interface{})
						if !ok {
							log.Printf("Warning: Element at index %d is not a map, skipping", elemIndex)
							continue
						}

						recordsRaw, recordsOk := elemMap["records"].(map[string]interface{})
						if !recordsOk {
							log.Printf("Warning: Records for element at index %d is not a map, skipping", elemIndex)
							continue
						}

						for dateKeyStr, recordValue := range recordsRaw {
							// dateKeyStr は RFC3339 ("2025-04-01T00:00:00Z")
							t, err := time.Parse(time.RFC3339, dateKeyStr)
							if err != nil {
								log.Printf("Failed to parse date key string %q from records: %v", dateKeyStr, err)
								continue
							}
							dateStrYYYYMMDD := t.Format("20060102") // "YYYYMMDD" 形式

							if _, ok := dataByDateStr[dateStrYYYYMMDD]; !ok {
								// 新しい日付の場合、想定される要素数分のnilスライスを作成
								dataByDateStr[dateStrYYYYMMDD] = make([]interface{}, expectedElementCount)
							}

							// 正しいインデックスに値を格納 (範囲チェック)
							if elemIndex < expectedElementCount {
								dataByDateStr[dateStrYYYYMMDD][elemIndex] = recordValue
							}
							allDateStrsMap[dateStrYYYYMMDD] = struct{}{}
						}
					}


					// 2. 日付文字列 ("YYYYMMDD") をソート
					var sortedDateStrs []string
					for dateStr := range allDateStrsMap {
						sortedDateStrs = append(sortedDateStrs, dateStr)
					}
					sort.Strings(sortedDateStrs)

					// 3. ヘッダー行を作成 (Markdownテーブル風)
					header := "| 日付 | 天気 | 降水% | 最高℃ | 最低℃ | 風速m/s | 風向 | 気圧Lv | 湿度% |"
					separator := "|:---|:---|:----:|:-----:|:-----:|:------:|:--:|:------:|:----:|"
					sb.WriteString(header + "\n")
					sb.WriteString(separator + "\n")

					// 4. 日付ごとにデータを整形して追加
					// 各列に対応する要素のインデックス
					elementIndices := map[string]int{
						"天気": 0, "降水%": 1, "最高℃": 2, "最低℃": 3,
						"風速m/s": 4, "風向": 5, "気圧Lv": 6, "湿度%": 7,
					}
					columnOrder := []string{"天気", "降水%", "最高℃", "最低℃", "風速m/s", "風向", "気圧Lv", "湿度%"}


					for _, dateStr := range sortedDateStrs {
						// "YYYYMMDD" から "MM/DD" 形式へ
						dateFormatted := "-"
						t, err := time.Parse("20060102", dateStr)
						if err == nil {
							dateFormatted = t.Format("01/02")
						}

						row := []string{dateFormatted} // 日付
						dateData, ok := dataByDateStr[dateStr]
						if !ok || len(dateData) < expectedElementCount {
							// データがないか、要素数が足りない場合は '-' で埋める
							log.Printf("Warning: Data missing or incomplete for date %s", dateStr)
							for i := 0; i < len(columnOrder); i++ {
								row = append(row, "-")
							}
						} else {
							// データがある場合
							for _, columnName := range columnOrder {
								elemIndex := elementIndices[columnName]
								value := dateData[elemIndex] // interface{} 型
								valueStr := "-" // Default if missing or nil

								if value != nil {
									// 型に応じてフォーマット (json.Number を考慮)
									switch v := value.(type) {
									case string:
										if columnName == "天気" {
											// 天気は文字列の場合と数値コードの場合がある
											weatherCodeInt, err := strconv.Atoi(v)
											emoji := "?"
											if err == nil { // 数値コードの場合
												simplifiedCode := (weatherCodeInt / 100) * 100
												if e, okEmoji := weatherEmojiMap[simplifiedCode]; okEmoji {
													emoji = e
												} else { emoji = v } // Mapにないコード
											} else { // 文字列の場合 (例: "くもり 時々 雨")
												// 文字列の場合はそのまま表示するか、代表的な絵文字を当てるか？
												// 一旦そのまま表示
												emoji = v
											}
											valueStr = emoji
										} else { // 風向など
											valueStr = v
										}
									case json.Number:
										floatVal, err := v.Float64()
										if err == nil {
											// 整数なら整数、小数なら小数点第一位まで
											if columnName == "降水%" || columnName == "湿度%" || columnName == "気圧Lv" || columnName == "風向" {
												if floatVal == float64(int(floatVal)) {
													valueStr = strconv.Itoa(int(floatVal))
												} else { valueStr = fmt.Sprintf("%.1f", floatVal) }
											} else { // 最高/最低気温、風速
												valueStr = fmt.Sprintf("%.1f", floatVal)
											}
										} else { valueStr = v.String() } // Float変換失敗
									case float64: // フォールバック
										if columnName == "降水%" || columnName == "湿度%" || columnName == "気圧Lv" || columnName == "風向" {
											if v == float64(int(v)) {
												valueStr = strconv.Itoa(int(v))
											} else { valueStr = fmt.Sprintf("%.1f", v) }
										} else { valueStr = fmt.Sprintf("%.1f", v) }
									default:
										valueStr = fmt.Sprintf("%v", v) // その他の型
									}
								}
								row = append(row, valueStr)
							}
						}
						sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
					}
					return sb.String(), 0, "zutool", nil

				default:
					log.Printf("Matched default case in switch: Unknown function %s", fn.Name) // ✨ Default Caseログ
					return "", 0, "", fmt.Errorf("不明な関数が呼び出されました: %s", fn.Name)
				}

				case genai.Text:
					log.Printf("Part %d is genai.Text: %s", i, string(v))
				default:
					log.Printf("Part %d is an unexpected type: %T", i, v)
				}
			}

			if !functionCallProcessed {
				log.Println("No FunctionCall was processed in response parts.")
			}

		} else {
			log.Println("Gemini response candidate content or parts are empty.")
		}
	} else {
		log.Println("Gemini response candidates are empty.")
	}

	responseText := getResponseText(resp)
	log.Printf("Final response text to be returned: %s", responseText)

	if responseText != "" {
		c.historyMgr.Add(userID, message, responseText)
		log.Printf("Added to history for user %s: message='%s', response='%s'", userID, message, responseText)
	} else {
		log.Printf("Skipping history add for user %s because responseText is empty.", userID)
	}


	return responseText, elapsed, c.modelCfg.ModelName, nil
}

func (c *Chat) getOllamaResponse(userID, fullInput string) (string, float64, error) {
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
