package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type AMeDAS struct {
	BaseURL string
}

func NewAMeDAS() *AMeDAS {
	return &AMeDAS{
		BaseURL: "https://www.jma.go.jp/bosai/amedas/data",
	}
}

func (a *AMeDAS) GetLatestTime() string {
	latestTimeURL := fmt.Sprintf("%s/latest_time.txt", a.BaseURL)
	resp, err := http.Get(latestTimeURL)
	if err != nil {
		log.Println("Error getting latest time:", err)
		return "20250310120000"
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP error! Status code: %d\n", resp.StatusCode)
		return "20250310120000"
	}

	var latestTime string
	err = json.NewDecoder(resp.Body).Decode(&latestTime)
	if err != nil {
		// body が text なので、NewDecoder は使えない
		// log.Println("Error decoding latest time:", err)
		// return "20250310120000"
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		if err != nil {
			log.Println("Error reading latest time:", err)
			return "20250310120000"
		}
		latestTime = string(buf[:n])
	}

	//latestTime = string(resp.Body)
	//latestTime = "2025-03-10T17:10:00+09:00" // TODO: fix

	// ISO 8601形式の文字列をパース
	t, err := time.Parse(time.RFC3339, latestTime)
	if err != nil {
		log.Println("Error parsing time:", err)
		return "20250310120000"
	}
	return t.Format("20060102150405")
}

func (a *AMeDAS) GetWeather(stationID string) string {
	latestTime := a.GetLatestTime()
	if latestTime == "20250310120000" {
		return "最新の天気データが取得できませんでした。"
	}

	weatherURL := fmt.Sprintf("%s/map/%s.json", a.BaseURL, latestTime)

	resp, err := http.Get(weatherURL)
	if err != nil {
		log.Println("Error getting weather data:", err)
		return "最新の天気データが取得できませんでした。"
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return "最新の天気データが見つかりませんでした。しばらくしてからもう一度試してください。"
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("天気情報の取得に失敗しました (HTTP %d)", resp.StatusCode)
	}

	var weatherData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&weatherData)
	if err != nil {
		log.Println("Error decoding weather data:", err)
		return "天気情報の取得に失敗しました。"
	}

	stationData, ok := weatherData[stationID].(map[string]interface{})
	if !ok {
		return "指定した地域の天気情報が見つかりません。"
	}

	// `temp` がリストの場合の処理
	tempData := stationData["temp"]
	var temp string
	switch v := tempData.(type) {
	case map[string]interface{}:
		tempValue, ok := v["value"].(float64)
		if ok {
			temp = fmt.Sprintf("%.1f", tempValue)
		} else {
			temp = "不明"
		}
	default:
		temp = "不明"
	}

	// `precipitation1h` の処理
	precipData := stationData["precipitation1h"]
	var precip string
	switch v := precipData.(type) {
	case map[string]interface{}:
		precipValue, ok := v["value"].(float64)
		if ok {
			precip = fmt.Sprintf("%.1f", precipValue)
		} else {
			precip = "不明"
		}
	default:
		precip = "不明"
	}

	// `wind` の処理
	windData := stationData["wind"]
	var windSpeed string
	switch v := windData.(type) {
	case map[string]interface{}:
		windSpeedValue, ok := v["value"].(float64)
		if ok {
			windSpeed = fmt.Sprintf("%.1f", windSpeedValue)
		} else {
			windSpeed = "不明"
		}
	default:
		windSpeed = "不明"
	}

	return fmt.Sprintf("アメダス観測地点(%s)の天気情報:\n 気温: %s°C\n 降水量: %s mm\n 風速: %s m/s", stationID, temp, precip, windSpeed)
}

func main() {
	amedas := NewAMeDAS()
	stationID := "66206" // 東京
	weatherInfo := amedas.GetWeather(stationID)
	fmt.Println(weatherInfo)
}
