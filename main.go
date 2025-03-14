package main

import (
	"context"
	"log"
	"os"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Config struct {
	DiscordBotToken string
	GeminiAPIKey    string
}

type JsonLoader struct {
	jsonFile string
	data     map[string]interface{}
	logger   *log.Logger
}

type JsonLoad struct {
	*JsonLoader
	logger *log.Logger
}

type ModelLoad struct {
	*JsonLoader
	logger *log.Logger
}

type Chat struct {
	token          string
	model          string
	defaultPrompt  string
	logger         *log.Logger
	userHistories  map[string][]string
	userHistoriesMutex sync.Mutex
	genaiModel *genai.GenerativeModel
	genaiClient *genai.Client
}

var (
	chat   *Chat
	config Config
	logger *log.Logger
	errLogger *log.Logger
	jsonLoadInstance *JsonLoad
	modelLoadInstance *ModelLoad
	dg *discordgo.Session
)

func main() {
	appLogFile, err := os.OpenFile("log/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening app log file: %v", err)
		return
	}
	defer appLogFile.Close()

	errorLogFile, err := os.OpenFile("log/error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening error log file: %v", err)
		return
	}
	defer errorLogFile.Close()

	logger = log.New(appLogFile, "bot: ", log.LstdFlags)
	errLogger = log.New(errorLogFile, "error: ", log.LstdFlags)

	err = godotenv.Load(".env")
	if err != nil {
		errLogger.Fatalf("Error loading .env file: %v", err)
		return
	}

	config = Config{
		DiscordBotToken: os.Getenv("DISCORD_BOT_TOKEN"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
	}

	if config.DiscordBotToken == "" || config.GeminiAPIKey == "" {
		logger.Fatalf("DISCORD_BOT_TOKEN or GEMINI_API_KEY is not set in .env file or environment variables")
		return
	}

	jsonLoadInstance, err = NewJsonLoad()
	if err != nil {
		log.Fatalf("Error creating JsonLoad instance: %v", err)
		return
	}

	modelLoadInstance, err = NewModelLoad()
	if err != nil {
		log.Fatalf("Error creating ModelLoad instance: %v", err)
		return
	}

	chat, err = NewChat(config.GeminiAPIKey, modelLoadInstance.GetModelName(), modelLoadInstance.GetPromptDefault())
	if err != nil {
		logger.Fatalf("Error creating chat instance: %v", err)
		return
	}
	defer func() {
		if chat != nil {
			chat.Close()
		}
	}()

	dg, err = discordgo.New("Bot " + config.DiscordBotToken)
	if err != nil {
		logger.Fatalf("Error creating Discord session: %v", err)
		return
	}
	defer dg.Close()

	dg.AddHandler(messageCreate)
	dg.AddHandler(onReady)
	dg.AddHandler(interactionCreate) // スラッシュコマンドのハンドラを登録

	dg.Identify = discordgo.Identify{
		Token: config.DiscordBotToken,
		Intents: discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds,
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "chat",
			Description: "おしゃべりしようよ dev version",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "メッセージ",
					Required:    true,
				},
			},
		},
		{
			Name:        "reset",
			Description: "あなたとのチャット履歴をリセット",
		},
	}
	
	// 接続の開始
	err = dg.Open()
	if err != nil {
		logger.Fatalf("Error opening connection: %v", err)
		return
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}


	logger.Println("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}
	logger.Println("Successfully deleted all commands.")
}

func NewChat(token string, model string, defaultPrompt string) (*Chat, error) {
	logger := log.New(os.Stdout, "chat: ", log.LstdFlags)
	logger.Printf("Entering NewChat function with args: token=%s, model=%s, defaultPrompt=%s", token, model, defaultPrompt)
	defer logger.Println("Exiting NewChat function")

	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(token))
	if err != nil {
		errLogger.Printf("Error creating Gemini client: %v", err)
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

    genaiModel := client.GenerativeModel(model)

    return &Chat{
        token:          token,
        model:          model,
        defaultPrompt:  defaultPrompt,
        logger:         logger,
        userHistories:  make(map[string][]string),
        genaiModel: genaiModel,
        genaiClient: client,
    }, nil
}

func (c *Chat) GetResponse(userID string, username string, message string, timestamp string) (string, float64, error) {
	c.logger.Printf("Entering GetResponse function with args: userID=%s, username=%s, message=%s, timestamp=%s", userID, username, message, timestamp)
	defer c.logger.Println("Exiting GetResponse function")

	c.userHistoriesMutex.Lock()
	defer c.userHistoriesMutex.Unlock()

    if strings.ToLower(strings.TrimSpace(message)) == "/reset" {
        c.ClearHistory(userID)
        return "チャット履歴をリセットしました！", 0.0, nil
    }

    historyText := strings.Join(c.userHistories[userID], "\n")
    fullInput := fmt.Sprintf("%s\n%s\n%s: %s", timestamp, historyText, username, message)

    if historyText == "" {
        fullInput = fmt.Sprintf("%s\n%s: %s", timestamp, username, message)
    }

    start := time.Now()

    ctx := context.Background()

    prompt := fmt.Sprintf("%s\n%s", c.defaultPrompt, fullInput) // デフォルトプロンプトとユーザー入力を結合
    resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
        c.logger.Printf("Gemini API error: %v", err)
        return fmt.Sprintf("エラーが発生しました。以下の内容をコピペして eraiza0816まで。\n```%v```", err), 0.0, err
    }

    elapsed := time.Since(start).Seconds() * 1000

    responseText := getResponseText(resp)

    c.userHistories[userID] = append(c.userHistories[userID], fmt.Sprintf("%s: %s", username, message))
    c.userHistories[userID] = append(c.userHistories[userID], fmt.Sprintf("Bot: %s", responseText))

    if len(c.userHistories[userID]) > 15 {
        c.userHistories[userID] = c.userHistories[userID][len(c.userHistories[userID])-15:]
    }

    return responseText, elapsed, nil
}

func (c *Chat) Close() {
	c.logger.Println("Entering Close function")
	defer c.logger.Println("Exiting Close function")
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

func (c *Chat) ClearHistory(userID string) {
	c.logger.Printf("Entering ClearHistory function with args: userID=%s", userID)
	defer c.logger.Println("Exiting ClearHistory function")
	c.userHistoriesMutex.Lock()
	defer c.userHistoriesMutex.Unlock()
	delete(c.userHistories, userID)
}

func NewJsonLoader(filePath string) (*JsonLoader, error) {
	logger := log.New(os.Stdout, "json_loader: ", log.LstdFlags)
	logger.Printf("Entering NewJsonLoader function with args: filePath=%s", filePath)
	defer logger.Println("Exiting NewJsonLoader function")
	loader := &JsonLoader{
		jsonFile: filePath,
		logger:   logger,
	}

    err := loader.LoadJson()
    if err != nil {
        return nil, err
    }

    return loader, nil
}

func (jl *JsonLoader) fileCheck() error {
	jl.logger.Println("Entering fileCheck function")
	defer jl.logger.Println("Exiting fileCheck function")
	if _, err := os.Stat("json"); os.IsNotExist(err) {
		jl.logger.Println("JSON Directory is not found.")
		return errors.New("JSONディレクトリがありません")
	}

    if jl.jsonFile == "" {
        jl.logger.Println("JSON_file is None.")
        return errors.New("JSONファイルが指定されていません")
    }

    if _, err := os.Stat(jl.jsonFile); os.IsNotExist(err) {
        jl.logger.Printf("JSON file '%s' is not found.", jl.jsonFile)
        return errors.New("JSONファイルがありません")
    }

    return nil
}

func (jl *JsonLoader) LoadJson() error {
	jl.logger.Println("Entering LoadJson function")
	defer jl.logger.Println("Exiting LoadJson function")
	err := jl.fileCheck()
	if err != nil {
		errLogger.Printf("Error in fileCheck: %v", err)
		return err
	}

    file, err := ioutil.ReadFile(jl.jsonFile)
    if err != nil {
        jl.logger.Printf("Failed to read JSON file: %v", err)
        return err
    }

    var data map[string]interface{}
    err = json.Unmarshal(file, &data)
    if err != nil {
        jl.logger.Printf("Failed to unmarshal JSON: %v", err)
        return err
    }

    jl.data = data
    return nil
}

func NewJsonLoad() (*JsonLoad, error) {
	logger := log.New(os.Stdout, "json_loader/command_loader: ", log.LstdFlags)
	logger.Println("Entering NewJsonLoad function")
	defer logger.Println("Exiting NewJsonLoad function")
	loader, err := NewJsonLoader("json/command.json")
	if err != nil {
		return nil, err
	}

    return &JsonLoad{
        JsonLoader: loader,
        logger:     logger,
    }, nil
}

func (jl *JsonLoad) embedFormater(embed map[string]interface{}) map[string]interface{} {
	jl.logger.Println("Entering embedFormater function")
	defer jl.logger.Println("Exiting embedFormater function")
	if color, ok := embed["color"].(string); ok {
		var colorInt int64
		fmt.Sscanf(color, "%x", &colorInt)
		embed["color"] = colorInt
	}

    if timestamp, ok := embed["timestamp"].(float64); ok {
        timeValue := time.Unix(int64(timestamp), 0).UTC().Format(time.RFC3339)
        embed["timestamp"] = timeValue
    }

    return embed
}

func (jl *JsonLoad) getCommand(command, key string) (string, error) {
	jl.logger.Printf("Entering getCommand function with args: command=%s, key=%s", command, key)
	defer jl.logger.Println("Exiting getCommand function")
	cmd, ok := jl.data[command].(map[string]interface{})
	if !ok {
		jl.logger.Printf("Command '%s' not found.", command)
		return "", fmt.Errorf("command '%s' not found", command)
	}

    value, ok := cmd[key].(string)
    if !ok {
        jl.logger.Printf("Key '%s/%s' not found.", command, key)
        return "", fmt.Errorf("key '%s/%s' not found", command, key)
    }

    return value, nil
}

func (jl *JsonLoad) GetCommandEmbed(command string) (map[string]interface{}, error) {
	jl.logger.Printf("Entering GetCommandEmbed function with args: command=%s", command)
	defer jl.logger.Println("Exiting GetCommandEmbed function")
	embedPath, err := jl.getCommand(command, "embed")
	if err != nil {
		return nil, err
	}

    embedLoader, err := NewJsonLoader(embedPath)
    if err != nil {
        return nil, err
    }

    embedData, ok := embedLoader.data["embed"].(map[string]interface{})
    if !ok {
        jl.logger.Printf("Invalid embed format for command '%s'.", command)
        return nil, fmt.Errorf("invalid embed format for command '%s'", command)
    }

    formattedEmbed := jl.embedFormater(embedData)
    return formattedEmbed, nil
}

func NewModelLoad() (*ModelLoad, error) {
	logger := log.New(os.Stdout, "json_loader/model_loader: ", log.LstdFlags)
	logger.Println("Entering NewModelLoad function")
	defer logger.Println("Exiting NewModelLoad function")
	loader, err := NewJsonLoader("json/model.json")
	if err != nil {
		return nil, err
	}

    return &ModelLoad{
        JsonLoader: loader,
        logger:     logger,
    }, nil
}

func (ml *ModelLoad) get(key string) (string, error) {
	ml.logger.Printf("Entering get function with args: key=%s", key)
	defer ml.logger.Println("Exiting get function")
	value, ok := ml.data[key].(string)
	if !ok {
		ml.logger.Printf("Key '%s' not found.", key)
		return "", fmt.Errorf("key '%s' not found", key)
	}
    return value, nil
}

func (ml *ModelLoad) getPrompt(key string) (string, error) {
	ml.logger.Printf("Entering getPrompt function with args: key=%s", key)
	defer ml.logger.Println("Exiting getPrompt function")
	prompts, ok := ml.data["prompts"].(map[string]interface{})
	if !ok {
		ml.logger.Println("Prompts section not found.")
		return "", errors.New("prompts section not found")
	}

    value, ok := prompts[key].(string)
    if !ok {
        ml.logger.Printf("Prompt '%s' not found.", key)
        return "", fmt.Errorf("prompt '%s' not found", key)
    }
    return value, nil
}

func (ml *ModelLoad) GetName() string {
	ml.logger.Println("Entering GetName function")
	defer ml.logger.Println("Exiting GetName function")
	name, err := ml.get("name")
	if err != nil || name == "" {
		ml.logger.Println("Name not found. Using default value 'gemini-2.0-flash'.")
		return "gemini-2.0-flash"
	}
    return name
}

func (ml *ModelLoad) GetModelName() string {
	ml.logger.Println("Entering GetModelName function")
	defer ml.logger.Println("Exiting GetModelName function")
	modelName, err := ml.get("model_name")
	if err != nil || modelName == "" {
		ml.logger.Println("Model name not found. Using default value 'gemini-2.0-flash'.")
		return "gemini-2.0-flash"
	}
    return modelName
}

func (ml *ModelLoad) GetIcon() string {
	ml.logger.Println("Entering GetIcon function")
	defer ml.logger.Println("Exiting GetIcon function")
	icon, err := ml.get("icon")
	if err != nil {
		ml.logger.Println("Icon not found.")
		return ""
	}
    return icon
}

func (ml *ModelLoad) GetPromptDefault() string {
	ml.logger.Println("Entering GetPromptDefault function")
	defer ml.logger.Println("Exiting GetPromptDefault function")
	defaultPrompt, err := ml.getPrompt("default")
	if err != nil || defaultPrompt == "" {
		ml.logger.Println("Default prompt not found. Using default value.")
		return "You, as a chatbot, respond to the following statement."
	}
    return defaultPrompt
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	logger.Println("Entering onReady function")
	defer logger.Println("Exiting onReady function")
	logger.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	err := s.UpdateGameStatus(0, "/chat ではじめよう。")
	if err != nil {
		logger.Println("Error updating game status:", err)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger.Println("Entering messageCreate function")
	defer logger.Println("Exiting messageCreate function")
	if m.Author.ID == s.State.User.ID {
		return
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Println("Entering interactionCreate function")
	defer logger.Println("Exiting interactionCreate function")
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

    data := i.ApplicationCommandData()

    switch data.Name {
    case "chat":
        userID := i.Member.User.ID
        username := i.Member.User.Username
        message := data.Options[0].StringValue()
        timestamp := time.Now().Format(time.RFC3339)

        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "お返事を書いてるよ！ちょっと待ってね！",
            },
        })

        responseText, elapsed, err := chat.GetResponse(userID, username, message, timestamp)
        if err != nil {
            logger.Printf("Error getting response: %v", err)
            responseText = "なんかエラーが発生しました。"
        }

        content := fmt.Sprintf("%s\n%.2fms", responseText, elapsed)
        _, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
            Content: &content,
        })

        if err != nil {
            logger.Printf("Error sending message: %v", err)
        }

    case "reset":
        userID := i.Member.User.ID
        chat.ClearHistory(userID)

        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: ":white_check_mark: チャット履歴をリセットしました！",
            },
        })
    }
}
