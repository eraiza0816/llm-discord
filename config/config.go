package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"encoding/json"
	"github.com/eraiza0816/llm-discord/loader"
	"github.com/joho/godotenv"
	"log"
	"sync"
)

// CustomPromptConfig はユーザーごとのカスタムプロンプト設定を表します。
type CustomPromptConfig struct {
	Prompts map[string]string `json:"prompts"`
}

var customPromptMutex sync.Mutex

type Config struct {
	DiscordBotToken string
	GeminiAPIKey    string
	Model           *loader.ModelConfig
	CustomModel     *CustomPromptConfig
}

func loadCustomPrompts(filePath string) (*CustomPromptConfig, error) {
	customPromptMutex.Lock()
	defer customPromptMutex.Unlock()

	config := &CustomPromptConfig{Prompts: make(map[string]string)}

	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("カスタムプロンプトファイル '%s' の読み込みに失敗しました: %w", filePath, err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, config); err != nil {
				log.Printf("カスタムプロンプトファイル '%s' のJSON解析に失敗しました: %v。空の設定として扱います。", filePath, err)
				config.Prompts = make(map[string]string)
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("カスタムプロンプトファイル '%s' の状態確認中にエラーが発生しました: %w", filePath, err)
	}

	return config, nil
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return nil, fmt.Errorf(".env ファイルの読み込みに失敗しました: %w", err)
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	if token == "" || geminiAPIKey == "" {
		missingVars := []string{}
		if token == "" {
			missingVars = append(missingVars, "DISCORD_BOT_TOKEN")
		}
		if geminiAPIKey == "" {
			missingVars = append(missingVars, "GEMINI_API_KEY")
		}
		return nil, errors.New("以下の環境変数が設定されていません: " + strings.Join(missingVars, ", "))
	}

	modelCfg, err := loader.LoadModelConfig("json/model.json")
	if err != nil {
		return nil, fmt.Errorf("model.json の読み込みに失敗しました: %w", err)
	}

	customModelCfg, err := loadCustomPrompts("json/custom_model.json")
	if err != nil {
		return nil, fmt.Errorf("custom_model.json の読み込みに失敗しました: %w", err)
	}

	return &Config{
		DiscordBotToken: token,
		GeminiAPIKey:    geminiAPIKey,
		Model:           modelCfg,
		CustomModel:     customModelCfg,
	}, nil
}
