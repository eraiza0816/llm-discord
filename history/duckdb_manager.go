package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb" // DuckDB driver
)

// DuckDBHistoryManager はDuckDBを使用してチャット履歴を管理します。
type DuckDBHistoryManager struct {
	db *sql.DB
}

// NewDuckDBHistoryManager は新しいDuckDBHistoryManagerを初期化します。
// データベースファイルは "data" ディレクトリ内に "chat_history.duckdb" という名前で作成されます。
func NewDuckDBHistoryManager() (*DuckDBHistoryManager, error) {
	dbPath := filepath.Join("data", "chat_history.duckdb")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("データディレクトリの作成に失敗しました: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("DuckDBデータベースへの接続に失敗しました: %w", err)
	}

	// テーブルが存在しない場合は作成
	// SQLiteとは異なり、DuckDBではJSON型を直接サポートしているため、history_jsonはJSON型として定義できる
	// ただし、互換性のためにTEXT型として保存し、Go側でJSONのシリアライズ/デシリアライズを行うアプローチも一般的
	// ここではTEXT型として扱う
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

// Add は指定されたユーザーとスレッドの履歴にメッセージと応答を追加します。
func (m *DuckDBHistoryManager) Add(userID, threadID, message, response string) error {
	history, err := m.Get(userID, threadID)
	if err != nil {
		// エラーが "履歴が見つかりません" の場合、新しい履歴を作成
		if err.Error() != fmt.Sprintf("ユーザー %s のスレッド %s の履歴が見つかりません", userID, threadID) {
			return fmt.Errorf("履歴の取得に失敗しました: %w", err)
		}
		history = []HistoryMessage{} // 空の履歴で初期化
	}

	history = append(history, HistoryMessage{Role: "user", Content: message})
	history = append(history, HistoryMessage{Role: "model", Content: response})

	// 履歴の最大サイズを超えた場合、古いものから削除 (SQLiteManagerと同様のロジック)
	// model.json から max_history_size を取得する必要があるが、ここでは固定値とするか、
	// もしくは HistoryManager の初期化時に渡すようにインターフェースを変更する必要がある。
	// 現状のインターフェースでは直接 model.json を参照できないため、一旦固定値で実装する。
	// TODO: max_history_size を設定可能にする
	maxHistorySize := 20 // 仮の最大履歴サイズ
	if len(history) > maxHistorySize*2 { // メッセージとレスポンスのペアなので *2
		history = history[len(history)-maxHistorySize*2:]
	}

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

	var history []HistoryMessage
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		return nil, fmt.Errorf("履歴のJSONデシリアライズに失敗しました: %w", err)
	}
	return history, nil
}

// Clear は指定されたユーザーとスレッドの履歴をクリアします。
func (m *DuckDBHistoryManager) Clear(userID, threadID string) error {
	deleteSQL := "DELETE FROM thread_histories WHERE user_id = ? AND thread_id = ?;"
	result, err := m.db.Exec(deleteSQL, userID, threadID)
	if err != nil {
		return fmt.Errorf("履歴の削除に失敗しました: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("削除された行数の取得に失敗しました: %v", err) // エラーログは出すが、処理は続行
	}
	if rowsAffected == 0 {
		log.Printf("ユーザー %s のスレッド %s の履歴は見つからなかったため、削除されませんでした。", userID, threadID)
	}
	return nil
}

// ClearAllByThreadID は指定されたスレッドの全ユーザーの履歴をクリアします。
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

// Close はデータベース接続を閉じます。
func (m *DuckDBHistoryManager) Close() error {
	if m.db != nil {
		log.Println("DuckDBデータベース接続を閉じます。")
		return m.db.Close()
	}
	return nil
}
