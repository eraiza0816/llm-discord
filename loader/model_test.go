package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestConfigFile(t *testing.T, dir string, filename string, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file %s: %v", path, err)
	}
	return path
}

func TestLoadModelConfig(t *testing.T) {
	// テスト用の一次ディレクトリを作成
	tempDir := t.TempDir()

	// --- Test Case 1: Valid Config ---
	validJSON := `{
		"name": "Test Bot",
		"model_name": "gemini-test",
		"icon": "icon_url",
		"max_history_size": 10,
		"prompts": {
			"default": "You are a test assistant.",
			"user1": "You are user1's assistant."
		},
		"about": {
			"title": "About Test Bot",
			"description": "This is a test bot.",
			"url": "http://example.com"
		},
		"ollama": {
			"enabled": false,
			"api_endpoint": "",
			"model_name": ""
		}
	}`
	validPath := createTestConfigFile(t, tempDir, "valid.json", validJSON)

	t.Run("Load valid config", func(t *testing.T) {
		cfg, err := LoadModelConfig(validPath)
		if err != nil {
			t.Fatalf("LoadModelConfig failed for valid config: %v", err)
		}
		if cfg == nil {
			t.Fatal("LoadModelConfig returned nil for valid config")
		}
		if cfg.Name != "Test Bot" {
			t.Errorf("Expected Name 'Test Bot', got %q", cfg.Name)
		}
		if cfg.Prompts["default"] != "You are a test assistant." {
			t.Errorf("Expected default prompt 'You are a test assistant.', got %q", cfg.Prompts["default"])
		}
		if cfg.MaxHistorySize != 10 {
			t.Errorf("Expected MaxHistorySize 10, got %d", cfg.MaxHistorySize)
		}
		if cfg.Ollama.Enabled != false {
			t.Errorf("Expected Ollama.Enabled false, got %v", cfg.Ollama.Enabled)
		}
	})

	// --- Test Case 2: File Not Found ---
	t.Run("Load non-existent file", func(t *testing.T) {
		_, err := LoadModelConfig(filepath.Join(tempDir, "not_found.json"))
		if err == nil {
			t.Error("LoadModelConfig should return error for non-existent file, but got nil")
		}
		// エラーの種類もチェックできるとより良いけど、今回は nil でないことだけ確認
	})

	// --- Test Case 3: Invalid JSON ---
	invalidJSON := `{ "name": "Test Bot", "prompts": { "default": "abc }` // JSONが壊れてる
	invalidPath := createTestConfigFile(t, tempDir, "invalid.json", invalidJSON)
	t.Run("Load invalid JSON", func(t *testing.T) {
		_, err := LoadModelConfig(invalidPath)
		if err == nil {
			t.Error("LoadModelConfig should return error for invalid JSON, but got nil")
		}
	})

	// --- Test Case 4: Missing Default Prompt ---
	missingDefaultJSON := `{
		"name": "Test Bot",
		"prompts": {
			"user1": "You are user1's assistant."
		}
	}`
	missingDefaultPath := createTestConfigFile(t, tempDir, "missing_default.json", missingDefaultJSON)
	t.Run("Load config missing default prompt", func(t *testing.T) {
		_, err := LoadModelConfig(missingDefaultPath)
		if err == nil {
			t.Error("LoadModelConfig should return error for missing default prompt, but got nil")
		}
		// エラーメッセージの内容もチェックするとより良い
		expectedErrMsg := "default prompt not defined"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
		}
	})
}

func TestModelConfig_GetPromptByUser(t *testing.T) {
	cfg := &ModelConfig{
		Prompts: map[string]string{
			"default": "Default Prompt",
			"userA":   "Prompt for User A",
			"userB":   "Prompt for User B",
		},
	}

	tests := []struct {
		name         string
		username     string
		expectedPrompt string
	}{
		{"Get prompt for existing user A", "userA", "Prompt for User A"},
		{"Get prompt for existing user B", "userB", "Prompt for User B"},
		{"Get prompt for non-existing user", "userC", "Default Prompt"},
		{"Get default prompt directly", "default", "Default Prompt"}, // "default" という名前のユーザーがいても、default が返る
		{"Get prompt with empty username", "", "Default Prompt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := cfg.GetPromptByUser(tt.username)
			if prompt != tt.expectedPrompt {
				t.Errorf("GetPromptByUser(%q) = %q; want %q", tt.username, prompt, tt.expectedPrompt)
			}
		})
	}

	// --- Test Case: Prompts map is nil ---
	t.Run("Get prompt when Prompts map is nil", func(t *testing.T) {
		cfgNilPrompts := &ModelConfig{Prompts: nil}
		expectedDefault := "You are a helpful assistant." // ハードコードされたデフォルト値
		prompt := cfgNilPrompts.GetPromptByUser("anyuser")
		if prompt != expectedDefault {
			t.Errorf("GetPromptByUser with nil Prompts map = %q; want %q", prompt, expectedDefault)
		}
	})

	// --- Test Case: Prompts map exists but is empty ---
	t.Run("Get prompt when Prompts map is empty", func(t *testing.T) {
		cfgEmptyPrompts := &ModelConfig{Prompts: map[string]string{}}
		expectedDefault := "You are a helpful assistant." // ハードコードされたデフォルト値
		prompt := cfgEmptyPrompts.GetPromptByUser("anyuser")
		if prompt != expectedDefault {
			t.Errorf("GetPromptByUser with empty Prompts map = %q; want %q", prompt, expectedDefault)
		}
	})
}
