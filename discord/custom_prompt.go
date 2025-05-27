package discord

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/eraiza0816/llm-discord/config"
)

const customPromptFilePath = "json/custom_model.json"

var saveMutex sync.Mutex // 保存処理の排他制御用

// loadCustomPromptsFromFile はファイルを直接読み込むヘルパー関数です。
// config.LoadConfig 経由での読み込みが推奨されます。
func loadCustomPromptsFromFile() (*config.CustomPromptConfig, error) {
	saveMutex.Lock() // 読み込み時もロックを取得する（保存処理との競合を避けるため）
	defer saveMutex.Unlock()

	cfg := &config.CustomPromptConfig{Prompts: make(map[string]string)}

	if _, err := os.Stat(customPromptFilePath); err == nil {
		data, err := os.ReadFile(customPromptFilePath)
		if err != nil {
			return nil, fmt.Errorf("カスタムプロンプトファイル '%s' の読み込みに失敗しました: %w", customPromptFilePath, err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, cfg); err != nil {
				// log.Printf は main など上位のロガーに任せるか検討
				fmt.Printf("カスタムプロンプトファイル '%s' のJSON解析に失敗しました: %v。空の設定として扱います。\n", customPromptFilePath, err)
				cfg.Prompts = make(map[string]string)
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("カスタムプロンプトファイル '%s' の状態確認中にエラーが発生しました: %w", customPromptFilePath, err)
	}

	return cfg, nil
}

func saveCustomPrompts(cfg *config.CustomPromptConfig) error {
	saveMutex.Lock()
	defer saveMutex.Unlock()

	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("カスタムプロンプト設定のJSONエンコードに失敗しました: %w", err)
	}

	err = os.WriteFile(customPromptFilePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("カスタムプロンプトファイル '%s' への書き込みに失敗しました: %w", customPromptFilePath, err)
	}

	return nil
}

// GetCustomPromptForUser は、config.Config からカスタムプロンプトを取得します。
// この関数は、config.LoadConfig() で読み込まれた設定情報を利用することを想定しています。
// 直接ファイルを読み込む場合は loadCustomPromptsFromFile を使用しますが、通常は不要です。
func GetCustomPromptForUser(cfg *config.Config, username string) (string, bool) {
	if cfg == nil || cfg.CustomModel == nil {
		return "", false
	}
	prompt, exists := cfg.CustomModel.Prompts[username]
	return prompt, exists
}

func SetCustomPromptForUser(username string, prompt string) error {
	cfg, err := loadCustomPromptsFromFile() // 保存時は最新のファイル内容を読み込む
	if err != nil {
		return err
	}

	if cfg.Prompts == nil {
		cfg.Prompts = make(map[string]string)
	}
	cfg.Prompts[username] = prompt

	return saveCustomPrompts(cfg)
}

func DeleteCustomPromptForUser(username string) error {
	cfg, err := loadCustomPromptsFromFile() // 保存時は最新のファイル内容を読み込む
	if err != nil {
		return err
	}

	if _, exists := cfg.Prompts[username]; exists {
		delete(cfg.Prompts, username)
		return saveCustomPrompts(cfg)
	}

	return nil
}
