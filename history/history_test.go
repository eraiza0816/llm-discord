package history

import (
	"testing"
)

func TestInMemoryHistoryManager(t *testing.T) {
	t.Run("NewInMemoryHistoryManager creates empty manager", func(t *testing.T) {
		mgr, err := NewInMemoryHistoryManager(10)
		if err != nil {
			t.Fatalf("NewInMemoryHistoryManager failed: %v", err)
		}
		if mgr == nil {
			t.Fatal("NewInMemoryHistoryManager returned nil")
		}
	})

	t.Run("Add and Get history", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		err := mgr.Add("user1", "thread1", "hello", "world")
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		msgs, err := mgr.Get("user1", "thread1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(msgs))
		}
		if msgs[0].Role != "user" || msgs[0].Content != "hello" {
			t.Errorf("Expected user/hello, got %s/%s", msgs[0].Role, msgs[0].Content)
		}
		if msgs[1].Role != "model" || msgs[1].Content != "world" {
			t.Errorf("Expected model/world, got %s/%s", msgs[1].Role, msgs[1].Content)
		}
	})

	t.Run("Get returns empty slice for unknown user", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		msgs, err := mgr.Get("unknown", "thread1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if msgs == nil {
			t.Error("Expected non-nil empty slice, got nil")
		}
		if len(msgs) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(msgs))
		}
	})

	t.Run("Clear removes history for user", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		mgr.Add("user1", "thread1", "hello", "world")
		err := mgr.Clear("user1", "thread1")
		if err != nil {
			t.Fatalf("Clear failed: %v", err)
		}
		msgs, _ := mgr.Get("user1", "thread1")
		if len(msgs) != 0 {
			t.Errorf("Expected 0 messages after clear, got %d", len(msgs))
		}
	})

	t.Run("History size limit is enforced", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(2)
		for i := 0; i < 5; i++ {
			mgr.Add("user1", "thread1", "msg", "resp")
		}
		msgs, _ := mgr.Get("user1", "thread1")
		if len(msgs) > 4 {
			t.Errorf("Expected at most 4 messages (2 pairs), got %d", len(msgs))
		}
	})

	t.Run("Different users have separate histories", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		mgr.Add("user1", "thread1", "msg1", "resp1")
		mgr.Add("user2", "thread1", "msg2", "resp2")

		msgs1, _ := mgr.Get("user1", "thread1")
		msgs2, _ := mgr.Get("user2", "thread1")

		if len(msgs1) != 2 || len(msgs2) != 2 {
			t.Errorf("Expected 2 messages each, got %d and %d", len(msgs1), len(msgs2))
		}
		if msgs1[0].Content != "msg1" {
			t.Errorf("user1 should have msg1, got %s", msgs1[0].Content)
		}
		if msgs2[0].Content != "msg2" {
			t.Errorf("user2 should have msg2, got %s", msgs2[0].Content)
		}
	})

	t.Run("ClearAllByThreadID does nothing for in-memory", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		mgr.Add("user1", "thread1", "msg", "resp")
		err := mgr.ClearAllByThreadID("thread1")
		if err != nil {
			t.Fatalf("ClearAllByThreadID failed: %v", err)
		}
		msgs, _ := mgr.Get("user1", "thread1")
		if len(msgs) != 2 {
			t.Errorf("InMemoryHistoryManager ClearAllByThreadID should be no-op, got %d messages", len(msgs))
		}
	})

	t.Run("GetBotConversationCount returns 0 for in-memory", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		mgr.Add("user1", "thread1", "msg", "resp")
		count, err := mgr.GetBotConversationCount("thread1", "user1")
		if err != nil {
			t.Fatalf("GetBotConversationCount failed: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0, got %d", count)
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		mgr, _ := NewInMemoryHistoryManager(10)
		err := mgr.Close()
		if err != nil {
			t.Errorf("Close should return nil, got %v", err)
		}
	})
}

func TestHistoryMessage(t *testing.T) {
	t.Run("message struct fields", func(t *testing.T) {
		msg := HistoryMessage{Role: "user", Content: "test content"}
		if msg.Role != "user" {
			t.Errorf("Expected role 'user', got %q", msg.Role)
		}
		if msg.Content != "test content" {
			t.Errorf("Expected content 'test content', got %q", msg.Content)
		}
	})
}
