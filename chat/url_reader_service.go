package chat

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/generative-ai-go/genai"
)

// URLReaderService はURLの内容を読み取り、テキストを抽出するサービスです。
type URLReaderService struct{}

// NewURLReaderService は新しいURLReaderServiceのインスタンスを作成します。
func NewURLReaderService() *URLReaderService {
	return &URLReaderService{}
}

// GetURLContentAndSummarize は指定されたURLの内容を取得し、主要なテキストを抽出して返します。
// GeminiのFunction Callingから呼び出されることを想定しています。
// 引数: urlString (読み取るURL)
// 戻り値: 抽出されたテキスト、エラー
func (s *URLReaderService) GetURLContentAsText(urlString string) (string, error) {
	if urlString == "" {
		return "", fmt.Errorf("URLが指定されていません")
	}

	// curlコマンドでHTMLを取得
	cmd := exec.Command("curl", "-L", urlString)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("Error executing curl for %s: %v\nStderr: %s", urlString, err, stderr.String())
		return "", fmt.Errorf("URLの取得に失敗しました: %w", err)
	}

	htmlContent := out.String()

	// goqueryでHTMLをパースしてテキストを抽出
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML for %s: %v", urlString, err)
		return "", fmt.Errorf("HTMLの解析に失敗しました: %w", err)
	}

	var textBuilder strings.Builder
	// より具体的に本文と思われる箇所を抽出するセレクタの例
	// "article, main, [role='main'], .post-content, .entry-content"
	// ここではシンプルにbody全体から取得
	doc.Find("body").Each(func(index int, item *goquery.Selection) {
		// スクリプトやスタイルタグの内容は除外
		item.Find("script, style, noscript, iframe, nav, footer, header, aside").Remove()
		textBuilder.WriteString(item.Text())
	})
	extractedText := strings.TrimSpace(textBuilder.String())

	if extractedText == "" {
		return "", fmt.Errorf("URLからテキストを抽出できませんでした")
	}

	// テキストが長すぎる場合は切り詰める (Function Callingの結果として返すには適切な長さに)
	const maxTextLength = 5000 // Function Callingの結果としては短めにする
	if len(extractedText) > maxTextLength {
		extractedText = extractedText[:maxTextLength] + "..."
	}

	return extractedText, nil
}

// GetURLReaderFunctionDeclaration はGeminiのFunction Calling用の関数宣言を返します。
func GetURLReaderFunctionDeclaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name: "get_url_content",
		Description: "ユーザーが提示したURLや、会話の流れで言及されたウェブページの内容を理解する必要がある場合に、そのURLの主要なテキストコンテンツを取得します。例えば、「この記事を読んで」や「このサイトには何が書いてある？」のようなリクエストに応答する際に使用します。",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"url": {
					Type:        genai.TypeString,
					Description: "内容を読み取りたいウェブページの完全なURL (例: https://example.com/article)。",
				},
			},
			Required: []string{"url"},
		},
	}
}
