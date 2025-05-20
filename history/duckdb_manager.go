package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

type DuckDBHistoryManager struct {
	db *sql.DB
}

func NewDuckDBHistoryManager() (*DuckDBHistoryManager, error) {
	dbPath := filepath.Join("data", "chat_history.duckdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("データディレクトリの作成に失敗しました: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("DuckDBデータベースへの接続に失敗しました: %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS thread_histories (
		thread_id VARCHAR NOT NULL,
		user_id VARCHAR NOT NULL,
		history_json TEXT,
		last_updated_at TIMESTAMP,
		PRIMARY KEY (thread_id, user_id)
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("thread_historiesテーブルの作成に失敗しました: %w", err)
	}

	log.Println("DuckDB HistoryManagerが正常に初期化されました。データベースパス:", dbPath)
	return &DuckDBHistoryManager{db: db}, nil
}

func (m *DuckDBHistoryManager) Add(userID, threadID, message, response string) error {
	var currentHistoryJSON string
	querySQL := "SELECT history_json FROM thread_histories WHERE user_id = ? AND thread_id = ?;"
	err := m.db.QueryRow(querySQL, userID, threadID).Scan(&currentHistoryJSON)

	var history []HistoryMessage
	if err != nil {
		if err == sql.ErrNoRows {
			history = []HistoryMessage{}
		} else {
			return fmt.Errorf("既存履歴のクエリ実行に失敗しました: %w", err)
		}
	} else {
		if err := json.Unmarshal([]byte(currentHistoryJSON), &history); err != nil {
			return fmt.Errorf("既存履歴のJSONデシリアライズに失敗しました: %w", err)
		}
	}

	history = append(history, HistoryMessage{Role: "user", Content: message})
	history = append(history, HistoryMessage{Role: "model", Content: response})

	historyJSON, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("履歴のJSONシリアライズに失敗しました: %w", err)
	}

	// DuckDBではUPSERT構文が利用可能 (ON CONFLICT DO UPDATE)
	upsertSQL := `
	INSERT INTO thread_histories (thread_id, user_id, history_json, last_updated_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT (thread_id, user_id) DO UPDATE SET
		history_json = excluded.history_json,
		last_updated_at = excluded.last_updated_at;`

	_, err = m.db.Exec(upsertSQL, threadID, userID, string(historyJSON), time.Now())
	if err != nil {
		return fmt.Errorf("履歴の挿入または更新に失敗しました: %w", err)
	}
	return nil
}

// Get は指定されたユーザーとスレッドの履歴を取得します。
// データベースからは全ての履歴を取得し、最新20ペアを返します。
func (m *DuckDBHistoryManager) Get(userID, threadID string) ([]HistoryMessage, error) {
	var historyJSON string
	querySQL := "SELECT history_json FROM thread_histories WHERE user_id = ? AND thread_id = ?;"
	err := m.db.QueryRow(querySQL, userID, threadID).Scan(&historyJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			// 履歴が見つからない場合は空のスライスとnilエラーを返す
			return []HistoryMessage{}, nil
		}
		return nil, fmt.Errorf("履歴のクエリ実行に失敗しました: %w", err)
	}

	var fullHistory []HistoryMessage
	if err := json.Unmarshal([]byte(historyJSON), &fullHistory); err != nil {
		return nil, fmt.Errorf("履歴のJSONデシリアライズに失敗しました: %w", err)
	}

	// 最新20ペアを抽出する
	maxHistorySizeToReturn := 20
	if len(fullHistory) > maxHistorySizeToReturn*2 {
		return fullHistory[len(fullHistory)-maxHistorySizeToReturn*2:], nil
	}

	return fullHistory, nil
}

func (m *DuckDBHistoryManager) Clear(userID, threadID string) error {
	deleteSQL := "DELETE FROM thread_histories WHERE user_id = ? AND thread_id = ?;"
	result, err := m.db.Exec(deleteSQL, userID, threadID)
	if err != nil {
		return fmt.Errorf("履歴の削除に失敗しました: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("削除された行数の取得に失敗しました: %v", err)
	}
	if rowsAffected == 0 {
		log.Printf("ユーザー %s のスレッド %s の履歴は見つからなかったため、削除されませんでした。", userID, threadID)
	}
	return nil
}

func (m *DuckDBHistoryManager) ClearAllByThreadID(threadID string) error {
	deleteSQL := "DELETE FROM thread_histories WHERE thread_id = ?;"
	result, err := m.db.Exec(deleteSQL, threadID)
	if err != nil {
		return fmt.Errorf("スレッド %s の全履歴の削除に失敗しました: %w", threadID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("スレッド %s の削除された行数の取得に失敗しました: %v", threadID, err)
	}
	if rowsAffected == 0 {
		log.Printf("スレッド %s の履歴は見つからなかったため、削除されませんでした。", threadID)
	} else {
		log.Printf("スレッド %s の履歴 %d 件を削除しました。", threadID, rowsAffected)
	}
	return nil
}

func (m *DuckDBHistoryManager) Close() error {
	if m.db != nil {
		log.Println("DuckDBデータベース接続を閉じます。")
		return m.db.Close()
	}
	return nil
}
