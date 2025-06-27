package history

import (
	"sync"
)

// HistoryMessage はチャット履歴の単一のメッセージを表します。
type HistoryMessage struct {
	Role    string `json:"role"`    // "user" または "model"
	Content string `json:"content"` // メッセージの内容
}

type HistoryManager interface {
	Add(userID string, threadID string, message string, response string) error // errorを返すように変更
	Get(userID string, threadID string) ([]HistoryMessage, error)             // 戻り値を []HistoryMessage, error に変更
	Clear(userID string, threadID string) error                               // errorを返すように変更
	ClearAllByThreadID(threadID string) error                                 // errorを返すように変更
	GetBotConversationCount(threadID, userID string) (int, error)
	Close() error
}

type InMemoryHistoryManager struct {
	histories      map[string][]HistoryMessage // ユーザーIDをキーにした履歴のスライス ([]HistoryMessage に変更)
	mutex          sync.Mutex
	maxHistorySize int
}

func NewInMemoryHistoryManager(maxSize int) (HistoryManager, error) { // 戻り値に error を追加
	return &InMemoryHistoryManager{
		histories:      make(map[string][]HistoryMessage),
		maxHistorySize: maxSize,
	}, nil
}

func (m *InMemoryHistoryManager) Add(userID string, threadID string, message string, response string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	userHistory := m.histories[userID]
	userHistory = append(userHistory, HistoryMessage{Role: "user", Content: message})
	userHistory = append(userHistory, HistoryMessage{Role: "model", Content: response})

	// 履歴の最大サイズを超えた場合、古いものから削除 (ペアで考慮)
	if len(userHistory) > m.maxHistorySize*2 {
		removeCount := len(userHistory) - m.maxHistorySize*2
		userHistory = userHistory[removeCount:]
	}
	m.histories[userID] = userHistory
	return nil
}

func (m *InMemoryHistoryManager) Get(userID string, threadID string) ([]HistoryMessage, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	history, ok := m.histories[userID]
	if !ok {
		return []HistoryMessage{}, nil // 履歴がない場合は空のスライスを返す
	}
	return history, nil
}

func (m *InMemoryHistoryManager) Clear(userID string, threadID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.histories, userID)
	return nil
}

func (m *InMemoryHistoryManager) ClearAllByThreadID(threadID string) error {
	// InMemoryHistoryManager ではこの操作はサポートしないか、
	// もしくは全ユーザーの履歴をクリアするなどの振る舞いになる。
	// ここでは何もしない。
	return nil
}

func (m *InMemoryHistoryManager) GetBotConversationCount(threadID, userID string) (int, error) {
	// InMemoryHistoryManagerではこの機能は未実装とする
	return 0, nil
}

func (m *InMemoryHistoryManager) Close() error {
	return nil
}
