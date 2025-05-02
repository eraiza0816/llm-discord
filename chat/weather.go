package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	zutoolapi "github.com/eraiza0816/zu2l/api"
	"github.com/google/generative-ai-go/genai"
)

type WeatherService interface {
	GetFunctionDeclarations() []*genai.FunctionDeclaration
	HandleFunctionCall(fn genai.FunctionCall) (string, error)
}

type weatherServiceImpl struct {
	client *zutoolapi.Client
}

func NewWeatherService() WeatherService {
	client := zutoolapi.NewClient("", "", 10*time.Second)
	return &weatherServiceImpl{
		client: client,
	}
}

// getLocationInfo は指定された場所の CityCode と Name を取得する
func (ws *weatherServiceImpl) getLocationInfo(location string) (cityCode string, pointName string, err error) {
	// GetWeatherPoint は GetWeatherPointResponse を返す
	weatherPointResponse, err := ws.client.GetWeatherPoint(location)
	if err != nil {
		log.Printf("GetWeatherPoint failed for location '%s': %v", location, err)
		// エラー時は空文字列とエラーを返す
		err = fmt.Errorf("地点情報の取得に失敗しました (エラー: %w)", err) // ユーザー向けメッセージは修正しない
		return
	}
	// Response 内の Result.Root スライスを確認
	if len(weatherPointResponse.Result.Root) == 0 {
		err = fmt.Errorf("場所「%s」が見つかりませんでした", location) // ユーザー向けメッセージは修正しない
		return
	}
	// Result.Root スライスの最初の要素から CityCode と Name を取得
	pointData := weatherPointResponse.Result.Root[0]
	cityCode = pointData.CityCode
	pointName = pointData.Name
	return // cityCode, pointName, nil が返る
}

// getWeatherEmoji は天気コード（数値または文字列）を受け取り、対応する絵文字を返す
// 不明なコードや変換エラーの場合は元のコード文字列または "?" を返す
func getWeatherEmoji(weatherCodeValue interface{}) string {
	var weatherCodeInt int
	var err error
	var originalCodeStr string

	switch v := weatherCodeValue.(type) {
	case string:
		originalCodeStr = v
		weatherCodeInt, err = strconv.Atoi(v)
	case json.Number:
		originalCodeStr = v.String()
		var int64Val int64
		int64Val, err = v.Int64()
		if err == nil {
			weatherCodeInt = int(int64Val)
		}
	case int:
		originalCodeStr = strconv.Itoa(v)
		weatherCodeInt = v
	case float64: // json.Number を使わない場合など
		originalCodeStr = fmt.Sprintf("%v", v) // 元の値を文字列として保持
		if v == float64(int(v)) {
			weatherCodeInt = int(v)
			err = nil
		} else {
			err = fmt.Errorf("float value cannot be directly converted to weather code int: %f", v)
		}
	default:
		originalCodeStr = fmt.Sprintf("%v", v)
		err = fmt.Errorf("unsupported type for weather code: %T", v)
	}

	if err != nil {
		log.Printf("Could not convert weather code '%s' to int: %v", originalCodeStr, err)
		// 変換エラーの場合は元の文字列を返すか、デフォルトの絵文字を返す
		return originalCodeStr // または "?"
	}

	// 100の位で丸める (例: 101 -> 100)
	simplifiedCode := (weatherCodeInt / 100) * 100
	if emoji, ok := weatherEmojiMap[simplifiedCode]; ok {
		return emoji
	}

	log.Printf("Weather code %d (simplified %d, original %q) not found in emoji map.", weatherCodeInt, simplifiedCode, originalCodeStr)
	// マップにない場合は元のコード文字列を返す
	return originalCodeStr
}

// --- End Helper Functions ---

// weatherEmojiMap は天気コードを絵文字にマッピングする
var weatherEmojiMap = map[int]string{
	100: "☀️", // 快晴
	200: "☁️", // 曇り
	300: "🌧️", // 雨
	400: "🌨️", // 雪
	// 必要に応じて他のコードを追加
}

func (ws *weatherServiceImpl) GetFunctionDeclarations() []*genai.FunctionDeclaration {
	return []*genai.FunctionDeclaration{
		{
			Name:        "getWeather",
			Description: "指定された場所の天気を取得する",
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
			Description: "指定された場所の頭痛予報を取得する",
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
			Description: "指定されたキーワード（地名など）で場所を検索し、地点コードなどの情報を取得する",
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
			Description: "指定された地点コードのOtenki ASP情報を取得する",
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
	}
}

func (ws *weatherServiceImpl) HandleFunctionCall(fn genai.FunctionCall) (string, error) {
	log.Printf("Handling Weather Function Call: %s, Args: %v", fn.Name, fn.Args)
	switch fn.Name {
	case "getWeather":
		return ws.handleGetWeather(fn.Args)
	case "getPainStatus":
		return ws.handleGetPainStatus(fn.Args)
	case "searchWeatherPoint":
		return ws.handleSearchWeatherPoint(fn.Args)
	case "getOtenkiAspInfo":
		return ws.handleGetOtenkiAspInfo(fn.Args)
	default:
		log.Printf("Unknown weather function call: %s", fn.Name)
		return "", fmt.Errorf("不明な天気関数が呼び出されました: %s", fn.Name) // ユーザー向けメッセージは修正しない
	}
}

func (ws *weatherServiceImpl) handleGetWeather(args map[string]interface{}) (string, error) {
	location, ok := args["location"].(string)
	if !ok {
		return "", fmt.Errorf("getWeather: location がありません")
	}

	// 地点情報をヘルパー関数で取得 (cityCode と pointName を直接受け取る)
	cityCode, pointName, err := ws.getLocationInfo(location)
	if err != nil {
		// エラーメッセージはヘルパー関数内で生成される
		return fmt.Sprintf("「%s」の天気情報を取得できませんでした: %v", location, err), nil // ユーザー向けメッセージは修正しない
	}
	// cityCode と pointName は取得済み

	weatherStatus, err := ws.client.GetWeatherStatus(cityCode)
	if err != nil {
		log.Printf("GetWeatherStatus failed for %s (%s): %v", pointName, cityCode, err)
		return fmt.Sprintf("「%s」(%s)の天気情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", pointName, cityCode, err), nil // ユーザー向けメッセージは修正しない
	}

	var sb strings.Builder
	// PlaceName が空の場合があるため、取得した地点名を使う
	displayName := weatherStatus.PlaceName
	if displayName == "" {
		displayName = pointName
	}
	sb.WriteString(fmt.Sprintf("【%s (%s) の天気】\n", displayName, location))
	if len(weatherStatus.Today) > 0 {
		sb.WriteString("今日の天気:\n") // より自然な表現
		for _, status := range weatherStatus.Today {
			tempStr := "---"
			if status.Temp != nil {
				tempStr = *status.Temp + "℃"
			}
			// --- 天気コードを絵文字に変換 (ヘルパー関数使用) ---
			weatherEmoji := getWeatherEmoji(string(status.Weather)) // status.Weather は WeatherEnum (string) と仮定
			// --- ここまで変換処理 ---
			sb.WriteString(fmt.Sprintf("  %s: %s, %shPa, %s\n",
				status.Time, tempStr, status.Pressure, weatherEmoji))
		}
	} else {
		sb.WriteString("  今日の詳細な天気データはないみたい…\n") // ユーザー向けメッセージは修正しない
	}
	return sb.String(), nil
}

func (ws *weatherServiceImpl) handleGetPainStatus(args map[string]interface{}) (string, error) {
	location, ok := args["location"].(string)
	if !ok {
		return "", fmt.Errorf("getPainStatus: location がありません")
	}

	// 地点情報をヘルパー関数で取得 (cityCode と pointName を直接受け取る)
	cityCode, pointName, err := ws.getLocationInfo(location)
	if err != nil {
		return fmt.Sprintf("「%s」の頭痛予報を取得できませんでした: %v", location, err), nil // ユーザー向けメッセージは修正しない
	}
	// cityCode と pointName は取得済み

	areaCode := ""
	if len(cityCode) >= 2 {
		areaCode = cityCode[:2] // areaCode は cityCode の先頭2文字
	} else { // areaCode が取得できなかった場合の else ブロックを追加
		log.Printf("Failed to extract area code from city code '%s' for location '%s'", cityCode, location)
		return fmt.Sprintf("「%s」(%s) の地域コードが特定できませんでした", pointName, location), nil // ユーザー向けメッセージは修正しない
	} // if len(cityCode) >= 2 の閉じ括弧を正しい位置に移動

	painStatus, err := ws.client.GetPainStatus(areaCode, &cityCode)
	if err != nil {
		log.Printf("GetPainStatus failed for %s (%s): %v", pointName, cityCode, err)
		return fmt.Sprintf("「%s」(%s)の頭痛情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", pointName, cityCode, err), nil // ユーザー向けメッセージは修正しない
	}

	var sb strings.Builder
	status := painStatus.PainnoterateStatus
	// AreaName が空の場合があるため、取得した地点名を使う
	displayName := status.AreaName
	if displayName == "" {
		displayName = pointName // 地点検索結果の名前を使う
	}
	sb.WriteString(fmt.Sprintf("【%s (%s) の頭痛予報】\n", displayName, location))
	// TimeStart, TimeEnd のフォーマットを調整 (必要であれば)
	// 例: tStart, err := time.Parse(time.RFC3339, status.TimeStart) ... tStart.Format("...")
	sb.WriteString(fmt.Sprintf("期間: %s 〜 %s\n", status.TimeStart, status.TimeEnd)) // 元のフォーマットのまま
	sb.WriteString("予報レベルの割合:\n") // より分かりやすい表現
	sb.WriteString(fmt.Sprintf("  ほぼ心配なし: %.1f%%\n", status.RateNormal))
	sb.WriteString(fmt.Sprintf("  やや注意: %.1f%%\n", status.RateLittle))
	sb.WriteString(fmt.Sprintf("  注意: %.1f%%\n", status.RatePainful))
	sb.WriteString(fmt.Sprintf("  警戒: %.1f%%\n", status.RateBad))
	return sb.String(), nil
}

func (ws *weatherServiceImpl) handleSearchWeatherPoint(args map[string]interface{}) (string, error) {
	keyword, ok := args["keyword"].(string)
	if !ok {
		return "", fmt.Errorf("searchWeatherPoint: keyword がありません")
	}

	// GetWeatherPoint を直接呼ぶ (ヘルパーは単一地点取得用のため)
	weatherPointResponse, err := ws.client.GetWeatherPoint(keyword)
	if err != nil {
		log.Printf("GetWeatherPoint failed for search keyword '%s': %v", keyword, err)
		return fmt.Sprintf("「%s」の地点検索に失敗しました (エラー: %v)", keyword, err), nil // ユーザー向けメッセージは修正しない
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【「%s」の地点検索結果】\n", keyword))
	if len(weatherPointResponse.Result.Root) > 0 {
		for _, point := range weatherPointResponse.Result.Root {
			sb.WriteString(fmt.Sprintf("  📍 %s (%s)\n", point.Name, point.CityCode)) // 表示形式を少し変更
		}
	} else {
		sb.WriteString(fmt.Sprintf("  「%s」に該当する地点は見つかりませんでした\n", keyword)) // ユーザー向けメッセージは修正しない
	}
	return sb.String(), nil
}

func (ws *weatherServiceImpl) handleGetOtenkiAspInfo(args map[string]interface{}) (string, error) {
	cityCode, ok := args["cityCode"].(string)
	if !ok {
		return "", fmt.Errorf("getOtenkiAspInfo: cityCode がありません")
	}

	apiResponse, err := ws.client.GetOtenkiASP(cityCode)
	if err != nil {
		log.Printf("GetOtenkiASP failed: %v", err)
		return fmt.Sprintf("「%s」のASP情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", cityCode, err), nil // ユーザー向けメッセージは修正しない
	}

	jsonData, err := json.Marshal(apiResponse)
	if err != nil {
		log.Printf("Failed to marshal API response: %v", err)
		return "APIレスポンスの処理中にエラーが起きたよ… ごめんね🙏", nil // ユーザー向けメッセージは修正しない
	}

	var genericData map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.UseNumber()
	if err := decoder.Decode(&genericData); err != nil {
		log.Printf("Failed to unmarshal API response into generic map: %v", err)
		log.Printf("JSON data was: %s", string(jsonData))
		return "APIレスポンスの解析中にエラーが起きたよ… ごめんね🙏", nil // ユーザー向けメッセージは修正しない
	}

	var sb strings.Builder
	dateTimeInterface, dtOk := genericData["date_time"]
	dateTimeFormatted := "不明"
	if dtOk {
		dateTimeStr, dtStrOk := dateTimeInterface.(string)
		if dtStrOk {
			// 複数のフォーマットを試行
			formats := []string{"2006-01-02 15", time.RFC3339}
			parsed := false
			for _, format := range formats {
				t, err := time.Parse(format, dateTimeStr)
				if err == nil {
					dateTimeFormatted = t.Format("2006-01-02 15:04")
					parsed = true
					break
				}
			}
			if !parsed {
				log.Printf("Failed to parse date_time string %q with known formats", dateTimeStr)
			}
		}
	}
	sb.WriteString(fmt.Sprintf("【%s のOtenki ASP情報 (%s)】\n", cityCode, dateTimeFormatted))

	elementsRawInterface, elementsOk := genericData["elements"]
	if !elementsOk {
		sb.WriteString("  天気データが見つからなかったみたい… (elements missing)\n") // ユーザー向けメッセージは修正しない
		return sb.String(), nil
	}
	elementsRaw, elementsSliceOk := elementsRawInterface.([]interface{})
	if !elementsSliceOk || len(elementsRaw) == 0 {
		sb.WriteString("  天気データが見つからなかったみたい… (elements not a slice or empty)\n") // ユーザー向けメッセージは修正しない
		return sb.String(), nil
	}

	dataByDateStr := make(map[string][]interface{})
	allDateStrsMap := make(map[string]struct{})
	// 要素の数を動的に取得するか、期待される要素数を定義
	expectedElementCount := 8 // 元のコードに基づき8とする

	for elemIndex, elemInterface := range elementsRaw {
		elemMap, ok := elemInterface.(map[string]interface{})
		if !ok { continue }
		recordsRaw, recordsOk := elemMap["records"].(map[string]interface{})
		if !recordsOk { continue }

		for dateKeyStr, recordValue := range recordsRaw {
			// RFC3339形式の日付時刻文字列をパース
			t, err := time.Parse(time.RFC3339, dateKeyStr)
			if err != nil {
				log.Printf("Failed to parse date key string %q: %v", dateKeyStr, err)
				continue
			}
			dateStrYYYYMMDD := t.Format("20060102") // YYYYMMDD形式に変換

			if _, ok := dataByDateStr[dateStrYYYYMMDD]; !ok {
				// スライスのサイズを期待される要素数で初期化
				dataByDateStr[dateStrYYYYMMDD] = make([]interface{}, expectedElementCount)
			}
			// インデックスが範囲内か確認
			if elemIndex < expectedElementCount {
				dataByDateStr[dateStrYYYYMMDD][elemIndex] = recordValue
			} else {
				log.Printf("Warning: Element index %d out of bounds (expected < %d)", elemIndex, expectedElementCount)
			}
			allDateStrsMap[dateStrYYYYMMDD] = struct{}{}
		}
	}

	var sortedDateStrs []string
	for dateStr := range allDateStrsMap {
		sortedDateStrs = append(sortedDateStrs, dateStr)
	}
	sort.Strings(sortedDateStrs)

	header := "| 日付 | 天気 | 降水% | 最高℃ | 最低℃ | 風速m/s | 風向 | 気圧Lv | 湿度% |"
	separator := "|:---|:---|:----:|:-----:|:-----:|:------:|:--:|:------:|:----:|"
	sb.WriteString(header + "\n")
	sb.WriteString(separator + "\n")

	// 要素名と期待されるインデックスのマッピング
	elementIndices := map[string]int{
		"天気": 0, "降水%": 1, "最高℃": 2, "最低℃": 3,
		"風速m/s": 4, "風向": 5, "気圧Lv": 6, "湿度%": 7,
	}
	// 表示する列の順序
	columnOrder := []string{"天気", "降水%", "最高℃", "最低℃", "風速m/s", "風向", "気圧Lv", "湿度%"}

	for _, dateStr := range sortedDateStrs {
		dateFormatted := "-"
		t, err := time.Parse("20060102", dateStr)
		if err == nil { dateFormatted = t.Format("01/02") }

		row := []string{dateFormatted}
		dateData, ok := dataByDateStr[dateStr]
		// データが存在し、かつ期待される要素数を持っているか確認
		if !ok || len(dateData) < expectedElementCount {
			log.Printf("Warning: Data missing or incomplete for date %s. Found %d elements, expected %d.", dateStr, len(dateData), expectedElementCount)
			// データが不足している場合はハイフンで埋める
			for i := 0; i < len(columnOrder); i++ { row = append(row, "-") }
		} else {
			for _, columnName := range columnOrder {
				elemIndex, indexOk := elementIndices[columnName]
				valueStr := "-"
				// インデックスが存在し、かつデータスライス内で有効か確認
				if indexOk && elemIndex < len(dateData) {
					value := dateData[elemIndex]
					// value が nil の場合も switch で処理するため、if value != nil は削除
					switch v := value.(type) { // switch 文開始
					case nil:
						// value が nil の場合は "-" のまま
					case string:
						if columnName == "天気" {
								// --- 天気コードを絵文字に変換 (ヘルパー関数使用) ---
								valueStr = getWeatherEmoji(v)
								// --- ここまで変換処理 ---
							} else {
								valueStr = v // 天気以外はそのまま
							}
						case json.Number:
							// 天気コードが Number で来る可能性も考慮
							if columnName == "天気" {
								// --- 天気コードを絵文字に変換 (ヘルパー関数使用) ---
								valueStr = getWeatherEmoji(v)
								// --- ここまで変換処理 ---
							} else {
								// 天気以外は数値として処理
								floatVal, err := v.Float64()
								if err == nil {
									// 整数かどうかでフォーマットを分ける
									if columnName == "降水%" || columnName == "湿度%" || columnName == "気圧Lv" || columnName == "風向" {
										if floatVal == float64(int(floatVal)) {
											valueStr = strconv.Itoa(int(floatVal))
										} else {
											valueStr = fmt.Sprintf("%.1f", floatVal) // 小数第一位まで
										}
									} else { // 最高/最低気温、風速など
										valueStr = fmt.Sprintf("%.1f", floatVal) // 小数第一位まで
									}
								} else {
									valueStr = v.String() // Float変換失敗時は元のNumber文字列
								}
							}
						case float64: // json.Number を使わない場合
							if columnName == "天気" {
								// --- 天気コードを絵文字に変換 (ヘルパー関数使用) ---
								valueStr = getWeatherEmoji(v)
								// --- ここまで変換処理 ---
							} else {
								// 天気以外は数値として処理
								if columnName == "降水%" || columnName == "湿度%" || columnName == "気圧Lv" || columnName == "風向" {
									if v == float64(int(v)) {
										valueStr = strconv.Itoa(int(v))
									} else {
										valueStr = fmt.Sprintf("%.1f", v)
									}
								} else {
									valueStr = fmt.Sprintf("%.1f", v)
								}
							}
						default:
							if columnName == "天気" {
								valueStr = getWeatherEmoji(v)
							} else {
								valueStr = fmt.Sprintf("%v", v)
							}
						} // switch v := value.(type) の閉じ括弧
				} else { // if indexOk && elemIndex < len(dateData) の else 節
					log.Printf("Warning: Index for column %q (%d) is invalid or out of bounds for date %s", columnName, elemIndex, dateStr)
				} // if indexOk && elemIndex < len(dateData) の閉じ括弧
				row = append(row, valueStr)
			}
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return sb.String(), nil
}
