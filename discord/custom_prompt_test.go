package discord

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// setupTestCustomPromptFile はテスト用のカスタムプロンプトファイルを作成・初期化するヘルパー
func setupTestCustomPromptFile(t *testing.T, initialData map[string]string) (string, func()) {
	t.Helper()
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_custom_model.json")

	originalPath := customPromptFilePath
	originalMutex := customPromptMutex
	customPromptFilePath = testFilePath
	customPromptMutex = sync.Mutex{}

	if initialData != nil {
		config := &CustomPromptConfig{Prompts: initialData}
		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal initial data: %v", err)
		}
		err = os.WriteFile(testFilePath, jsonData, 0644)
		if err != nil {
			t.Fatalf("Failed to write initial test file: %v", err)
		}
	} else {
	}

	cleanup := func() {
		customPromptFilePath = originalPath
		customPromptMutex = originalMutex
	}
	return testFilePath, cleanup
}

func readTestCustomPromptFile(t *testing.T, path string) *CustomPromptConfig {
	t.Helper()
	config := &CustomPromptConfig{Prompts: make(map[string]string)}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config
		}
		t.Fatalf("Failed to read test file %s: %v", path, err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, config); err != nil {
			t.Fatalf("Failed to unmarshal test file %s: %v", path, err)
		}
	}
	return config
}


func TestGetCustomPromptForUser(t *testing.T) {
	initialData := map[string]string{
		"user1": "Prompt for user1",
		"user2": "Another prompt",
	}
	_, cleanup := setupTestCustomPromptFile(t, initialData)
	defer cleanup()

	t.Run("Get existing user prompt", func(t *testing.T) {
		prompt, exists, err := GetCustomPromptForUser("user1")
		if err != nil {
			t.Fatalf("GetCustomPromptForUser failed: %v", err)
		}
		if !exists {
			t.Error("Expected prompt to exist for user1, but it doesn't")
		}
		if prompt != "Prompt for user1" {
			t.Errorf("Expected prompt 'Prompt for user1', got %q", prompt)
		}
	})

	t.Run("Get non-existing user prompt", func(t *testing.T) {
		prompt, exists, err := GetCustomPromptForUser("user3")
		if err != nil {
			t.Fatalf("GetCustomPromptForUser failed: %v", err)
		}
		if exists {
			t.Error("Expected prompt not to exist for user3, but it does")
		}
		if prompt != "" {
			t.Errorf("Expected empty prompt for non-existing user, got %q", prompt)
		}
	})

	t.Run("Get prompt when file does not exist", func(t *testing.T) {
		_, cleanup := setupTestCustomPromptFile(t, nil)
		defer cleanup()

		prompt, exists, err := GetCustomPromptForUser("anyuser")
		if err != nil {
			t.Fatalf("GetCustomPromptForUser failed for non-existent file: %v", err)
		}
		if exists {
			t.Error("Expected prompt not to exist when file doesn't exist")
		}
		if prompt != "" {
			t.Errorf("Expected empty prompt when file doesn't exist, got %q", prompt)
		}
	})
}

func TestSetCustomPromptForUser(t *testing.T) {
	initialData := map[string]string{
		"user1": "Original prompt",
	}
	testFilePath, cleanup := setupTestCustomPromptFile(t, initialData)
	defer cleanup()

	t.Run("Set new user prompt", func(t *testing.T) {
		newUser := "user_new"
		newPrompt := "This is a new prompt"
		err := SetCustomPromptForUser(newUser, newPrompt)
		if err != nil {
			t.Fatalf("SetCustomPromptForUser failed for new user: %v", err)
		}

		config := readTestCustomPromptFile(t, testFilePath)
		if prompt, exists := config.Prompts[newUser]; !exists || prompt != newPrompt {
			t.Errorf("Prompt for %s was not set correctly. Got: %q, Exists: %v", newUser, prompt, exists)
		}
		// 元のユーザーのプロンプトが残っているか確認
		if prompt, exists := config.Prompts["user1"]; !exists || prompt != "Original prompt" {
			t.Errorf("Original prompt for user1 was unexpectedly changed or deleted.")
		}
	})

	t.Run("Update existing user prompt", func(t *testing.T) {
		existingUser := "user1"
		updatedPrompt := "This is the updated prompt"
		err := SetCustomPromptForUser(existingUser, updatedPrompt)
		if err != nil {
			t.Fatalf("SetCustomPromptForUser failed for existing user: %v", err)
		}

		// ファイルを読み込んで確認
		config := readTestCustomPromptFile(t, testFilePath)
		if prompt, exists := config.Prompts[existingUser]; !exists || prompt != updatedPrompt {
			t.Errorf("Prompt for %s was not updated correctly. Got: %q, Exists: %v", existingUser, prompt, exists)
		}
	})

	t.Run("Set prompt when file does not exist initially", func(t *testing.T) {
		// ファイルが存在しない状態から開始
		testFilePathEmpty, cleanupEmpty := setupTestCustomPromptFile(t, nil)
		defer cleanupEmpty()

		user := "first_user"
		prompt := "First prompt ever"
		err := SetCustomPromptForUser(user, prompt)
		if err != nil {
			t.Fatalf("SetCustomPromptForUser failed for initially non-existent file: %v", err)
		}

		// ファイルが作成され、データが書き込まれているか確認
		config := readTestCustomPromptFile(t, testFilePathEmpty)
		if p, exists := config.Prompts[user]; !exists || p != prompt {
			t.Errorf("Prompt for %s was not set correctly in new file. Got: %q, Exists: %v", user, p, exists)
		}
	})
}

func TestDeleteCustomPromptForUser(t *testing.T) {
	initialData := map[string]string{
		"user_to_delete": "Delete me",
		"user_to_keep":   "Keep me",
	}
	testFilePath, cleanup := setupTestCustomPromptFile(t, initialData)
	defer cleanup()

	t.Run("Delete existing user prompt", func(t *testing.T) {
		userToDelete := "user_to_delete"
		err := DeleteCustomPromptForUser(userToDelete)
		if err != nil {
			t.Fatalf("DeleteCustomPromptForUser failed for existing user: %v", err)
		}

		// ファイルを読み込んで確認
		config := readTestCustomPromptFile(t, testFilePath)
		if _, exists := config.Prompts[userToDelete]; exists {
			t.Errorf("Prompt for %s should have been deleted, but still exists.", userToDelete)
		}
		// 他のユーザーのプロンプトが残っているか確認
		if _, exists := config.Prompts["user_to_keep"]; !exists {
			t.Errorf("Prompt for user_to_keep was unexpectedly deleted.")
		}
	})

	t.Run("Delete non-existing user prompt", func(t *testing.T) {
		nonExistingUser := "user_does_not_exist"
		configBefore := readTestCustomPromptFile(t, testFilePath)
		numPromptsBefore := len(configBefore.Prompts)

		err := DeleteCustomPromptForUser(nonExistingUser)
		if err != nil {
			t.Fatalf("DeleteCustomPromptForUser failed for non-existing user: %v", err)
		}

		configAfter := readTestCustomPromptFile(t, testFilePath)
		if _, exists := configAfter.Prompts[nonExistingUser]; exists {
			t.Errorf("Prompt for non-existing user %s was unexpectedly created.", nonExistingUser)
		}
		if len(configAfter.Prompts) != numPromptsBefore {
			t.Errorf("Number of prompts changed after deleting non-existing user. Before: %d, After: %d", numPromptsBefore, len(configAfter.Prompts))
		}
		if _, exists := configAfter.Prompts["user_to_keep"]; !exists {
			t.Errorf("Prompt for user_to_keep was unexpectedly deleted while deleting non-existing user.")
		}
	})

	t.Run("Delete from non-existent file", func(t *testing.T) {
		_, cleanupEmpty := setupTestCustomPromptFile(t, nil)
		defer cleanupEmpty()

		err := DeleteCustomPromptForUser("anyuser")
		if err != nil {
			t.Fatalf("DeleteCustomPromptForUser failed for non-existent file: %v", err)
		}
	})
}
