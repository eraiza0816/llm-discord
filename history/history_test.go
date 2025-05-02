package history

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewInMemoryHistoryManager(t *testing.T) {
	manager := NewInMemoryHistoryManager(10)
	if manager == nil {
		t.Fatal("NewInMemoryHistoryManager returned nil")
	}
	if _, ok := manager.(*InMemoryHistoryManager); !ok {
		t.Fatal("NewInMemoryHistoryManager did not return *InMemoryHistoryManager")
	}
}

func TestInMemoryHistoryManager_AddGet(t *testing.T) {
	manager := NewInMemoryHistoryManager(4)

	userID1 := "user1"
	userID2 := "user2"

	t.Run("Add and Get for User 1", func(t *testing.T) {
		if got := manager.Get(userID1); got != "" {
			t.Errorf("Expected empty history for %s, but got %q", userID1, got)
		}

		manager.Add(userID1, "hello", "hi there")
		expected1 := "hello\nhi there"
		if got := manager.Get(userID1); got != expected1 {
			t.Errorf("Expected %q for %s, but got %q", expected1, userID1, got)
		}

		manager.Add(userID1, "how are you?", "I'm good!")
		expected2 := "hello\nhi there\nhow are you?\nI'm good!"
		if got := manager.Get(userID1); got != expected2 {
			t.Errorf("Expected %q for %s, but got %q", expected2, userID1, got)
		}
	})

	t.Run("Add and Get for User 2 (independent)", func(t *testing.T) {
		if got := manager.Get(userID2); got != "" {
			t.Errorf("Expected empty history for %s, but got %q", userID2, got)
		}

		manager.Add(userID2, "test", "response")
		expected3 := "test\nresponse"
		if got := manager.Get(userID2); got != expected3 {
			t.Errorf("Expected %q for %s, but got %q", expected3, userID2, got)
		}

		expectedUser1 := "hello\nhi there\nhow are you?\nI'm good!"
		if got := manager.Get(userID1); got != expectedUser1 {
			t.Errorf("History for %s should not change, expected %q, but got %q", userID1, expectedUser1, got)
		}
	})
}

func TestInMemoryHistoryManager_Add_MaxSize(t *testing.T) {
	maxSize := 6
	manager := NewInMemoryHistoryManager(maxSize)
	userID := "user_max"

	for i := 0; i < maxSize/2; i++ {
		msg := fmt.Sprintf("msg%d", i)
		res := fmt.Sprintf("res%d", i)
		manager.Add(userID, msg, res)
	}

	expectedFull := []string{}
	for i := 0; i < maxSize/2; i++ {
		expectedFull = append(expectedFull, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	expectedFullStr := strings.Join(expectedFull, "\n")
	if got := manager.Get(userID); got != expectedFullStr {
		t.Errorf("Expected full history %q, but got %q", expectedFullStr, got)
	}

	manager.Add(userID, "new_msg", "new_res")

	expectedAfterOverflow := []string{}
	for i := 1; i < maxSize/2; i++ {
		expectedAfterOverflow = append(expectedAfterOverflow, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	expectedAfterOverflow = append(expectedAfterOverflow, "new_msg", "new_res")
	expectedAfterOverflowStr := strings.Join(expectedAfterOverflow, "\n")

	if got := manager.Get(userID); got != expectedAfterOverflowStr {
		t.Errorf("Expected history after overflow %q, but got %q", expectedAfterOverflowStr, got)
	}

	lines := strings.Split(manager.Get(userID), "\n")
	if len(lines) != maxSize {
		if !(len(lines) == 1 && lines[0] == "") {
			t.Errorf("Expected history length to be %d lines, but got %d lines", maxSize, len(lines))
		} else if maxSize != 0 {
             t.Errorf("Expected history length to be %d lines, but got 0 lines", maxSize)
        }
	}
}


func TestInMemoryHistoryManager_Clear(t *testing.T) {
	manager := NewInMemoryHistoryManager(10)
	userID1 := "user_to_clear"
	userID2 := "user_to_keep"

	manager.Add(userID1, "msg1", "res1")
	manager.Add(userID2, "msg2", "res2")

	if manager.Get(userID1) == "" {
		t.Errorf("History for %s should exist before clear", userID1)
	}
	if manager.Get(userID2) == "" {
		t.Errorf("History for %s should exist before clear", userID2)
	}

	manager.Clear(userID1)

	if got := manager.Get(userID1); got != "" {
		t.Errorf("Expected empty history for %s after clear, but got %q", userID1, got)
	}
	expectedUser2 := "msg2\nres2"
	if got := manager.Get(userID2); got != expectedUser2 {
		t.Errorf("History for %s should remain after clearing %s, expected %q, but got %q", userID2, userID1, expectedUser2, got)
	}

	manager.Clear("non_existent_user")
}
