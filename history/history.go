package history

import (
	"strings"
	"sync"
)

type HistoryManager interface {
	Add(userID string, message string, response string)
	Get(userID string) string
	Clear(userID string)
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

func (m *InMemoryHistoryManager) Add(userID string, message string, response string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	userHistory := m.histories[userID]

	userHistory = append(userHistory, message, response)

	if len(userHistory) > m.maxHistorySize {
		removeCount := len(userHistory) - m.maxHistorySize
		userHistory = userHistory[removeCount:]
	}

	m.histories[userID] = userHistory
}

func (m *InMemoryHistoryManager) Get(userID string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return strings.Join(m.histories[userID], "\n")
}

func (m *InMemoryHistoryManager) Clear(userID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.histories, userID)
}
