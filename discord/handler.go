package discord

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/kballard/go-shellquote"

	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/loader"
	"github.com/eraiza0816/llm-discord/utils"
)

var (
	dg          *discordgo.Session
	llmConfig   *config.LLMConfig
	modelConfig *loader.ModelConfig
	chatLog     *log.Logger
)

func init() {
	// .envファイルの読み込み
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	// 環境変数から設定を読み込む
	llmConfig = config.LoadConfig()

	// JSONファイルからモデル設定を読み込む
	modelConfig, err = loader.LoadModelConfig("json/model.json")
	if err != nil {
		fmt.Println("Error loading model config:", err)
		panic(err)
	}

	// ログ設定
	logFile, err := os.OpenFile("log/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		panic(err)
	}
	chatLog = log.New(logFile, "Chat: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Run() {
	// Discord botのセッションを作成
	var err error
	dg, err = discordgo.New("Bot " + llmConfig.DiscordToken)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Intentを設定
	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent

	// メッセージハンドラを設定
	dg.AddHandler(messageCreate)

	// Discordに接続
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection:", err)
		return
	}

	// Botが起動したことを通知
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// セッションを閉じる
	dg.Close()
}

// messageCreate は、メッセージが作成されたときに呼び出されます。
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Bot自身からのメッセージは無視
	if m.Author.ID == s.State.User.ID {
		return
	}

	// メッセージの内容を確認
	content := m.Content

	// チャンネルIDを確認
	channelID := m.ChannelID

	// contentが空の場合は処理を中断
	if content == "" {
		return
	}

	// OpenAI APIを呼び出す
	resp, err := chat.CallOpenAI(s, m, llmConfig, modelConfig, content)
	if err != nil {
		fmt.Println("Error calling OpenAI:", err)
		return
	}

	// OpenAIからの応答をDiscordに送信
	if resp != "" {
		// ログ出力
		chatLog.Printf("ChannelID: %s, User: %s, Message: %s, Response: %s\n", channelID, m.Author.Username, content, resp)

		// 応答を分割して送信
		// 1000文字を超える場合は分割
		embed := &discordgo.MessageEmbed{
			Description: resp,
			Fields:      []*discordgo.MessageEmbedField{}, // 空のスライスで初期化
		}

		// フィールドを追加
		splitAndAddFields(embed, "Response", resp)

		// embedを送信
		_, err = s.ChannelMessageSendEmbed(channelID, embed)
		if err != nil {
			fmt.Println("Error sending embed:", err)
			return
		}
	}

	// メッセージを解析してコマンドを実行
	if strings.HasPrefix(content, "!") {
		// コマンドを解析
		parts := strings.SplitN(content[1:], " ", 2)
		command := parts[0]
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}

		// evalコマンドの場合
		if command == "eval" {
			// 実行を許可されたユーザーIDを確認
			if m.Author.ID != llmConfig.AdminUserID {
				s.ChannelMessageSend(m.ChannelID, "You are not authorized to use this command.")
				return
			}

			// 引数を解析
			quoted, err := shellquote.Split(args)
			if err != nil {
				fmt.Println("Error splitting arguments:", err)
				s.ChannelMessageSend(m.ChannelID, "Error splitting arguments.")
				return
			}

			// 引数が2つ以上ある場合はエラー
			if len(quoted) < 2 {
				s.ChannelMessageSend(m.ChannelID, "Usage: !eval <channelID> <message>")
				return
			}

			// チャンネルIDとメッセージを取得
			targetChannelID := quoted[0]
			targetMessage := quoted[1]

			// メッセージを送信
			_, err = s.ChannelMessageSend(targetChannelID, targetMessage)
			if err != nil {
				fmt.Println("Error sending message:", err)
				s.ChannelMessageSend(m.ChannelID, "Error sending message.")
				return
			}

			// 応答を送信
			s.ChannelMessageSend(m.ChannelID, "Message sent.")
		}

		// weatherコマンドの場合
		if command == "weather" {
			// 都市名を取得
			city := args

			// 天気情報を取得
			weatherInfo, err := GetWeather(city)
			if err != nil {
				fmt.Println("Error getting weather:", err)
				s.ChannelMessageSend(m.ChannelID, "Error getting weather.")
				return
			}

			// embedを作成
			embed := &discordgo.MessageEmbed{
				Title:       "Weather in " + city,
				Description: "Weather information for " + city,
				Fields:    []*discordgo.MessageEmbedField{}, // 空のスライスで初期化
			}

			// フィールドを追加
			splitAndAddFields(embed, "Temperature", strconv.FormatFloat(weatherInfo.Temperature, 'f', 2, 64)+"°C")
			splitAndAddFields(embed, "Condition", weatherInfo.Condition)
			splitAndAddFields(embed, "Humidity", strconv.FormatFloat(weatherInfo.Humidity, 'f', 2, 64)+"%")
			splitAndAddFields(embed, "Wind Speed", strconv.FormatFloat(weatherInfo.WindSpeed, 'f', 2, 64)+" m/s")

			// embedを送信
			_, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
			if err != nil {
				fmt.Println("Error sending embed:", err)
				return
			}
		}
	}
}

// 指定された長さでメッセージを分割する関数
func splitMessage(message string, maxLength int) []string {
	var messages []string
	for i := 0; i < len(message); i += maxLength {
		end := i + maxLength
		if end > len(message) {
			end = len(message)
		}
		messages = append(messages, message[i:end])
	}
	return messages
}

func splitAndAddFields(embed *discordgo.MessageEmbed, title string, value string) {
	const maxLen = 1024
	// 指定文字数で分割（最後まで漏れなく）
	for i := 0; i < len(value); i += maxLen {
		end := i + maxLen
		if end > len(value) {
			end = len(value)
		}
		fieldTitle := title
		if i > 0 {
			fieldTitle = title + "（続き）"
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fieldTitle,
			Value:  value[i:end],
			Inline: false,
		})
	}
}
