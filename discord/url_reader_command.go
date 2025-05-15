package discord

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
)

func URLReaderCommand(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service) {
	// オプションからURLを取得
	var url string
	if len(i.ApplicationCommandData().Options) > 0 {
		option := i.ApplicationCommandData().Options[0]
		if option.Type == discordgo.ApplicationCommandOptionString {
			url = option.StringValue()
		}
	}

	if url == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URLを指定してください。",
			},
		})
		return
	}

	// curlコマンドでHTMLを取得
	cmd := exec.Command("curl", "-L", url) // -L オプションでリダイレクトを追跡
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("Error executing curl for %s: %v\nStderr: %s", url, err, stderr.String())
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("URLの取得に失敗しました: %v", err),
			},
		})
		return
	}

	htmlContent := out.String()

	// goqueryでHTMLをパースしてテキストを抽出
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML for %s: %v", url, err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("HTMLの解析に失敗しました: %v", err),
			},
		})
		return
	}

	// 主要なテキストコンテンツを抽出 (bodyタグ内のテキストを基本とする)
	// より洗練された抽出ロジックも可能 (例: <article>, <main> タグ、<p> タグのみなど)
	var textBuilder strings.Builder
	doc.Find("body").Each(func(index int, item *goquery.Selection) {
		textBuilder.WriteString(item.Text()) // item.Text() は子要素のテキストも再帰的に取得
	})
	extractedText := strings.TrimSpace(textBuilder.String())

	if extractedText == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URLからテキストを抽出できませんでした。",
			},
		})
		return
	}

	// テキストが長すぎる場合は切り詰める (Gemini APIの制限を考慮)
	// ここでは単純に文字数で制限するが、トークン数で制限する方がより正確
	const maxTextLength = 10000 // 例: 10000文字
	if len(extractedText) > maxTextLength {
		extractedText = extractedText[:maxTextLength] + "..."
	}

	// LLMに要約を依頼
	// スレッドIDの取得 (ChatCommandと同様のロジック)
	var threadID string
	if i.ChannelID != "" {
		ch, err := s.State.Channel(i.ChannelID)
		if err != nil {
			ch, err = s.Channel(i.ChannelID)
			// エラー処理は省略。実際には適切に処理する
		}
		if ch != nil && ch.IsThread() {
			threadID = ch.ID
		} else {
			threadID = i.ChannelID
		}
	} else if i.Message != nil && i.Message.ChannelID != "" {
		// 同様のロジック
		threadID = i.Message.ChannelID
	}


	// 要約用のプロンプトを作成
	prompt := fmt.Sprintf("以下のウェブサイトの内容を要約してください。\n\nURL: %s\n\n抽出されたテキスト:\n%s\n\n要約:", url, extractedText)

	// ChatServiceを使用してGemini APIを呼び出す
	// 履歴は保存しない一時的なチャットとして扱うか、専用の履歴タイプを設けるか検討
	// ここでは履歴なしで直接呼び出すことを想定 (ChatServiceのメソッドに依存)
	// もしChatServiceに直接プロンプトを渡して応答を得るメソッドがない場合、
	// ChatServiceを拡張するか、chatパッケージ内の低レベル関数を利用する必要がある。
	// ここでは、GenerateContentWithRetryが直接プロンプトを受け付けると仮定。
	// 実際には、chatSvc.GenerateResponse(threadID, prompt, userID) のような形になる可能性が高い。
	// userIDの取得も必要。
	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}


	// chatSvc.GenerateContentWithRetry を使う場合、履歴コンテキストが必要になる。
	// 今回はURL要約なので、独立したリクエストとして扱いたい。
	// chat.GenerateContent 単体で呼び出すか、ChatServiceに専用メソッドを追加する。
	// ここでは、chat.GenerateContent を直接呼び出すことを試みる。
	// ただし、chat.GenerateContent は chat パッケージ内の非公開関数の可能性がある。
	// chatSvc が提供するインターフェースを使うのが望ましい。
	// 仮に chatSvc.SummarizeText(text string) のようなメソッドがあると想定。
	// もしそのようなメソッドがなければ、chatSvc.ProcessMessage のような汎用メソッドを使う。

	// 応答を待っていることをユーザーに通知
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})


	// ChatServiceのGetResponseメソッドを使用する
	// GetResponseは履歴を保存する。URL要約専用の履歴タイプを設けるか、
	// 履歴保存しないメソッドをChatServiceに追加するのが望ましいが、今回は既存メソッドを利用。
	// GetResponseの引数に合わせる: userID, threadID, username, message, timestamp, prompt
	username := ""
	if i.Member != nil && i.Member.User != nil {
		username = i.Member.User.Username
	} else if i.User != nil {
		username = i.User.Username
	}
	timestamp := time.Now().Format(time.RFC3339) // 現在時刻をRFC3339形式で

	// GetResponseの最後の引数 `prompt` は、システムプロンプトやカスタムプロンプトに該当する。
	// 今回の要約タスクでは、`prompt` 変数に格納した指示全体を `message` として渡し、
	// システムプロンプトは空にするか、汎用的なものを渡す。
	// ここでは、`prompt` 変数（要約指示）を `message` として扱い、システムプロンプトは空とする。
	systemPrompt := "" // または "あなたは役立つアシスタントです。" のような汎用プロンプト

	responseContent, _, _, err := chatSvc.GetResponse(userID, threadID, username, prompt, timestamp, systemPrompt)

	if err != nil {
		log.Printf("Error generating summary for %s: %v", url, err)
		errorMsg := fmt.Sprintf("要約の生成中にエラーが発生しました: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &errorMsg,
		})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &responseContent,
	})
}
