package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordBotToken string
	GeminiAPIKey    string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		return nil, err
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	if token == "" || geminiAPIKey == "" {
		return nil, errors.New("環境変数が設定されていません")
	}

	return &Config{
		DiscordBotToken: token,
		GeminiAPIKey:    geminiAPIKey,
	}, nil
}
