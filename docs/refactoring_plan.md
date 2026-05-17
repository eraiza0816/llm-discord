# リファクタリング計画

## 現状分析

### アーキテクチャ概要

```
main.go
 ├─ config.LoadConfig() → config.Config
 ├─ discord.StartBot(cfg)
 │    ├─ discord/session.go    : DiscordSession インターフェース
 │    ├─ discord/handler.go    : メッセージハンドリング（主要ロジック）
 │    ├─ discord/chat_command.go  : /chat スラッシュコマンド
 │    ├─ discord/about_command.go : /about スラッシュコマンド
 │    ├─ discord/edit_command.go  : /edit スラッシュコマンド
 │    ├─ discord/reset_command.go : /reset スラッシュコマンド
 │    ├─ discord/custom_prompt.go : カスタムプロンプト管理
 │    ├─ discord/embeds.go     : Embed 分割ユーティリティ
 │    └─ discord/utils.go      : 便利関数
 ├─ chat.NewChat(cfg, historyMgr) → chat.Service
 │    ├─ chat/chat.go          : Gemini API 呼び出し（約345行）
 │    ├─ chat/ollama.go        : Ollama 呼び出し
 │    ├─ chat/prompt.go        : プロンプト構築
 │    ├─ chat/url_reader_service.go : URL 取得サービス
 │    └─ chat/utils.go         : 便利関数
 ├─ history.NewDuckDBHistoryManager() → history.HistoryManager
 │    ├─ history/duckdb_manager.go  : DuckDB 履歴管理
 │    ├─ history/history.go         : インターフェース定義
 │    ├─ history/audit_log.go       : 監査ログ
 │    ├─ history/audit_log_monitor.go : 監査ログモニター
 │    └─ history/url_downloader.go   : URL ダウンローダー
 └─ loader.LoadModelConfig() → loader.ModelConfig
      └─ loader/model.go      : model.json 読み込み
```

### 依存関係の流れ

```
main.go
  → config (起動時1回)
  → discord.StartBot(cfg) → discord (ハンドラ)
  → chat.NewChat(cfg, historyMgr) → chat (LLM)
     → history → DuckDB
     → loader (model.json)
```

---

## 問題点と改善提案

### 重大度: HIGH

#### 1. `chat/chat.go` の `GetResponse` が巨大（約200行）

- **問題**: 1つのメソッドに Ollama/Gemini の処理分岐、Function Calling、エラーハンドリング、リトライロジック、モデル切り替えが詰め込まれている。
- **改善**: 下記の単位に分割する。
  - `generateResponse` (共通のレスポンス生成フロー)
  - `callGemini` (Gemini API 呼び出し)
  - `callOllama` (Ollama API 呼び出し)
  - `handleFunctionCall` (Function Calling 処理)
  - `retryWithSecondaryModel` (429エラー時のセカンダリモデルへのフォールバック)

#### 2. ロガーが散在・重複

- **問題**:
  - `chat/chat.go` のパッケージレベル変数 `errorLogger` (`log/error.log` 出力) 
  - `log/app.log` への標準ログ出力
  - `history/audit_log.go` の監査ログ
  - ログの出力先・フォーマットが統一されていない
- **改善**: パッケージレベル変数のロガーを廃止し、標準 `log` パッケージに統一。どうしても必要な場合は `config.Config` 経由でロガーを注入する。

### 重大度: MEDIUM

#### 3. `config/config.go` の設定管理

- **問題**:
  - `.env`, `json/model.json`, `json/custom_model.json` の3ファイルを `LoadConfig` 1関数で読み込んでいる
  - 設定のバリデーションが不足（空値チェック程度）
- **改善**: ファイルごとに読み込み関数を分離し、明示的なバリデーションを追加する。

#### 4. HTTP クライアントの共通化

- **問題**: Ollama 呼び出し (`chat/ollama.go`) で `http.Client` を生成している。
- **改善**: `config.Config` に共通の `HTTPClient` を持たせるか、`chat` パッケージに HTTP クライアント管理を集約する。

#### 5. `config.Config` のミューテーション

- **問題**: `chat/chat.go` の `GetResponse` 内で `cfg.Model = secondaryModelCfg` のように `config.Config` のフィールドを書き換えている。Goのベストプラクティスに反する（共有状態のミューテーション）。
- **改善**: `Config` はイミュータブルに扱う。モデル切り替えが必要な場合は関数内のローカル変数で管理する。

### 重大度: LOW

#### 6. エラーハンドリングの一貫性

- **問題**: 一部のエラーは `log.Printf`、一部は `errorLogger.Printf`、一部は `return err` と混在。
- **改善**: 呼び出し元でハンドリングすべきエラーは返す。ログ出力のみで良いものは標準の `log` に統一。

#### 7. `chat/prompt.go` の分割

- **問題**: プロンプト構築、システムプロンプト固定文字列、Function Calling のツール定義が1ファイルに混在。
- **改善**: `chat/prompt_builder.go`（構築ロジック）、`chat/system_prompt.go`（固定プロンプト）、`chat/tools.go`（ツール定義）に分割。

#### 8. テスト不足

- **問題**: `chat` パッケージと `history` パッケージにテストファイルが存在しない。
- **改善**: ユニットテストを追加する。

---

## フェーズ分け

### フェーズ1（即効性・安全性の高い改善）

| # | タスク | 成果物 | 影響範囲 |
|---|--------|--------|----------|
| 1 | `chat/chat.go` の `GetResponse` を分割 | 可読性・保守性向上 | chat/chat.go |
| 2 | ロガーの統一（`errorLogger` の廃止） | ログ出力の一貫性 | chat/chat.go, chat/ollama.go |
| 3 | `config.LoadConfig` の責務分割 | 設定管理の明確化 | config/config.go |

### フェーズ2（中期的な改善）

| # | タスク | 成果物 | 影響範囲 |
|---|--------|--------|----------|
| 4 | HTTP クライアントの共通化 | 重複削除 | chat/ollama.go, chat/url_reader_service.go |
| 5 | `Config` ミューテーションの修正 | 安全性向上 | chat/chat.go, config/config.go |

### フェーズ3（長期的な改善）

| # | タスク | 成果物 | 影響範囲 |
|---|--------|--------|----------|
| 6 | `chat/prompt.go` の分割 | 明確な責務分離 | chat/prompt.go |
| 7 | テスト追加 | 品質保証 | chat/, history/ |
| 8 | エラーハンドリング統一 | 一貫性向上 | 全パッケージ |

---

## リファクタリング後のアーキテクチャイメージ

```
main.go
 ├─ config.LoadConfig()
 │    ├─ loadEnvConfig()        ← 新
 │    ├─ loadModelConfig()      ← 新
 │    └─ loadCustomPromptConfig() ← 新
 ├─ discord.StartBot(cfg)
 │    └─ (変更なし)
 ├─ chat.NewChat(cfg, historyMgr)
 │    ├─ chat/chat.go            ← 分割後のエントリポイント
 │    ├─ chat/gemini.go          ← 新 (GeminiAPI呼び出し)
 │    ├─ chat/ollama.go          ← 変更なし
 │    ├─ chat/prompt_builder.go  ← 新
 │    ├─ chat/system_prompt.go   ← 新
 │    ├─ chat/tools.go           ← 新
 │    ├─ chat/url_reader_service.go ← 変更なし
 │    └─ chat/utils.go           ← 変更なし
 └─ (以下同じ)
```

---

## 期待される効果

| 指標 | 現状 | 目標 |
|------|------|------|
| `chat/chat.go` の行数 | 345行 | 150行以下（エントリポイントのみ） |
| `GetResponse` のネスト深さ | 5レベル以上 | 3レベル以下 |
| ログ出力の手段 | 3種類（標準log, errorLogger, auditLog） | 2種類（標準log, auditLog） |
| 設定読み込み関数の責務 | 1関数で3ファイル | 3関数で各1ファイル |
