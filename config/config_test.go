package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCustomPrompts(t *testing.T) {
	t.Run("load non-existent file returns empty config", func(t *testing.T) {
		cfg, err := loadCustomPrompts(filepath.Join(t.TempDir(), "nonexistent.json"))
		if err != nil {
			t.Fatalf("loadCustomPrompts should not error for non-existent file: %v", err)
		}
		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
		if cfg.Prompts == nil {
			t.Error("Expected non-nil prompts map")
		}
		if len(cfg.Prompts) != 0 {
			t.Errorf("Expected empty prompts map, got %d entries", len(cfg.Prompts))
		}
	})

	t.Run("load valid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "custom.json")
		content := `{"prompts": {"user1": "custom prompt for user1"}}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		cfg, err := loadCustomPrompts(path)
		if err != nil {
			t.Fatalf("loadCustomPrompts failed: %v", err)
		}
		if cfg.Prompts["user1"] != "custom prompt for user1" {
			t.Errorf("Expected prompt for user1, got %q", cfg.Prompts["user1"])
		}
	})

	t.Run("load invalid JSON returns empty config with no error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.json")
		if err := os.WriteFile(path, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		cfg, err := loadCustomPrompts(path)
		if err != nil {
			t.Fatalf("loadCustomPrompts should not error for invalid JSON (logs and continues): %v", err)
		}
		if cfg.Prompts == nil {
			t.Error("Expected non-nil prompts map")
		}
		if len(cfg.Prompts) != 0 {
			t.Errorf("Expected empty prompts for invalid JSON, got %d", len(cfg.Prompts))
		}
	})

	t.Run("load empty file returns empty config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.json")
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		cfg, err := loadCustomPrompts(path)
		if err != nil {
			t.Fatalf("loadCustomPrompts should not error for empty file: %v", err)
		}
		if len(cfg.Prompts) != 0 {
			t.Errorf("Expected empty prompts map, got %d entries", len(cfg.Prompts))
		}
	})

	t.Run("load file with non-existent directory returns empty config (file not found is ok)", func(t *testing.T) {
		cfg, err := loadCustomPrompts("/nonexistent/dir/file.json")
		if err != nil {
			t.Fatalf("loadCustomPrompts should not error for non-existent directory: %v", err)
		}
		if cfg == nil {
			t.Fatal("Expected non-nil config")
		}
	})
}
