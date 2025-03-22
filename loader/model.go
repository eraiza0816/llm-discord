package loader

import (
	"encoding/json"
	"errors"
	"os"
)

type ModelConfig struct {
	Name      string            `json:"name"`
	ModelName     string            `json:"model_name"`
	Icon          string            `json:"icon"`
	MaxHistorySize int               `json:"max_history_size"`
	Prompts       map[string]string `json:"prompts"`
	About         About             `json:"about"`
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
		return m.Prompts["default"]
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
