package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/eraiza0816/llm-discord/loader"
	"github.com/joho/godotenv"
)

type Config struct {
	DiscordBotToken string
	GeminiAPIKey    string
	Model           *loader.ModelConfig
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

	return &Config{
		DiscordBotToken: token,
		GeminiAPIKey:    geminiAPIKey,
		Model:           modelCfg,
	}, nil
}
