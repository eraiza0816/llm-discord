package history

import (
	"strings"
	"sync"
)

// HistoryManager は、ユーザーごとの対話履歴を管理するインターフェースだよ！
type HistoryManager interface {
	// Add は、指定されたユーザーIDの履歴にメッセージと応答を追加するよ。
	// 履歴が最大サイズを超えたら、古いものから削除するんだ！
	Add(userID string, message string, response string)
	// Get は、指定されたユーザーIDの履歴を結合した文字列として取得するよ。
	Get(userID string) string
	// Clear は、指定されたユーザーIDの履歴を削除するよ。
	Clear(userID string)
}

// InMemoryHistoryManager は、メモリ上で履歴を管理する HistoryManager の実装だよ！
type InMemoryHistoryManager struct {
	histories      map[string][]string // ユーザーIDをキーにした履歴のスライス
	mutex          sync.Mutex          // 複数 goroutine からのアクセスを保護するためのミューテックス
	maxHistorySize int                 // 保持する履歴の最大数（メッセージと応答のペア数ではない点に注意！）
}

// NewInMemoryHistoryManager は、新しい InMemoryHistoryManager を作成して返すよ！
func NewInMemoryHistoryManager(maxSize int) HistoryManager {
	return &InMemoryHistoryManager{
		histories:      make(map[string][]string),
		maxHistorySize: maxSize,
		// mutex はゼロ値で初期化されるからこれでOK！
	}
}

// Add は、ユーザーの履歴にメッセージと応答を追加するよ。
func (m *InMemoryHistoryManager) Add(userID string, message string, response string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// ユーザーの履歴を取得（なければ新しく作る）
	userHistory := m.histories[userID]

	// メッセージと応答を追加
	userHistory = append(userHistory, message, response)

	// 履歴が最大サイズを超えていたら、古いものから削除
	// スライスの要素数が maxSize を超えたら、超過分を先頭から削除する
	if len(userHistory) > m.maxHistorySize {
		// 削除する要素数を計算
		removeCount := len(userHistory) - m.maxHistorySize
		// スライスを再作成して古い要素を削除
		userHistory = userHistory[removeCount:]
	}

	// 更新した履歴をマップに保存
	m.histories[userID] = userHistory
}

// Get は、ユーザーの履歴を結合した文字列として取得するよ。
func (m *InMemoryHistoryManager) Get(userID string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 履歴を取得して、改行で結合して返す
	return strings.Join(m.histories[userID], "\n")
}

// Clear は、ユーザーの履歴を削除するよ。
func (m *InMemoryHistoryManager) Clear(userID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// マップから該当ユーザーのキーを削除
	delete(m.histories, userID)
}