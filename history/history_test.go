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
	// 型アサーションで InMemoryHistoryManager かどうかチェック
	if _, ok := manager.(*InMemoryHistoryManager); !ok {
		t.Fatal("NewInMemoryHistoryManager did not return *InMemoryHistoryManager")
	}
	// maxHistorySize が設定されているかチェック（内部フィールドなので直接アクセスはできないけど、挙動で確認する）
}

func TestInMemoryHistoryManager_AddGet(t *testing.T) {
	manager := NewInMemoryHistoryManager(4) // 最大4要素 (メッセージ/応答で2要素 * 2ペア分)

	userID1 := "user1"
	userID2 := "user2"

	// --- User 1 ---
	t.Run("Add and Get for User 1", func(t *testing.T) {
		// 最初は空のはず
		if got := manager.Get(userID1); got != "" {
			t.Errorf("Expected empty history for %s, but got %q", userID1, got)
		}

		// 1ペア追加
		manager.Add(userID1, "hello", "hi there")
		expected1 := "hello\nhi there"
		if got := manager.Get(userID1); got != expected1 {
			t.Errorf("Expected %q for %s, but got %q", expected1, userID1, got)
		}

		// 2ペア目追加 (これで最大サイズ)
		manager.Add(userID1, "how are you?", "I'm good!")
		expected2 := "hello\nhi there\nhow are you?\nI'm good!"
		if got := manager.Get(userID1); got != expected2 {
			t.Errorf("Expected %q for %s, but got %q", expected2, userID1, got)
		}
	})

	// --- User 2 ---
	t.Run("Add and Get for User 2 (independent)", func(t *testing.T) {
		// User 2 はまだ空のはず
		if got := manager.Get(userID2); got != "" {
			t.Errorf("Expected empty history for %s, but got %q", userID2, got)
		}

		manager.Add(userID2, "test", "response")
		expected3 := "test\nresponse"
		if got := manager.Get(userID2); got != expected3 {
			t.Errorf("Expected %q for %s, but got %q", expected3, userID2, got)
		}

		// User 1 の履歴が変わってないか確認
		expectedUser1 := "hello\nhi there\nhow are you?\nI'm good!"
		if got := manager.Get(userID1); got != expectedUser1 {
			t.Errorf("History for %s should not change, expected %q, but got %q", userID1, expectedUser1, got)
		}
	})
}

func TestInMemoryHistoryManager_Add_MaxSize(t *testing.T) {
	maxSize := 6 // メッセージ/応答で 3ペア分
	manager := NewInMemoryHistoryManager(maxSize)
	userID := "user_max"

	// maxSize ちょうどまで追加
	for i := 0; i < maxSize/2; i++ {
		msg := fmt.Sprintf("msg%d", i)
		res := fmt.Sprintf("res%d", i)
		manager.Add(userID, msg, res)
	}

	// maxSize ちょうどか確認
	expectedFull := []string{}
	for i := 0; i < maxSize/2; i++ {
		expectedFull = append(expectedFull, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	expectedFullStr := strings.Join(expectedFull, "\n")
	if got := manager.Get(userID); got != expectedFullStr {
		t.Errorf("Expected full history %q, but got %q", expectedFullStr, got)
	}

	// さらに追加 (古いものが消えるはず)
	manager.Add(userID, "new_msg", "new_res")

	// 最初のペア (msg0, res0) が消えているか確認
	expectedAfterOverflow := []string{}
	for i := 1; i < maxSize/2; i++ { // i=0 をスキップ
		expectedAfterOverflow = append(expectedAfterOverflow, fmt.Sprintf("msg%d", i), fmt.Sprintf("res%d", i))
	}
	expectedAfterOverflow = append(expectedAfterOverflow, "new_msg", "new_res") // 新しいペアを追加
	expectedAfterOverflowStr := strings.Join(expectedAfterOverflow, "\n")

	if got := manager.Get(userID); got != expectedAfterOverflowStr {
		t.Errorf("Expected history after overflow %q, but got %q", expectedAfterOverflowStr, got)
	}

	// 内部のスライスの長さを確認（直接アクセスできないので、Getの結果の行数で代用）
	// 注意: この確認方法は厳密ではないが、挙動の目安にはなる
	lines := strings.Split(manager.Get(userID), "\n")
	if len(lines) != maxSize {
		// 空文字列の場合 lines は [""] で長さ 1 になるので、空の場合のチェックを追加
		if !(len(lines) == 1 && lines[0] == "") {
			t.Errorf("Expected history length to be %d lines, but got %d lines", maxSize, len(lines))
		} else if maxSize != 0 { // maxSizeが0でないのに空の場合もエラー
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

	// クリア前の確認
	if manager.Get(userID1) == "" {
		t.Errorf("History for %s should exist before clear", userID1)
	}
	if manager.Get(userID2) == "" {
		t.Errorf("History for %s should exist before clear", userID2)
	}

	// User 1 をクリア
	manager.Clear(userID1)

	// クリア後の確認
	if got := manager.Get(userID1); got != "" {
		t.Errorf("Expected empty history for %s after clear, but got %q", userID1, got)
	}
	// User 2 の履歴が残っているか確認
	expectedUser2 := "msg2\nres2"
	if got := manager.Get(userID2); got != expectedUser2 {
		t.Errorf("History for %s should remain after clearing %s, expected %q, but got %q", userID2, userID1, expectedUser2, got)
	}

	// 存在しないユーザーをクリアしてもエラーにならないこと
	manager.Clear("non_existent_user")
	// (特にチェックする項目はないが、panic しなければOK)
}
