package loader

import (
	"encoding/json"
	"errors"
	"os"
)

type ModelConfig struct {
	Name      string            `json:"name"`
	ModelName string            `json:"model_name"`
	Icon      string            `json:"icon"`
	Prompts   map[string]string `json:"prompts"`
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
