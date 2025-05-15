package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testThreadID = "test-thread"
const testThreadID2 = "test-thread-2"

func TestNewInMemoryHistoryManager(t *testing.T) {
	manager := NewInMemoryHistoryManager(10)
	assert.NotNil(t, manager)
	_, ok := manager.(*InMemoryHistoryManager)
	assert.True(t, ok, "NewInMemoryHistoryManager did not return *InMemoryHistoryManager")
}

func TestInMemoryHistoryManager_AddGet(t *testing.T) {
	manager := NewInMemoryHistoryManager(4)

	userID1 := "user1"
	userID2 := "user2"

	t.Run("Add and Get for User 1", func(t *testing.T) {
		assert.Empty(t, manager.Get(userID1, testThreadID))

		manager.Add(userID1, testThreadID, "hello", "hi there")
		expected1 := "hello\nhi there"
		assert.Equal(t, expected1, manager.Get(userID1, testThreadID))

		manager.Add(userID1, testThreadID, "how are you?", "I'm good!")
		expected2 := "hello\nhi there\nhow are you?\nI'm good!"
		assert.Equal(t, expected2, manager.Get(userID1, testThreadID))
	})

	t.Run("Add and Get for User 2 (independent)", func(t *testing.T) {
		assert.Empty(t, manager.Get(userID2, testThreadID))

		manager.Add(userID2, testThreadID, "test", "response")
		expected3 := "test\nresponse"
		assert.Equal(t, expected3, manager.Get(userID2, testThreadID))

		// User1's history should remain unchanged
		expectedUser1 := "hello\nhi there\nhow are you?\nI'm good!"
		assert.Equal(t, expectedUser1, manager.Get(userID1, testThreadID))
	})
}

func TestInMemoryHistoryManager_Add_MaxSize(t *testing.T) {
	maxSize := 6 // Max 6 items (3 pairs of message/response)
	manager := NewInMemoryHistoryManager(maxSize)
	userID := "user_max"

	for i := 0; i < maxSize/2; i++ {
		msg := fmt.Sprintf("msg%d", i)
		res := fmt.Sprintf("res%d", i)
		manager.Add(userID, testThreadID, msg, res)
	}

	var expectedFull []string
	for i := 0; i < maxSize/2; i++ {
		expectedFull = append(expectedFull, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	assert.Equal(t, strings.Join(expectedFull, "\n"), manager.Get(userID, testThreadID))

	manager.Add(userID, testThreadID, "new_msg", "new_res")

	var expectedAfterOverflow []string
	// The first pair (msg0, res0) should be removed
	for i := 1; i < maxSize/2; i++ {
		expectedAfterOverflow = append(expectedAfterOverflow, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	expectedAfterOverflow = append(expectedAfterOverflow, "new_msg", "new_res")
	assert.Equal(t, strings.Join(expectedAfterOverflow, "\n"), manager.Get(userID, testThreadID))

	lines := strings.Split(manager.Get(userID, testThreadID), "\n")
	if maxSize == 0 {
		assert.Condition(t, func() bool { return len(lines) == 1 && lines[0] == "" || len(lines) == 0 }, "History should be empty or a single empty string for maxSize 0")
	} else {
		assert.Len(t, lines, maxSize, "History length should be maxSize")
	}
}

func TestInMemoryHistoryManager_Clear(t *testing.T) {
	manager := NewInMemoryHistoryManager(10)
	userID1 := "user_to_clear"
	userID2 := "user_to_keep"

	manager.Add(userID1, testThreadID, "msg1", "res1")
	manager.Add(userID2, testThreadID, "msg2", "res2")

	require.NotEmpty(t, manager.Get(userID1, testThreadID))
	require.NotEmpty(t, manager.Get(userID2, testThreadID))

	manager.Clear(userID1, testThreadID)

	assert.Empty(t, manager.Get(userID1, testThreadID))
	expectedUser2 := "msg2\nres2"
	assert.Equal(t, expectedUser2, manager.Get(userID2, testThreadID))

	manager.Clear("non_existent_user", testThreadID) // Should not panic
}

func TestInMemoryHistoryManager_ClearAllByThreadID(t *testing.T) {
	manager := NewInMemoryHistoryManager(10)
	userID1 := "user1"
	userID2 := "user2"

	manager.Add(userID1, testThreadID, "msg1_thread1", "res1_thread1")
	manager.Add(userID2, testThreadID, "msg2_thread1", "res2_thread1")
	manager.Add(userID1, testThreadID2, "msg1_thread2", "res1_thread2")

	// InMemoryHistoryManager's ClearAllByThreadID is a no-op as per current implementation
	// This test primarily ensures it doesn't panic.
	// If its behavior changes, this test needs to be updated.
	manager.ClearAllByThreadID(testThreadID)

	// Histories should remain as InMemoryHistoryManager doesn't implement thread-specific clearing for ClearAllByThreadID
	assert.NotEmpty(t, manager.Get(userID1, testThreadID))
	assert.NotEmpty(t, manager.Get(userID2, testThreadID))
	assert.NotEmpty(t, manager.Get(userID1, testThreadID2))
}

// --- SQLiteHistoryManager Tests ---

func setupSQLiteManager(t *testing.T, maxHistorySize int) (HistoryManager, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "history_test_sqlite")
	require.NoError(t, err)

	manager, err := NewSQLiteHistoryManager(tempDir, maxHistorySize)
	require.NoError(t, err)
	require.NotNil(t, manager)

	sqliteManager, ok := manager.(*SQLiteHistoryManager)
	require.True(t, ok)

	cleanup := func() {
		err := sqliteManager.Close()
		assert.NoError(t, err, "Failed to close SQLite manager")
		err = os.RemoveAll(tempDir)
		assert.NoError(t, err, "Failed to remove temp dir")
	}

	return manager, cleanup
}

func TestNewSQLiteHistoryManager(t *testing.T) {
	manager, cleanup := setupSQLiteManager(t, 10)
	defer cleanup()
	assert.NotNil(t, manager)
}

func TestSQLiteHistoryManager_AddGet(t *testing.T) {
	manager, cleanup := setupSQLiteManager(t, 4) // Max 4 pairs
	defer cleanup()

	userID1 := "user1-sqlite"
	userID2 := "user2-sqlite"

	t.Run("Add and Get for User 1, Thread 1", func(t *testing.T) {
		assert.Empty(t, manager.Get(userID1, testThreadID))

		manager.Add(userID1, testThreadID, "hello sqlite", "hi there sqlite")
		expected1 := "hello sqlite\nhi there sqlite"
		assert.Equal(t, expected1, manager.Get(userID1, testThreadID))

		manager.Add(userID1, testThreadID, "how are you sqlite?", "I'm good sqlite!")
		expected2 := "hello sqlite\nhi there sqlite\nhow are you sqlite?\nI'm good sqlite!"
		assert.Equal(t, expected2, manager.Get(userID1, testThreadID))
	})

	t.Run("Add and Get for User 1, Thread 2 (independent thread)", func(t *testing.T) {
		assert.Empty(t, manager.Get(userID1, testThreadID2))
		manager.Add(userID1, testThreadID2, "message for thread 2", "response for thread 2")
		expectedT2 := "message for thread 2\nresponse for thread 2"
		assert.Equal(t, expectedT2, manager.Get(userID1, testThreadID2))

		// History for thread1 should remain unchanged
		expectedUser1Thread1 := "hello sqlite\nhi there sqlite\nhow are you sqlite?\nI'm good sqlite!"
		assert.Equal(t, expectedUser1Thread1, manager.Get(userID1, testThreadID))
	})

	t.Run("Add and Get for User 2, Thread 1 (independent user)", func(t *testing.T) {
		assert.Empty(t, manager.Get(userID2, testThreadID))
		manager.Add(userID2, testThreadID, "user2 message", "user2 response")
		expectedU2T1 := "user2 message\nuser2 response"
		assert.Equal(t, expectedU2T1, manager.Get(userID2, testThreadID))

		// History for user1, thread1 should remain unchanged
		expectedUser1Thread1 := "hello sqlite\nhi there sqlite\nhow are you sqlite?\nI'm good sqlite!"
		assert.Equal(t, expectedUser1Thread1, manager.Get(userID1, testThreadID))
	})
}

func TestSQLiteHistoryManager_Add_MaxSize(t *testing.T) {
	maxPairs := 3 // Max 3 pairs of message/response (total 6 items)
	manager, cleanup := setupSQLiteManager(t, maxPairs)
	defer cleanup()
	userID := "user_max_sqlite"

	for i := 0; i < maxPairs; i++ {
		msg := fmt.Sprintf("msg%d", i)
		res := fmt.Sprintf("res%d", i)
		manager.Add(userID, testThreadID, msg, res)
	}

	var expectedFull []string
	for i := 0; i < maxPairs; i++ {
		expectedFull = append(expectedFull, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	assert.Equal(t, strings.Join(expectedFull, "\n"), manager.Get(userID, testThreadID))

	// Add one more pair, causing the oldest pair (msg0, res0) to be removed
	manager.Add(userID, testThreadID, "new_msg", "new_res")

	var expectedAfterOverflow []string
	for i := 1; i < maxPairs; i++ { // Starts from msg1
		expectedAfterOverflow = append(expectedAfterOverflow, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	expectedAfterOverflow = append(expectedAfterOverflow, "new_msg", "new_res")
	assert.Equal(t, strings.Join(expectedAfterOverflow, "\n"), manager.Get(userID, testThreadID))

	lines := strings.Split(manager.Get(userID, testThreadID), "\n")
	if maxPairs == 0 {
		assert.Condition(t, func() bool { return len(lines) == 1 && lines[0] == "" || len(lines) == 0 })
	} else {
		assert.Len(t, lines, maxPairs*2, "History length should be maxPairs * 2")
	}
}

func TestSQLiteHistoryManager_Clear(t *testing.T) {
	manager, cleanup := setupSQLiteManager(t, 10)
	defer cleanup()

	userID1 := "user_clear1_sqlite"
	userID2 := "user_clear2_sqlite"

	manager.Add(userID1, testThreadID, "msg1_u1t1", "res1_u1t1")
	manager.Add(userID1, testThreadID2, "msg1_u1t2", "res1_u1t2") // Different thread
	manager.Add(userID2, testThreadID, "msg1_u2t1", "res1_u2t1") // Different user

	require.NotEmpty(t, manager.Get(userID1, testThreadID))
	require.NotEmpty(t, manager.Get(userID1, testThreadID2))
	require.NotEmpty(t, manager.Get(userID2, testThreadID))

	manager.Clear(userID1, testThreadID) // Clear only for user1 in testThreadID

	assert.Empty(t, manager.Get(userID1, testThreadID))
	assert.NotEmpty(t, manager.Get(userID1, testThreadID2), "History for user1 in thread2 should remain")
	assert.NotEmpty(t, manager.Get(userID2, testThreadID), "History for user2 in thread1 should remain")

	manager.Clear("non_existent_user", testThreadID) // Should not panic
	manager.Clear(userID1, "non_existent_thread")   // Should not panic
}

func TestSQLiteHistoryManager_ClearAllByThreadID(t *testing.T) {
	manager, cleanup := setupSQLiteManager(t, 10)
	defer cleanup()

	userID1 := "user_cat1_sqlite"
	userID2 := "user_cat2_sqlite"

	manager.Add(userID1, testThreadID, "u1t1_msg1", "u1t1_res1")
	manager.Add(userID2, testThreadID, "u2t1_msg1", "u2t1_res1") // Same thread, different user
	manager.Add(userID1, testThreadID2, "u1t2_msg1", "u1t2_res1") // Different thread

	require.NotEmpty(t, manager.Get(userID1, testThreadID))
	require.NotEmpty(t, manager.Get(userID2, testThreadID))
	require.NotEmpty(t, manager.Get(userID1, testThreadID2))

	manager.ClearAllByThreadID(testThreadID)

	assert.Empty(t, manager.Get(userID1, testThreadID), "All history for user1 in testThreadID should be cleared")
	assert.Empty(t, manager.Get(userID2, testThreadID), "All history for user2 in testThreadID should be cleared")
	assert.NotEmpty(t, manager.Get(userID1, testThreadID2), "History for testThreadID2 should remain")

	manager.ClearAllByThreadID("non_existent_thread") // Should not panic
}

func TestSQLiteHistoryManager_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_persist")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbFilePath := filepath.Join(tempDir, dbFileName)
	maxSize := 5
	userID := "persist_user"
	threadID := "persist_thread"

	// Initial manager, add data
	manager1, err := NewSQLiteHistoryManager(tempDir, maxSize)
	require.NoError(t, err)
	sqliteManager1, ok := manager1.(*SQLiteHistoryManager)
	require.True(t, ok)

	manager1.Add(userID, threadID, "msg1", "res1")
	manager1.Add(userID, threadID, "msg2", "res2")
	expectedHistory1 := "msg1\nres1\nmsg2\nres2"
	assert.Equal(t, expectedHistory1, manager1.Get(userID, threadID))
	err = sqliteManager1.Close()
	require.NoError(t, err)

	// New manager instance, should load from the same DB file
	manager2, err := NewSQLiteHistoryManager(tempDir, maxSize)
	require.NoError(t, err)
	sqliteManager2, ok := manager2.(*SQLiteHistoryManager)
	require.True(t, ok)
	defer sqliteManager2.Close()

	assert.Equal(t, expectedHistory1, manager2.Get(userID, threadID), "History should persist across instances")

	// Add more data with the second instance
	manager2.Add(userID, threadID, "msg3", "res3")
	expectedHistory2 := "msg1\nres1\nmsg2\nres2\nmsg3\nres3"
	assert.Equal(t, expectedHistory2, manager2.Get(userID, threadID))
}
