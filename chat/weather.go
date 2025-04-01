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

// weatherEmojiMap は天気コードを絵文字にマッピングします。
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
			Description: "指定されたキーワード（地名など）で場所を検索し、地点コードなどの情報を取得します。",
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
			Description: "指定された地点コードのOtenki ASP情報を取得します。",
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
		return "", fmt.Errorf("不明な天気関数が呼び出されました: %s", fn.Name)
	}
}

func (ws *weatherServiceImpl) handleGetWeather(args map[string]interface{}) (string, error) {
	location, ok := args["location"].(string)
	if !ok {
		return "", fmt.Errorf("getWeather: location がありません")
	}

	weatherPoint, err := ws.client.GetWeatherPoint(location)
	if err != nil {
		log.Printf("GetWeatherPoint failed for weather: %v", err)
		return fmt.Sprintf("「%s」の地点情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", location, err), nil
	}
	if len(weatherPoint.Result.Root) == 0 {
		return fmt.Sprintf("「%s」って場所が見つからなかったみたい…🤔", location), nil
	}
	cityCode := weatherPoint.Result.Root[0].CityCode

	weatherStatus, err := ws.client.GetWeatherStatus(cityCode)
	if err != nil {
		log.Printf("GetWeatherStatus failed: %v", err)
		return fmt.Sprintf("「%s」(%s)の天気情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", location, cityCode, err), nil
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
			// --- 天気コードを絵文字に変換 ---
			var weatherStr string // string 型として宣言
			weatherCodeStr := string(status.Weather) // WeatherEnum を string に変換
			weatherCodeInt, err := strconv.Atoi(weatherCodeStr) // string を int に変換
			if err == nil {
				// 100の位で丸める
				simplifiedCode := (weatherCodeInt / 100) * 100
				if emoji, okEmoji := weatherEmojiMap[simplifiedCode]; okEmoji {
					weatherStr = emoji // マップにあれば絵文字に置き換え
				} else {
					// マップにない場合は元のコード（文字列）を表示
					weatherStr = weatherCodeStr
					log.Printf("Weather code %d (simplified %d) not found in emoji map for getWeather. Using original code string %q.", weatherCodeInt, simplifiedCode, weatherCodeStr)
				}
			} else {
				// int 変換に失敗した場合 (通常は起こらないはず)
				weatherStr = weatherCodeStr // 元の文字列を使用
				log.Printf("Could not convert weather enum string %q to int in getWeather: %v", weatherCodeStr, err)
			}
			// --- ここまで変換処理 ---
			sb.WriteString(fmt.Sprintf("  %s: %s, %shPa, %s\n",
				status.Time, tempStr, status.Pressure, weatherStr)) // weatherStr は常に string
		}
	} else {
		sb.WriteString("  今日のデータはないみたい…\n")
	}
	return sb.String(), nil
}

func (ws *weatherServiceImpl) handleGetPainStatus(args map[string]interface{}) (string, error) {
	location, ok := args["location"].(string)
	if !ok {
		return "", fmt.Errorf("getPainStatus: location がありません")
	}

	weatherPoint, err := ws.client.GetWeatherPoint(location)
	if err != nil {
		log.Printf("GetWeatherPoint failed for headache: %v", err)
		return fmt.Sprintf("「%s」の地点情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", location, err), nil
	}
	if len(weatherPoint.Result.Root) == 0 {
		return fmt.Sprintf("「%s」って場所が見つからなかったみたい…🤔", location), nil
	}
	cityCode := weatherPoint.Result.Root[0].CityCode
	areaCode := ""
	if len(cityCode) >= 2 {
		areaCode = cityCode[:2]
	}
	if areaCode == "" {
		log.Printf("Failed to extract area code from city code: %s", cityCode)
		return fmt.Sprintf("「%s」の地域コードがわからなかった… ごめんね🙏", location), nil
	}

	painStatus, err := ws.client.GetPainStatus(areaCode, &cityCode)
	if err != nil {
		log.Printf("GetPainStatus failed: %v", err)
		return fmt.Sprintf("「%s」(%s)の頭痛情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", location, cityCode, err), nil
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
	return sb.String(), nil
}

func (ws *weatherServiceImpl) handleSearchWeatherPoint(args map[string]interface{}) (string, error) {
	keyword, ok := args["keyword"].(string)
	if !ok {
		return "", fmt.Errorf("searchWeatherPoint: keyword がありません")
	}

	weatherPoint, err := ws.client.GetWeatherPoint(keyword)
	if err != nil {
		log.Printf("GetWeatherPoint failed for search: %v", err)
		return fmt.Sprintf("「%s」の地点検索に失敗しちゃった… ごめんね🙏 (エラー: %v)", keyword, err), nil
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
		return fmt.Sprintf("「%s」のASP情報の取得に失敗しちゃった… ごめんね🙏 (エラー: %v)", cityCode, err), nil
	}

	jsonData, err := json.Marshal(apiResponse)
	if err != nil {
		log.Printf("Failed to marshal API response: %v", err)
		return "APIレスポンスの処理中にエラーが起きたよ… ごめんね🙏", nil
	}

	var genericData map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.UseNumber()
	if err := decoder.Decode(&genericData); err != nil {
		log.Printf("Failed to unmarshal API response into generic map: %v", err)
		log.Printf("JSON data was: %s", string(jsonData))
		return "APIレスポンスの解析中にエラーが起きたよ… ごめんね🙏", nil
	}

	var sb strings.Builder
	dateTimeInterface, dtOk := genericData["date_time"]
	dateTimeFormatted := "不明"
	if dtOk {
		dateTimeStr, dtStrOk := dateTimeInterface.(string)
		if dtStrOk {
			// 複数のフォーマットを試す
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
		sb.WriteString("  天気データが見つからなかったみたい… (elements missing)\n")
		return sb.String(), nil
	}
	elementsRaw, elementsSliceOk := elementsRawInterface.([]interface{})
	if !elementsSliceOk || len(elementsRaw) == 0 {
		sb.WriteString("  天気データが見つからなかったみたい… (elements not a slice or empty)\n")
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
					if value != nil {
						switch v := value.(type) {
						case string:
							if columnName == "天気" {
								weatherCodeInt, err := strconv.Atoi(v)
								emoji := "?"
								if err == nil {
									// 100の位で丸める (例: 101 -> 100)
									simplifiedCode := (weatherCodeInt / 100) * 100
									if e, okEmoji := weatherEmojiMap[simplifiedCode]; okEmoji {
										emoji = e
									} else {
										// マップにない場合は元のコード（文字列）を表示
										emoji = v
										log.Printf("Weather code %d (simplified %d) not found in emoji map.", weatherCodeInt, simplifiedCode)
									}
								} else {
									// 数値に変換できない場合は元の文字列を表示
									emoji = v
									log.Printf("Could not convert weather code string %q to int: %v", v, err)
								}
								valueStr = emoji
							} else {
								valueStr = v
							}
						case json.Number:
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
						case float64: // json.Number を使わない場合
							if columnName == "降水%" || columnName == "湿度%" || columnName == "気圧Lv" || columnName == "風向" {
								if v == float64(int(v)) {
									valueStr = strconv.Itoa(int(v))
								} else {
									valueStr = fmt.Sprintf("%.1f", v)
								}
							} else {
								valueStr = fmt.Sprintf("%.1f", v)
							}
						default:
							valueStr = fmt.Sprintf("%v", v) // その他の型はそのまま文字列化
						}
					}
				} else {
					log.Printf("Warning: Index for column %q (%d) is invalid or out of bounds for date %s", columnName, elemIndex, dateStr)
				}
				row = append(row, valueStr)
			}
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return sb.String(), nil
}
