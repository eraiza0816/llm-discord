package discord

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

const customPromptFilePath = "json/custom_model.json"

var customPromptMutex sync.Mutex

func loadCustomPrompts() (*CustomPromptConfig, error) {
	customPromptMutex.Lock()
	defer customPromptMutex.Unlock()

	config := &CustomPromptConfig{Prompts: make(map[string]string)}

	if _, err := os.Stat(customPromptFilePath); err == nil {
		data, err := os.ReadFile(customPromptFilePath)
		if err != nil {
			return nil, fmt.Errorf("カスタムプロンプトファイル '%s' の読み込みに失敗しました: %w", customPromptFilePath, err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, config); err != nil {
				log.Printf("カスタムプロンプトファイル '%s' のJSON解析に失敗しました: %v。空の設定として扱います。", customPromptFilePath, err)
				config.Prompts = make(map[string]string)
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("カスタムプロンプトファイル '%s' の状態確認中にエラーが発生しました: %w", customPromptFilePath, err)
	}

	return config, nil
}

func saveCustomPrompts(config *CustomPromptConfig) error {
	customPromptMutex.Lock()
	defer customPromptMutex.Unlock()

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("カスタムプロンプト設定のJSONエンコードに失敗しました: %w", err)
	}

	err = os.WriteFile(customPromptFilePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("カスタムプロンプトファイル '%s' への書き込みに失敗しました: %w", customPromptFilePath, err)
	}

	return nil
}

func GetCustomPromptForUser(username string) (string, bool, error) {
	config, err := loadCustomPrompts()
	if err != nil {
		return "", false, err
	}

	prompt, exists := config.Prompts[username]
	return prompt, exists, nil
}

func SetCustomPromptForUser(username string, prompt string) error {
	config, err := loadCustomPrompts()
	if err != nil {
		return err
	}

	config.Prompts[username] = prompt

	return saveCustomPrompts(config)
}

func DeleteCustomPromptForUser(username string) error {
	config, err := loadCustomPrompts()
	if err != nil {
		return err
	}

	if _, exists := config.Prompts[username]; exists {
		delete(config.Prompts, username)
		return saveCustomPrompts(config)
	}

	return nil
}
