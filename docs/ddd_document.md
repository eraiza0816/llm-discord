# ドメイン駆動設計 ドキュメント

## プロジェクトの目的と概要

このプロジェクトは、Discord上でLLMのGeminiとOllamaとチャットできるDiscord Bot「ぺちこ」を開発しています。
ユーザーはDiscordのインターフェースを通じてGeminiまたはOllamaと対話できます。

## 主要な機能

- LLMとのチャット機能: ユーザーはテキストメッセージを送信し、GeminiまたはOllamaからの応答を受信できます。
- チャット履歴のリセット機能: ユーザーは`/reset`コマンドを使用してチャット履歴をクリアできます。
- BOTの説明表示機能: `/about`コマンドで、json/model.json に定義されたBOTの説明を表示します。
- プロンプト編集機能: ユーザーは`/edit`コマンドを使用してプロンプトを編集できます。
  - `/edit`コマンドで"delete"と送信された場合、custom_model.json から該当ユーザーの行を削除します。

## ドメインに関する用語（ユビキタス言語）

- Bot: Discord Bot
- Gemini: GoogleのLLM
- Ollama: ローカルで動作するLLM
- LLM: GeminiやOllamaなどの大規模言語モデル
- チャット履歴 (History): ユーザーごとのLLMとの対話の記録。
- コマンド: Botへの指示 (例: `/chat`, `/reset`, `/about`)
- プロンプト: GeminiまたはOllamaへの指示文。ユーザーごと、またはデフォルトの指示文が設定可能。
- ぺちこ: このBotの名前
- モデル設定 (ModelConfig): `model.json` から読み込まれるBotの基本設定。
- カスタムプロンプト設定 (CustomPromptConfig): `custom_model.json` から読み書きされるユーザー固有のプロンプト設定。

## コンテキストマップ

- サブドメイン:
  - チャット: ユーザーとLLMとの対話の管理
  - コマンド処理: ユーザーからのコマンドの解析と実行
  - 設定管理: Botの設定の読み込みと管理 (`.env`, `model.json`, `custom_model.json`)
  - **履歴管理**: ユーザーごとのチャット履歴の永続化と取得 (新規)
- 境界づけられたコンテキスト:
  - Discord Bot: Discord APIとのインターフェース (`discord` パッケージ)
  - LLMクライアント: LLM API (Gemini, Ollama) とのインターフェース (`chat` パッケージ)
  - 設定ローダー: 設定ファイルの読み込み (`loader`, `config` パッケージ)
  - 履歴マネージャー: チャット履歴の管理 (`history` パッケージ)
- 関係:
  - Discord BotはLLMクライアントを利用してチャット機能を提供します。
  - Discord Botはユーザーからのコマンドを受け付け、適切な処理を呼び出します。
  - LLMクライアントは履歴マネージャーを利用して対話履歴を取得・保存します。
  - Discord Botは設定ローダーから設定を読み込みます。
  - Discord Botは履歴マネージャーを利用して履歴をリセットします。

## ドメインモデル

### エンティティ

- **ユーザー (User)**
  - ID: DiscordのユーザーID (userID)。Discord APIから取得。
  - ユーザー名: Discordのユーザー名 (username)。Discord APIから取得。
  - 役割: (将来の拡張を見据えて) ユーザーの役割を定義。

- **メッセージ (Message)**
  - 内容: ユーザーまたはBotからのメッセージ (content)。
  - タイムスタンプ: メッセージの送信時刻 (timestamp)。
  - ユーザーID: メッセージを送信したユーザーのID (userID)。
  - 送信者: メッセージの送信者 (ユーザーまたはBot)。

### 値オブジェクト

- **タイムスタンプ (Timestamp)**
  - メッセージの送信時刻。

- **モデル設定 (ModelConfig)** (`loader/model.go`)
  - モデル名: 使用するLLMのモデル名 (model_name)。
  - アイコン: Botのアイコン (icon)。
  - プロンプト: LLMへの指示文 (prompts)。ユーザーごと、またはデフォルトの指示文が設定可能。
  - BOTの説明: BOTの説明 (about)。
  - チャット履歴の最大サイズ (max_history_size): 保持するチャット履歴の最大件数。
  - Ollama設定 (OllamaConfig): Ollama APIに関する設定。

- **カスタムプロンプト設定 (CustomPromptConfig)** (`discord/types.go`)
  - ユーザープロンプト (Prompts): ユーザー名をキーとしたカスタムプロンプトのマップ。

### ドメインサービス / アプリケーションサービス / インフラストラクチャサービス

- **ChatService** (`chat/chat.go`)
  - 役割: LLMとのやり取りを行う。履歴管理は `HistoryManager` に移譲。
  - メソッド:
    - `GetResponse(userID, username, message, timestamp, prompt)`: LLMを呼び出し、応答、処理時間、モデル名を取得する。内部で `HistoryManager` を利用して履歴を取得・追加する。
    - `Close()`: LLMクライアントを閉じる。

- **HistoryManager** (`history/history.go`)
  - 役割: ユーザーごとのチャット履歴を管理する。
  - メソッド:
    - `Add(userID, message, response)`: 履歴を追加する。
    - `Get(userID)`: 履歴を取得する。
    - `Clear(userID)`: 履歴をクリアする。

- **設定ローダー (config, loader)**
  - `config.LoadConfig()`: 環境変数(`.env`)を読み込む。
  - `loader.LoadModelConfig(filepath)`: `model.json` を読み込む。
  - `discord/custom_prompt.go` 内の関数群: `custom_model.json` の読み書きを行う。

### ドメインイベント

（変更なし）

## プログラムファイルの詳細

- **main.go:**
  - 役割: Discord Botの起動と設定を行う。
  - 処理:
    - `config.LoadConfig()`: 環境変数を読み込み、設定をロードする。
    - `discord.StartBot(cfg)`: Discord Botを起動する。

- **config/config.go:**
  - 役割: 環境変数の読み込みと管理を行う。
  - 処理:
    - `LoadConfig()`: `.env`ファイルを読み込み、環境変数を設定する。エラー発生時は `log.Fatalf` せずにエラーを返す。

- **chat/chat.go:**
  - 役割: LLM (Gemini, Ollama) との通信処理を行う。
  - 処理:
    - `NewChat(token, model, defaultPrompt, modelCfg, historyMgr)`: LLMクライアントと `HistoryManager` を受け取り、`ChatService` を初期化する。
    - `GetResponse(userID, username, message, timestamp, prompt)`: `HistoryManager` から履歴を取得し、LLMを呼び出し、応答を取得し、履歴を `HistoryManager` に追加する。`/reset` はここで処理せず、`discord/reset_command.go` で処理される。
    - `getOllamaResponse(fullInput string)`: Ollama APIを呼び出し、応答を取得する。ストリーミングレスポンスの解析は `parseOllamaStreamResponse` に分離。
    - `parseOllamaStreamResponse(reader *bufio.Reader)`: Ollamaのストリーミングレスポンスを解析する。
    - `Close()`: LLMクライアントを閉じる。

- **history/history.go:**
  - 役割: チャット履歴の管理ロジックを提供する。
  - 処理:
    - `HistoryManager` インターフェースを定義。
    - `InMemoryHistoryManager` 構造体と、そのメソッド (`Add`, `Get`, `Clear`) を実装。メモリ上で履歴を保持し、最大サイズを超えたら古いものから削除する。Mutexによる排他制御を行う。

- **discord/discord.go:**
  - 役割: Discord APIとのインターフェースを提供し、Botの起動、コマンド登録を行う。
  - 処理:
    - `StartBot(cfg)`: Discord Botを起動し、`setupHandlers` を呼び出してハンドラとサービスを初期化する。`model.json` の読み込みや `ChatService` の初期化は `setupHandlers` に移譲。

- **loader/model.go:**
  - 役割: `model.json` の読み込み処理と構造体定義を行う。
  - 処理:
    - `ModelConfig` 構造体などを定義。
    - `LoadModelConfig(filepath)`: `model.json`ファイルを読み込み、`ModelConfig`構造体にマッピングする。
    - `GetPromptByUser(username)`: `ModelConfig` からユーザー固有またはデフォルトのプロンプトを取得する。

- **json/custom_model.json:**
  - 役割: ユーザーがプロンプトをカスタマイズするための設定ファイル。`discord/custom_prompt.go` によって管理される。
  - 処理:
    - ユーザーは、`/edit` コマンドを通じてこのファイルの内容を変更できる。

- **discord/edit_command.go:**
  - 役割: `/edit`コマンドの処理を行う。
  - 処理:
    - ユーザーから`/edit`コマンドを受け取り、`discord/custom_prompt.go` の `SetCustomPromptForUser` または `DeleteCustomPromptForUser` を呼び出してカスタムテンプレートを更新または削除する。
    - 更新または削除の結果をEmbedで表示する。エラーハンドリングは `sendErrorResponse` を使用。

- **discord/handler.go:**
  - 役割: Discordのイベントハンドリングとコマンドディスパッチ、サービスの初期化を行う。
  - 処理:
    - `setupHandlers(s, geminiAPIKey)`: Bot起動時に呼び出され、ログ設定、`model.json` の読み込み、`HistoryManager` と `ChatService` の初期化、コマンドハンドラの登録を行う。エラー発生時は `log.Fatalf` せずにエラーを返す。
    - `interactionCreate` ハンドラ: 受け取ったインタラクションがアプリケーションコマンドの場合、コマンド名に応じて各コマンドハンドラ関数を呼び出す。`chatSvc`, `modelCfg`, `historyMgr` などの必要な依存関係を渡す。
    - `onReady`: Bot準備完了時のログ出力。

- **discord/chat_command.go:**
  - 役割: `/chat` コマンドの処理を行う。
  - 処理:
    - `chatCommandHandler(s, i, chatSvc, modelCfg)`: `discord/custom_prompt.go` の `GetCustomPromptForUser` でカスタムプロンプトを取得し、`chatSvc.GetResponse` を呼び出してLLMからの応答を取得し、結果をEmbedで表示する。エラーハンドリングは `sendErrorResponse` を使用。

- **discord/reset_command.go:**
  - 役割: `/reset` コマンドの処理を行う。
  - 処理:
    - `resetCommandHandler(s, i, historyMgr)`: 引数で受け取った `HistoryManager` の `Clear` メソッドを呼び出して履歴を削除し、結果をEmbedで表示する。

- **discord/about_command.go:**
  - 役割: `/about` コマンドの処理を行う。
  - 処理:
    - `aboutCommandHandler(s, i, modelCfg)`: 引数で受け取った `modelCfg` を元にBotの説明Embedを作成して表示する。エラーハンドリングは `sendErrorResponse` を使用。

- **discord/utils.go:** (新規)
  - 役割: Discord関連の共通ユーティリティ関数を提供する。
  - 処理:
    - `sendErrorResponse`: 標準的なエラー応答Embedを送信する。
    - `sendEphemeralErrorResponse`: 本人にのみ見えるエラー応答を送信する。

- **discord/embeds.go:** (新規)
  - 役割: Discord Embed作成に関するヘルパー関数を提供する。
  - 処理:
    - `splitToEmbedFields`: 長いテキストをEmbedフィールドの最大文字数で分割する。

- **discord/types.go:** (新規)
  - 役割: `discord` パッケージ内で共通して使用される型定義を行う。
  - 処理:
    - `CustomPromptConfig`: `custom_model.json` の構造体定義。

- **discord/custom_prompt.go:** (新規)
  - 役割: `json/custom_model.json` の読み書きロジックをカプセル化する。
  - 処理:
    - `GetCustomPromptForUser`, `SetCustomPromptForUser`, `DeleteCustomPromptForUser`: ユーザー固有のプロンプトを取得、設定、削除する。
    - `loadCustomPrompts`, `saveCustomPrompts`: ファイルの読み書きとMutexによる排他制御を行う。

## 今後の展望
- 検索機能を持たせる。GeminiのグラウンディングAPI を使って，ユーザーからの質問を受け付けて、検索結果とGeminiまたはOllamaの応答を組み合わせて、ユーザーに回答する。
