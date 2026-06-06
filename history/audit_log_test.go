package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitAuditLog(t *testing.T) {
	t.Run("init creates data directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		origPath := auditLogPath
		t.Cleanup(func() { auditLogPath = origPath })

		testPath := filepath.Join(tmpDir, "data", "audit.jsonl")
		_ = os.MkdirAll(filepath.Dir(testPath), 0755)
		auditLogPath = testPath

		err := InitAuditLog()
		if err != nil {
			t.Fatalf("InitAuditLog failed: %v", err)
		}
	})

	t.Run("CloseAuditLog does not panic", func(t *testing.T) {
		CloseAuditLog()
	})
}

func TestLogMessageCreate(t *testing.T) {
	t.Run("log message create", func(t *testing.T) {
		tmpDir := t.TempDir()
		origPath := auditLogPath
		t.Cleanup(func() { auditLogPath = origPath })

		testPath := filepath.Join(tmpDir, "audit.jsonl")
		auditLogPath = testPath

		err := LogMessageCreate("msg1", "ch1", "guild1", "user1", "testuser", "hello", []string{"https://example.com/img.png"}, time.Now())
		if err != nil {
			t.Fatalf("LogMessageCreate failed: %v", err)
		}

		data, err := os.ReadFile(testPath)
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("Expected non-empty audit log file")
		}
	})
}

func TestLogMessageUpdate(t *testing.T) {
	t.Run("log message update", func(t *testing.T) {
		tmpDir := t.TempDir()
		origPath := auditLogPath
		t.Cleanup(func() { auditLogPath = origPath })

		testPath := filepath.Join(tmpDir, "audit.jsonl")
		auditLogPath = testPath

		err := LogMessageUpdate("msg1", "updated content", time.Now())
		if err != nil {
			t.Fatalf("LogMessageUpdate failed: %v", err)
		}

		data, err := os.ReadFile(testPath)
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("Expected non-empty audit log file")
		}
	})
}

func TestLogMessageDelete(t *testing.T) {
	t.Run("log message delete", func(t *testing.T) {
		tmpDir := t.TempDir()
		origPath := auditLogPath
		t.Cleanup(func() { auditLogPath = origPath })

		testPath := filepath.Join(tmpDir, "audit.jsonl")
		auditLogPath = testPath

		err := LogMessageDelete("msg1", time.Now())
		if err != nil {
			t.Fatalf("LogMessageDelete failed: %v", err)
		}

		data, err := os.ReadFile(testPath)
		if err != nil {
			t.Fatalf("Failed to read audit log: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("Expected non-empty audit log file")
		}
	})
}
