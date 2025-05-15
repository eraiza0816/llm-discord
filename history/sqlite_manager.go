package history

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFileName        = "history.db"
	createTableStmt = `
CREATE TABLE IF NOT EXISTS thread_histories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    thread_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    history_json TEXT NOT NULL,
    last_updated_at TIMESTAMP NOT NULL,
    UNIQUE(thread_id, user_id)
);`
	insertHistoryStmt   = `INSERT INTO thread_histories (thread_id, user_id, history_json, last_updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(thread_id, user_id) DO UPDATE SET history_json = excluded.history_json, last_updated_at = excluded.last_updated_at;`
	getHistoryStmt      = `SELECT history_json FROM thread_histories WHERE thread_id = ? AND user_id = ? ORDER BY last_updated_at DESC LIMIT 1;`
	clearHistoryStmt    = `DELETE FROM thread_histories WHERE thread_id = ? AND user_id = ?;`
	clearAllByThreadIDStmt = `DELETE FROM thread_histories WHERE thread_id = ?;`
)

// SQLiteHistoryManager manages chat history using SQLite.
type SQLiteHistoryManager struct {
	db             *sql.DB
	mutex          sync.Mutex
	maxHistorySize int // 保持する履歴の最大数（メッセージと応答のペア数）
}

// NewSQLiteHistoryManager creates a new SQLiteHistoryManager.
// It ensures the database file and table are created.
// dbPath: ディレクトリパス。この下に history.db が作成される。
func NewSQLiteHistoryManager(dbPath string, maxHistorySize int) (HistoryManager, error) {
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, err
	}
	dbFilePath := filepath.Join(dbPath, dbFileName)

	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(createTableStmt); err != nil {
		return nil, err
	}

	return &SQLiteHistoryManager{
		db:             db,
		maxHistorySize: maxHistorySize,
	}, nil
}

// Add adds a message and response to the history for a given user and thread.
func (m *SQLiteHistoryManager) Add(userID, threadID, message, response string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var existingHistory []string
	var currentHistoryJSON string

	row := m.db.QueryRow(getHistoryStmt, threadID, userID)
	err := row.Scan(&currentHistoryJSON)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting existing history for user %s, thread %s: %v", userID, threadID, err)
		// エラーがあっても、新しい履歴で上書きを試みる
	}

	if currentHistoryJSON != "" {
		if err := json.Unmarshal([]byte(currentHistoryJSON), &existingHistory); err != nil {
			log.Printf("Error unmarshalling existing history for user %s, thread %s: %v", userID, threadID, err)
			// エラーがあっても、新しい履歴で上書きを試みる
			existingHistory = []string{}
		}
	}

	newHistory := append(existingHistory, message, response)

	// 履歴の最大サイズを超えた場合、古いものから削除 (メッセージと応答のペアなので *2 する)
	if len(newHistory) > m.maxHistorySize*2 {
		removeCount := len(newHistory) - (m.maxHistorySize * 2)
		newHistory = newHistory[removeCount:]
	}

	historyJSON, err := json.Marshal(newHistory)
	if err != nil {
		log.Printf("Error marshalling history for user %s, thread %s: %v", userID, threadID, err)
		return
	}

	_, err = m.db.Exec(insertHistoryStmt, threadID, userID, string(historyJSON), time.Now())
	if err != nil {
		log.Printf("Error adding history for user %s, thread %s: %v", userID, threadID, err)
	}
}

// Get retrieves the chat history for a given user and thread.
func (m *SQLiteHistoryManager) Get(userID, threadID string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var historyJSON string
	row := m.db.QueryRow(getHistoryStmt, threadID, userID)
	err := row.Scan(&historyJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return "" // 履歴がない場合は空文字列
		}
		log.Printf("Error getting history for user %s, thread %s: %v", userID, threadID, err)
		return ""
	}

	var history []string
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		log.Printf("Error unmarshalling history for user %s, thread %s: %v", userID, threadID, err)
		return ""
	}
	return strings.Join(history, "\n")
}

// Clear removes the chat history for a given user and thread.
func (m *SQLiteHistoryManager) Clear(userID, threadID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, err := m.db.Exec(clearHistoryStmt, threadID, userID)
	if err != nil {
		log.Printf("Error clearing history for user %s, thread %s: %v", userID, threadID, err)
	}
}

// ClearAllByThreadID removes all chat history for a given thread.
func (m *SQLiteHistoryManager) ClearAllByThreadID(threadID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, err := m.db.Exec(clearAllByThreadIDStmt, threadID)
	if err != nil {
		log.Printf("Error clearing all history for thread %s: %v", threadID, err)
	}
}

// Close closes the database connection.
func (m *SQLiteHistoryManager) Close() error {
	return m.db.Close()
}
