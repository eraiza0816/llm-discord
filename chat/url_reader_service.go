package chat

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/generative-ai-go/genai"
)

// URLReaderService はURLの内容を取得し、テキストを抽出するサービスです。
type URLReaderService interface {
	GetURLContentAsText(url string) (string, error)
	GetURLReaderFunctionDeclaration() *genai.FunctionDeclaration
}

type urlReaderServiceImpl struct{}

// NewURLReaderService はURLReaderServiceの新しいインスタンスを作成します。
func NewURLReaderService() URLReaderService {
	return &urlReaderServiceImpl{}
}

// GetURLContentAsText は指定されたURLからHTMLを取得し、主要なテキストコンテンツを抽出します。
// 抽出されるテキストは最大2000文字に制限されます。
func (s *urlReaderServiceImpl) GetURLContentAsText(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("URLの取得に失敗しました (%s): %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("URLの取得に失敗しました (%s): ステータスコード %d", url, res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("HTMLのパースに失敗しました (%s): %w", url, err)
	}

	// 主要なコンテンツが含まれそうなタグからテキストを抽出
	// scriptタグとstyleタグの内容は除外
	doc.Find("script, style").Remove()
	// bodyタグ内のテキストを取得し、トリムして改行をスペースに置換
	text := strings.TrimSpace(doc.Find("body").Text())
	text = strings.ReplaceAll(text, "\n", " ")
	// 連続するスペースを1つにまとめる
	text = strings.Join(strings.Fields(text), " ")


	const maxTextLength = 2000
	if len(text) > maxTextLength {
		// runeでスライスしてマルチバイト文字が壊れないようにする
		runes := []rune(text)
		text = string(runes[:maxTextLength]) + "..."
	}

	if text == "" {
		return "コンテンツが見つかりませんでした。", nil
	}

	return text, nil
}

// GetURLReaderFunctionDeclaration は get_url_content 関数の FunctionDeclaration を返します。
func (s *urlReaderServiceImpl) GetURLReaderFunctionDeclaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "get_url_content",
		Description: "指定されたURLの主要なテキストコンテンツを取得します。",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"url": {
					Type:        genai.TypeString,
					Description: "内容を取得するURL。",
				},
			},
			Required: []string{"url"},
		},
	}
}
