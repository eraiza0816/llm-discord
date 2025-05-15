package history

import (
	"strings"
	"sync"
)

type HistoryManager interface {
	Add(userID string, threadID string, message string, response string)
	Get(userID string, threadID string) string
	Clear(userID string, threadID string)
	ClearAllByThreadID(threadID string) // 特定のスレッドの全ユーザー履歴をクリア（オプション）
	Close() error                       // 追加
}

type InMemoryHistoryManager struct {
	histories      map[string][]string // ユーザーIDをキーにした履歴のスライス
	mutex          sync.Mutex          // 複数 goroutine からのアクセスを保護するためのミューテックス
	maxHistorySize int                 // 保持する履歴の最大数（メッセージと応答のペア数ではない点に注意！）
}

func NewInMemoryHistoryManager(maxSize int) HistoryManager {
	return &InMemoryHistoryManager{
		histories:      make(map[string][]string),
		maxHistorySize: maxSize,
		// mutex はゼロ値で初期化されるからこれでOK！
	}
}

// InMemoryHistoryManager はメモリ上で履歴を管理しますが、スレッドIDを考慮するようには変更しません。
// スレッド対応は SQLiteHistoryManager で行います。
// 既存の InMemoryHistoryManager は、スレッド非対応のチャンネルでの一時的な履歴管理や、
// テスト用途などで引き続き利用される可能性があります。
// そのため、インターフェースの変更に合わせてダミーの threadID 引数を追加しますが、
// 実際の動作は従来のユーザーID単位のままです。

func (m *InMemoryHistoryManager) Add(userID string, threadID string, message string, response string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// threadID は InMemoryHistoryManager では使用しない
	userHistory := m.histories[userID]
	userHistory = append(userHistory, message, response)

	if len(userHistory) > m.maxHistorySize {
		removeCount := len(userHistory) - m.maxHistorySize
		userHistory = userHistory[removeCount:]
	}
	m.histories[userID] = userHistory
}

func (m *InMemoryHistoryManager) Get(userID string, threadID string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// threadID は InMemoryHistoryManager では使用しない
	return strings.Join(m.histories[userID], "\n")
}

func (m *InMemoryHistoryManager) Clear(userID string, threadID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// threadID は InMemoryHistoryManager では使用しない
	delete(m.histories, userID)
}

// ClearAllByThreadID は InMemoryHistoryManager では実装しません（または全ユーザーの全履歴をクリアするなどの代替動作）。
// 今回は何もしないようにします。
func (m *InMemoryHistoryManager) ClearAllByThreadID(threadID string) {
	// InMemoryHistoryManager ではこの操作はサポートしないか、
	// もしくは全ユーザーの履歴をクリアするなどの振る舞いになる。
	// ここでは何もしない。
}

// Close implements HistoryManager.
// For InMemoryHistoryManager, there's nothing to close.
func (m *InMemoryHistoryManager) Close() error {
	return nil // 何もクローズするものがないのでnilを返す
}
