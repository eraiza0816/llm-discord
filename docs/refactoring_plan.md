# リファクタリング計画書

作成日: 2026-05-18
対象: llm-discord-go

## 1. 概要

本ドキュメントは llm-discord-go のコードベースを分析し、アーキテクチャ改善、保守性向上、テスト容易性向上を目的としたリファクタリング計画をまとめる。

## 2. 現状分析

### 2.1 全体アーキテクチャ

```
main.go
├── config/config.go
├── discord/ (Discord Bot ハンドリング)
│   ├── discord.go (エントリポイント、session管理)
│   ├── handler.go (全interactionの受付)
│   ├── about_command.go
│   ├── chat_command.go
│   ├── edit_command.go
│   ├── reset_command.go
│   ├── custom_prompt.go
│   ├── session.go
│   ├── embeds.go
│   ├── utils.go
│   └── _test.go
├── chat/ (LLMとの通信)
│   ├── chat.go (Invoke: プロバイダディスパッチ)
│   ├── ollama.go (Ollama実装)
│   ├── openai.go (OpenAI実装)
│   ├── service.go (Service層)
│   ├── url_reader_service.go
│   ├── prompt.go (プロンプト管理)
│   └── utils.go
├── history/ (会話履歴管理)
│   ├── history.go
│   ├── audit_log.go
│   ├── audit_log_monitor.go
│   ├── duckdb_manager.go
│   └── url_downloader.go
├── loader/ (モデル定義)
│   ├── model.go
│   └── model_test.go
└── docs/
```

### 2.2 発見された問題点

#### [P0 - クリティカル] バグ

1. **chat/chat.go `Invoke` - context 未使用**
   - `ctx context.Context` を受け取っているが、内部で `context.Background()` を生成して使用している
   - キャンセレーションやタイムアウトが効かない

2. **chat/ollama.go `Invoke` - context 未使用・引数順序不整合**
   - `ctx context.Context` が引数にあるが使われていない
   - `Invoke(ctx, url, image)` のインタフェースに対して、内部実装は引数順序や解釈が統一されていない

#### [P1 - 重要] 設計の問題

3. **discord/handler.go - 単一関数に全コマンド処理が集約**
   - `HandleInteraction` が 200 行を超え、すべての slash command を switch で処理
   - 新規コマンド追加のたびに関数が肥大化
   - テストが困難

4. **プロバイダ選択が文字列ベース**
   - `provider` フィールドが単なる string
   - chat.go 内部で `switch provider { case "openai": ... case "ollama": ... }`
   - 新しいプロバイダ追加時に chat.go の修正が必要

5. **discord パッケージと chat パッケージの結合**
   - discord が chat の具体実装に依存している
   - discord/discord.go が `chat.Invoke(provider, ...)` を直接呼ぶ
   - インタフェースを介した依存関係になっておらず、テスト差し替えが困難

#### [P2 - 改善推奨] コード品質

6. **loader/model.go - 2つのLoad関数で重複**
   - `LoadModel()` と `LoadCustomModel()` が類似処理
   - モデル選択ロジックが分岐で複雑化

7. **chat/url_reader_service.go - 責務の混在**
   - URL読み取りとプロンプト生成が同一ファイルに混在
   - URL解決がチャットフローと密結合

8. **discord/custom_prompt.go - 管理方法**
   - カスタムプロンプトがコード上のファイルシステムパスに依存

9. **history/ パッケージ - 1ファイルに複数責務**
   - duckdb_manager.go に DB 管理と履歴追加ロジックが混在
   - audit_log.go に監査ログと履歴管理の共通コード

10. **エラーハンドリングの不整合**
    - 一部は `return err`、一部は `log.Fatal`、一部はログ出力のみ
    - エラーのラップが不十分で原因追跡が困難

11. **テスト不足**
    - loader/model_test.go のみテストあり
    - discord/handler_test.go はテストファイルがあるが空の可能性
    - モックを使ったユニットテストが存在しない

## 3. リファクタリング計画

### Phase 1: バグ修正（優先度: P0）

| # | タスク | ファイル | 内容 |
|---|--------|----------|------|
| 1 | context伝播の修正 | chat/chat.go, chat/ollama.go, chat/openai.go | `context.Background()` を引数の `ctx` に置き換え |
| 2 | 引数順序の統一 | chat/ollama.go, chat/openai.go | インタフェース呼び出しと実装の引数順序を統一 |

### Phase 2: アーキテクチャ改善（優先度: P1）

#### 2-1: ChatProvider インタフェースの導入

```go
// chat/provider.go (新規)
type ChatProvider interface {
    Invoke(ctx context.Context, prompt string, imageURL string) (string, error)
    Name() string
}
```

- OllamaProvider, OpenAIProvider がこのインタフェースを実装
- chat/chat.go の switch 文を削除し、マップベースのプロバイダ管理に変更
- 新しいプロバイダ追加時に chat.go の修正不要に

#### 2-2: Command パターン/ルーターの導入 (discord/)

```go
// discord/command.go (新規)
type CommandHandler interface {
    Handle(session *discordgo.Session, interaction *discordgo.InteractionCreate, cfg *Config) error
}
```

- 各コマンド（about, chat, edit, reset, custom_prompt）がインタフェースを実装
- `HandleInteraction` の switch 文を削除し、登録されたハンドラに委譲
- テストは各ハンドラ単位で独立して記述可能に

#### 2-3: discord と chat の依存関係をインタフェース化

- discord パッケージは chat.ChatProvider インタフェースにのみ依存
- 実際のプロバイダは main.go で注入

### Phase 3: コード品質改善（優先度: P2）

#### 3-1: loader/model.go リファクタリング

- `LoadModel()` と `LoadCustomModel()` を統合
- 設定情報を構造体で受け取る形式に変更

#### 3-2: エラーハンドリング統一

- すべてのエラーに `fmt.Errorf("context: %w", err)` 形式でコンテキスト付与
- `log.Fatal` を main.go 以外で使用禁止
- パッケージレベルのエラー型の検討

#### 3-3: URL読み取り機能の分離

- `url_reader_service.go` の責務を分割
  - URL読み取り（外部API呼び出し）とプロンプト加工を分離
  - 必要に応じて独立したパッケージ化

#### 3-4: history/ パッケージ整理

- `HistoryManager` インタフェースの明示
- データアクセス層とビジネスロジックの分離
- DuckDB依存をインタフェースの背後に隠蔽

### Phase 4: テスト基盤の整備（優先度: P1）

1. インタフェース導入後、モックを使ったユニットテストを実装
2. 各パッケージに `_test.go` ファイルを作成
3. CIでのテスト実行設定

## 4. 期待される効果

| 項目 | 現状 | 改善後 |
|------|------|--------|
| 新規コマンド追加 | handler.go + discord.go の修正が必要 | 新しいハンドラファイル追加のみ |
| 新規プロバイダ追加 | chat.go のswitch文修正 | provider.go に構造体追加のみ |
| ユニットテスト | ほぼ不可能 | インタフェースモックで単独テスト可能 |
| context伝搬 | 実質未使用 | 全チェーンで適切に伝搬 |
| 循環依存 | 発生していないが監視が必要 | 明示的な依存方向で管理 |
| コード理解容易性 | 中 | 高（責務分離完了） |

## 5. 非機能要件・制約

- 外部パッケージは新規導入しない（go mod に追加しない）
- Go 標準ライブラリのみで実装
- 既存の CLI からの動作（引数・環境変数）は維持
- データベーススキーマ変更なし
- 段階的にリファクタリングし、各Phase完了後に動作確認

## 6. 優先順位提案

1. Phase 1（バグ修正）を最優先
2. Phase 2-1（ChatProviderインタフェース）+ Phase 2-3（依存性注入）を同時進行
3. Phase 2-2（Commandパターン）を次に実施
4. Phase 3（コード品質）、Phase 4（テスト）は並行して実施可能なものから着手

---

以上がリファクタリング計画の全体像である。実際の実装着手時には各Phaseをさらにタスク分割し、変更単位を小さくして進めることを推奨する。
