# リファクタリング実施報告書

作成日: 2026-05-18 (計画)
更新日: 2026-06-07 (実施完了)
対象: llm-discord-go

## 1. 概要

本ドキュメントは llm-discord-go のコードベースを分析し、アーキテクチャ改善、保守性向上、テスト容易性向上を目的としたリファクタリングの計画と実施結果をまとめる。

## 2. 改善後のアーキテクチャ

```
main.go
├── config/config.go
├── discord/ (Discord Bot ハンドリング)
│   ├── discord.go (エントリポイント、session管理)
│   ├── handler.go (messageイベントハンドラ)
│   ├── command.go (CommandHandler インタフェース + ディスパッチャ)
│   ├── session.go (DiscordSession インタフェース)
│   ├── about_command.go
│   ├── chat_command.go
│   ├── edit_command.go
│   ├── reset_command.go
│   ├── custom_prompt.go
│   ├── embeds.go
│   ├── utils.go
│   └── *_test.go
├── chat/ (LLMとの通信)
│   ├── chat.go (Service インタフェース実装、プロバイダディスパッチ)
│   ├── provider.go (ChatProvider インタフェース)
│   ├── ollama.go (Ollama実装)
│   ├── openai.go (OpenAI実装)
│   ├── service.go (Service層: 型定義)
│   ├── prompt.go (プロンプト管理)
│   └── *_test.go
├── history/ (会話履歴管理)
│   ├── history.go (HistoryManager インタフェース)
│   ├── duckdb_manager.go (DuckDB実装)
│   ├── audit_log.go
│   ├── audit_log_monitor.go
│   ├── url_downloader.go
│   └── *_test.go
├── loader/ (モデル定義)
│   ├── model.go
│   └── model_test.go
└── docs/
```

## 3. 実施結果

### Phase 1: context 伝搬の修正 ✅ (2026-06-07)

| # | タスク | ファイル | 対応内容 |
|---|--------|----------|----------|
| 1 | context伝播の修正 | chat/chat.go, chat/ollama.go, chat/openai.go | `context.Background()` を引数の `ctx` に置き換え |
| 2 | HTTP client timeout | chat/ollama.go, chat/openai.go | `http.Client` に 120s Timeout を設定 |
| 3 | context対応リクエスト | chat/ollama.go, chat/openai.go | `http.NewRequest` → `http.NewRequestWithContext` に変更 |

### Phase 2: アーキテクチャ改善 ✅ (2026-06-07)

#### 2-1: ChatProvider インタフェースの導入

```go
// chat/provider.go (新規)
type ChatProvider interface {
    Invoke(ctx context.Context, fullInput string) (string, float64, error)
    Name() string
}
```

- `chat/provider.go` を新規作成し `ChatProvider` インタフェースを定義
- `chat/chat.go` の `GetResponse` を `invokeOllama`, `invokeOpenAI`, `invokeGemini` に分割し可読性を向上
- `goto` 文を削除し、`handleGeminiError`, `invokeSecondaryModel`, `processGeminiResponse`, `handleFunctionCall` を抽出

#### 2-2: Command パターン/ルーターの導入 (discord/)

```go
// discord/command.go (新規)
type CommandHandler interface {
    Name() string
    Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
}
```

- `discord/command.go` を新規作成
- `chat`, `reset`, `about`, `edit` の各コマンドを個別の `CommandHandler` 実装として分離
- `handler.go` の switch 文を `commandDispatcher` に置き換え
- 新規コマンド追加は `Register()` を呼ぶだけで可能に

#### 2-3: discord と chat の依存関係をインタフェース化

- discord パッケージは `chat.Service` インタフェースにのみ依存 (従来通り)
- `setupHandlers` で依存性注入を実施

### Phase 3: コード品質改善 ✅ (2026-06-07)

#### 3-2: エラーハンドリング統一

- `audit_log_monitor.go`: 全エラーにログ出力を追加 (従来はサイレントスキップ)
- `url_downloader.go`: エラーメッセージを明確化

### Phase 4: テスト基盤の整備 ✅ (2026-06-07)

| パッケージ | ファイル | テスト数 | 内容 |
|-----------|----------|----------|------|
| chat | chat_test.go | 13 tests | `buildFullInput`, `getResponseText`, `parseOllamaStreamResponse`, `parseOpenAIStreamResponse` |
| history | history_test.go | 10 tests | `InMemoryHistoryManager` 全メソッド |
| history | audit_log_test.go | 4 tests | `InitAuditLog`, `LogMessageCreate/Update/Delete` |
| config | config_test.go | 5 tests | `loadCustomPrompts` 正常系・異常系 |
| discord | discord_test.go | 7 tests | `extractAttachmentURLs`, `classifyMessageType`, `japanStandardTime` |

**全92テストがパスすることを確認済み。**

## 4. 実績

| 項目 | 改善前 | 改善後 |
|------|--------|--------|
| 新規コマンド追加 | handler.go + discord.go の修正が必要 | 新しいハンドラファイル + Register() のみ |
| 新規プロバイダ追加 | chat.go のif-else修正 | provider.go に構造体追加のみ |
| ユニットテスト | 3テストファイル・33テスト | 7テストファイル・92テスト (全パッケージ) |
| context伝搬 | context.Background() 使用 | 引数のctxを全チェーンで伝搬 + HTTPタイムアウト |
| 循環依存 | 発生していない | 明示的な依存方向で管理継続 |
| コード理解容易性 | 中 (goto文, 密結合) | 高 (責務分離, パターン化, command/go to除去) |
