package loader

import (
	"encoding/json"
	"errors"
	"os"
)

type ModelConfig struct {
	Name               string            `json:"name"`
	ModelName          string            `json:"model_name"`
	SecondaryModelName string            `json:"secondary_model_name,omitempty"`
	Icon               string            `json:"icon"`
	MaxHistorySize     int               `json:"max_history_size"`
	Prompts            map[string]string `json:"prompts"`
	About         About             `json:"about"`
	Ollama        OllamaConfig      `json:"ollama"`
	OpenAI        OpenAIConfig      `json:"openai"`
}

type OllamaConfig struct {
	Enabled     bool   `json:"enabled"`
	APIEndpoint string `json:"api_endpoint"`
	ModelName   string `json:"model_name"`
}

type OpenAIConfig struct {
	Enabled     bool   `json:"enabled"`
	APIEndpoint string `json:"api_endpoint"`
	ModelName   string `json:"model_name"`
	APIKey      string `json:"api_key,omitempty"`
}

type About struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

func (m *ModelConfig) GetPromptByUser(username string) string {
	if m.Prompts != nil {
		if prompt, exists := m.Prompts[username]; exists {
			return prompt
		}
		if defaultPrompt, exists := m.Prompts["default"]; exists && defaultPrompt != "" {
			return defaultPrompt
		}
	}
	return "You are a helpful assistant."
}

func LoadModelConfig(filepath string) (*ModelConfig, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var cfg ModelConfig
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	if cfg.Prompts["default"] == "" {
		return nil, errors.New("default prompt not defined")
	}

	return &cfg, nil
}
