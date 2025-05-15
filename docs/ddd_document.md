# ドメイン駆動設計 ドキュメント

## プロジェクトの目的と概要

このプロジェクトは、Discord上でLLMのGeminiとOllamaとチャットできるDiscord Bot「ぺちこ」を開発しています。
ユーザーはDiscordのインターフェースを通じてGeminiまたはOllamaと対話できます。

## 主要な機能

- LLMとのチャット機能: ユーザーはテキストメッセージを送信し、GeminiまたはOllamaからの応答を受信できます。スレッド内での会話の場合、スレッド単位で履歴が管理されます。
- チャット履歴のリセット機能: ユーザーは`/reset`コマンドを使用して、コマンドを実行したスレッド（またはチャンネル）のチャット履歴をクリアできます。
- BOTの説明表示機能: `/about`コマンドで、json/model.json に定義されたBOTの説明を表示します。
- プロンプト編集機能: ユーザーは`/edit`コマンドを使用してプロンプトを編集できます。
  - `/edit`コマンドで"delete"と送信された場合、custom_model.json から該当ユーザーの行を削除します。
- 天気情報提供機能: ユーザーがメッセージに「天気」と地名を含むと、`zu2l` API (`GetWeatherPoint`, `GetWeatherStatus`) を呼び出して天気情報を返します。
- 頭痛情報提供機能: ユーザーがメッセージに「頭痛」または「ずつう」と地名を含むと、`zu2l` API (`GetWeatherPoint`, `GetPainStatus`) を呼び出して頭痛情報を返します。
- Otenki ASP情報提供機能: ユーザーがメッセージに「asp情報」と地点コードを含むと、`zu2l` API (`GetOtenkiASP`) を呼び出してOtenki ASP情報を返します。
- 地点検索機能: ユーザーがメッセージに「地点検索」とキーワードを含むと、`zu2l` API (`GetWeatherPoint`) を呼び出して地点情報を返します。

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
  - 履歴管理: Discordのスレッド（またはチャンネル）単位およびユーザー単位でのチャット履歴の永続化と取得 (DuckDBを使用)
- 境界づけられたコンテキスト:
  - Discord Bot: Discord APIとのインターフェース (`discord` パッケージ)。スレッドIDの取得とハンドラへの引き渡しを含む。
  - LLMクライアント: LLM API (Gemini, Ollama) とのインターフェース (`chat` パッケージ)。スレッドIDを考慮した履歴の取得と保存を行う。
  - 設定ローダー: 設定ファイルの読み込み (`loader`, `config` パッケージ)
  - 履歴マネージャー: チャット履歴の管理 (`history` パッケージ)。DuckDBによる永続化、スレッドIDとユーザーIDに基づいた履歴操作。
- 関係:
  - Discord BotはLLMクライアントを利用してチャット機能を提供します。
  - Discord Botはユーザーからのコマンドを受け付け、適切な処理を呼び出します。
  - LLMクライアントは履歴マネージャーを利用して対話履歴を取得・保存します。
  - Discord Botは設定ローダーから設定を読み込みます。
  - Discord Botは履歴マネージャーを利用して履歴をリセットします。

## ドメインモデル

### エンティティ

- ユーザー (User)
  - ID: DiscordのユーザーID (userID)。Discord APIから取得。
  - ユーザー名: Discordのユーザー名 (username)。Discord APIから取得。
  - 役割: (将来の拡張を見据えて) ユーザーの役割を定義。

- メッセージ (Message)
  - 内容: ユーザーまたはBotからのメッセージ (content)。
  - タイムスタンプ: メッセージの送信時刻 (timestamp)。
  - ユーザーID: メッセージを送信したユーザーのID (userID)。
  - 送信者: メッセージの送信者 (ユーザーまたはBot)。

### 値オブジェクト

- タイムスタンプ (Timestamp)
  - メッセージの送信時刻。

- モデル設定 (ModelConfig) (`loader/model.go`)
  - モデル名: 使用するLLMのモデル名 (model_name)。
  - セカンダリモデル名 (SecondaryModelName): Gemini API のクォータ超過 (429エラー) 時にフォールバックとして試行するモデル名 (任意)。
  - アイコン: Botのアイコン (icon)。
  - プロンプト: LLMへの指示文 (prompts)。ユーザーごと、またはデフォルトの指示文が設定可能。
  - BOTの説明: BOTの説明 (about)。
  - チャット履歴の最大サイズ (max_history_size): 保持するチャット履歴の最大件数。
  - Ollama設定 (OllamaConfig): Ollama APIに関する設定。

- カスタムプロンプト設定 (CustomPromptConfig) (`discord/types.go`)
  - ユーザープロンプト (Prompts): ユーザー名をキーとしたカスタムプロンプトのマップ。

### ドメインサービス / アプリケーションサービス / インフラストラクチャサービス

- ChatService (`chat/chat.go`, `chat/prompt.go`, `chat/utils.go`, `chat/ollama.go`)
  - 役割: LLM (Gemini, Ollama) とのやり取り、および Gemini の Function Calling 機能のディスパッチを担当する。天気関連の Function Calling 処理は `WeatherService` に委譲する。
  - 処理:
    - `NewChat(token, historyMgr)` (`chat/chat.go`): Geminiクライアント、`HistoryManager`、`WeatherService` を初期化する。初期の Gemini モデル名は `model.json` から読み込むが、設定自体は保持しない。`WeatherService` から取得した Function Declaration を含む `Tool` を定義し、初期 Gemini モデルに設定する。
    - `GetResponse(userID, threadID, username, message, timestamp, prompt)` (`chat/chat.go`):
      1. `model.json` を読み込む。
      2. `buildFullInput` (`chat/prompt.go`) を呼び出して、プロンプト、現在日時情報、ツール指示、履歴、ユーザーメッセージを結合した入力文字列を生成する。
      3. `model.json` の `ollama.enabled` が `true` の場合、`getOllamaResponse` を呼び出して Ollama にリクエストを送信する。
      4. `ollama.enabled` が `false` の場合、`model.json` の `model_name` を使用して Gemini にリクエストを送信する (`genaiModel.GenerateContent`)。
      5. Gemini API から 429 (Quota Exceeded) エラーが返された場合:
         - `model.json` の `secondary_model_name` が設定されていれば、そのモデルで Gemini API に再リクエストする。
         - 再リクエストも失敗した場合、または `secondary_model_name` が未設定の場合で、かつ `model.json` の `ollama.enabled` が `true` であれば、`getOllamaResponse` を呼び出して Ollama にフォールバックする。
         - 上記いずれのフォールバックも実行できない場合は、エラーを返す。
      6. Gemini からの応答 (`resp.Candidates[0].Content.Parts`) をループで確認する。
      7. 応答パーツの中に `genai.Text` 型が見つかった場合、その内容を `llmIntroText` バッファに追記する。
      5. 応答パーツの中に `genai.FunctionCall` 型が見つかった場合:
         - `weatherService.HandleFunctionCall` を呼び出して、天気関連の Function Call 処理を委譲する。
         - `toolResult` に結果を格納し、`functionCallProcessed` フラグを立てる。（エラーハンドリングも含む）
      8. ループ終了後、`functionCallProcessed` が `true` の場合:
         - `llmIntroText` (導入文) と `toolResult` (ツール実行結果) を結合する。
         - 結合した応答を `historyMgr.Add` で履歴に追加する。
         - 結合した応答を `return` する (source は "zutool")。
      9. `functionCallProcessed` が `false` の場合 (通常のテキスト応答):
         - `llmIntroText` を取得する (空の場合は `getResponseText` (`chat/utils.go`) でフォールバック)。
         - 応答を `historyMgr.Add` で履歴に追加する。
         - テキスト応答を `return` する (source は実際に使用されたモデル名)。
    - `Close()` (`chat/chat.go`): Geminiクライアントを閉じる。
    - `buildFullInput` (`chat/prompt.go`): LLMへの入力文字列を構築する。
    - `getResponseText` (`chat/utils.go`): 応答テキスト抽出ヘルパー。
    - (Ollama関連の `getOllamaResponse`, `parseOllamaStreamResponse` は `chat/ollama.go` に分離されている。)

- WeatherService (`chat/weather.go`) (新規)
  - 役割: 天気関連の Function Calling 処理と `zu2l` API との連携を担当する。
  - 処理:
    - `NewWeatherService`: `zutoolapi.Client` を初期化する。
    - `GetFunctionDeclarations`: 天気関連の Function Declaration (`getWeather`, `getPainStatus`, `searchWeatherPoint`, `getOtenkiAspInfo`) のリストを返す。
    - `HandleFunctionCall`: 受け取った `genai.FunctionCall` の名前 (`fn.Name`) に基づいて、対応する内部ハンドラ (`handleGetWeather` など) を呼び出す。
    - `handleGetWeather`, `handleGetPainStatus`, `handleSearchWeatherPoint`, `handleGetOtenkiAspInfo`: 各 Function Call の具体的な処理。`zutoolapi.Client` を使用して `zu2l` API を呼び出し、結果を整形して文字列として返す。地点情報の取得や天気コードの絵文字変換はヘルパー関数を利用する。
    - `getLocationInfo` (ヘルパー関数): `GetWeatherPoint` API を呼び出し、地点コードと地点名を取得する共通処理。
    - `getWeatherEmoji` (ヘルパー関数): 天気コード（数値または文字列）を受け取り、`weatherEmojiMap` を参照して対応する絵文字を返す共通処理。
    - `weatherEmojiMap`: 天気コードに対応する絵文字を定義する。

- HistoryManager (`history/history.go`, `history/duckdb_manager.go`)
  - 役割: Discordのスレッド（またはチャンネル）単位およびユーザー単位でのチャット履歴を管理する。DuckDBデータベースを使用して履歴を永続化する。
  - インターフェース (`history.HistoryManager`):
    - `Add(userID, threadID, message, response)`: 指定されたユーザーとスレッドの履歴にメッセージと応答を追加する。履歴は全て保存され、古いものは削除されない。
    - `Get(userID, threadID)`: 指定されたユーザーとスレッドの履歴を取得する。データベースからは全ての履歴を取得し、最新20ペアを返す。
    - `Clear(userID, threadID)`: 指定されたユーザーとスレッドの履歴をクリアする。
    - `ClearAllByThreadID(threadID)`: 指定されたスレッドの全ユーザーの履歴をクリアする。
    - `Close()`: データベース接続などのリソースを解放する。
  - 実装 (`history.DuckDBHistoryManager`):
    - DuckDBデータベースへの接続、テーブル作成、CRUD操作を実装。
    - `thread_histories` テーブル (thread_id, user_id, history_json, last_updated_at) を使用。
  - 実装 (`history.InMemoryHistoryManager`):
    - 従来のユーザーID単位のメモリ上履歴管理。スレッドIDはダミー引数として受け取るが使用しない。テスト用などに利用。

- 設定ローダー (config, loader)
  - 役割: 環境変数の読み込みと管理を行う。
  - 処理:
    - `config.LoadConfig()`: 環境変数(`.env`)を読み込む。
    - `loader.LoadModelConfig(filepath)`: `model.json` を読み込む。
    - `discord/custom_prompt.go` 内の関数群: `custom_model.json` の読み書きを行う。

### ドメインイベント

（変更なし）

## プログラムファイルの詳細

- main.go:
  - 役割: Discord Botの起動と設定を行う。
  - 処理:
    - `config.LoadConfig()`: 環境変数を読み込み、設定をロードする。
    - `discord.StartBot(cfg)`: Discord Botを起動する。

- config/config.go:
  - 役割: 環境変数の読み込みと管理を行う。
  - 処理:
    - `LoadConfig()`: `.env`ファイルを読み込み、環境変数を設定する。エラー発生時は `log.Fatalf` せずにエラーを返す。

- chat/chat.go:
  - 役割: `ChatService` の主要な実装。LLM (Gemini, Ollama) API との通信、Gemini の Function Calling のディスパッチ（`WeatherService` への委譲を含む）、応答生成のコアロジック、エラー時のフォールバック処理を担当。
  - 処理:
    - `NewChat`: サービスと依存関係 (`WeatherService` を含む) を初期化。`model.json` は初期モデル名取得のために一時的に読み込むが、設定は保持しない。
    - `GetResponse(userID, threadID, username, message, timestamp, prompt)`: コマンド実行ごとに `model.json` を読み込む。ユーザーからのメッセージ、スレッドID、タイムスタンプを受け取り、`model.json` の設定に基づいて使用するLLMを決定。`historyMgr` を使用してスレッドIDに基づいた履歴を取得・保存する。Gemini API で 429 エラーが発生した場合のフォールバック処理などを行う。
    - `Close`: リソースを解放する。

- chat/prompt.go:
  - 役割: LLM に送信するプロンプト（入力文字列）の構築ロジックを担当。
  - 処理:
    - `buildFullInput(systemPrompt, userMessage, historyMgr, userID, threadID, timestamp)`: システムプロンプト、現在日時情報、ツール指示、会話履歴（スレッドIDとユーザーIDで取得）、ユーザーメッセージを結合する。

- chat/weather.go: (新規)
  - 役割: `WeatherService` の実装。天気関連の Function Calling 処理と `zu2l` API との連携を担当。
  - 処理:
    - `WeatherService` インターフェースと `weatherServiceImpl` 構造体を定義。
    - `NewWeatherService`: サービスの初期化。
    - `GetFunctionDeclarations`: 天気関連ツールの定義を返す。
    - `HandleFunctionCall` および `handle*` メソッド群: 各 Function Call の具体的な処理と `zu2l` API 呼び出し。地点情報取得や絵文字変換はヘルパー関数を利用。
    - `getLocationInfo`, `getWeatherEmoji` (ヘルパー関数): コードの共通化と可読性向上のための内部ヘルパー。
    - `weatherEmojiMap`: 天気コードと絵文字のマッピング。

- chat/utils.go:
  - 役割: `chat` パッケージ内で共通して使用されるヘルパー関数を提供する。
  - 処理:
    - `getResponseText`: Gemini の応答からテキスト部分を抽出する。

- chat/ollama.go:
  - 役割: Ollama LLM との連携に関するロジックを担当。
  - 処理:
    - `getOllamaResponse(userID, threadID, message, fullInput, ollamaCfg)`: 引数で受け取った `OllamaConfig` を使用して Ollama API にリクエストを送信し、応答を取得してスレッドIDとユーザーIDに基づいて履歴に追加する。
    - `parseOllamaStreamResponse`: Ollama からのストリーミング応答を解析する。

- history/history.go:
  - 役割: チャット履歴管理のインターフェース (`HistoryManager`) とインメモリ実装 (`InMemoryHistoryManager`) を提供する。
  - 処理:
    - `HistoryManager` インターフェース: `Add`, `Get`, `Clear`, `ClearAllByThreadID`, `Close` メソッドを定義。スレッドIDを引数に取る。
    - `InMemoryHistoryManager`: 従来のユーザーID単位のメモリ上履歴管理。スレッドIDはダミー引数として受け取る。`Close` は何もしない。

- history/duckdb_manager.go: (新規)
  - 役割: `HistoryManager` インターフェースのDuckDB実装 (`DuckDBHistoryManager`) を提供する。
  - 処理:
    - `NewDuckDBHistoryManager`: DuckDBデータベースへの接続とテーブル (`thread_histories`) の初期化を行う。
    - `Add`, `Get`, `Clear`, `ClearAllByThreadID`: スレッドIDとユーザーIDに基づいてDuckDBデータベース内の履歴を操作する。履歴はJSON形式で保存。
    - `Close`: DuckDBデータベース接続を閉じる。

- history/sqlite_manager.go: (削除)
  - SQLiteによる履歴管理は削除された。

- discord/discord.go:
  - 役割: Discord APIとのインターフェースを提供し、Botの起動、コマンド登録、リソース管理を行う。
  - 処理:
    - `StartBot(cfg)`: Discord Botを起動し、`setupHandlers` を呼び出してハンドラとサービスを初期化する。起動シーケンス中の各ステップ（セッション作成、セッションオープン、コマンド登録、ハンドラ設定など）でエラーハンドリングを強化し、詳細なログを出力するように修正。`HistoryManager` および Discord セッション (`session`) の `Close` メソッドが、関数の正常終了時および異常終了時に `defer` を用いて確実に呼び出されるように修正し、リソースリークを防ぐ。

- loader/model.go:
  - 役割: `model.json` の読み込み処理と構造体定義を行う。
  - 処理:
    - `ModelConfig` 構造体などを定義。`SecondaryModelName` フィールドを追加。
    - `LoadModelConfig(filepath)`: `model.json`ファイルを読み込み、`ModelConfig`構造体にマッピングする。
    - `GetPromptByUser(username)`: `ModelConfig` からユーザー固有またはデフォルトのプロンプトを取得する。

- json/custom_model.json:
  - 役割: ユーザーがプロンプトをカスタマイズするための設定ファイル。`discord/custom_prompt.go` によって管理される。
  - 処理:
    - ユーザーは、`/edit` コマンドを通じてこのファイルの内容を変更できる。

- discord/edit_command.go:
  - 役割: `/edit`コマンドの処理を行う。
  - 処理:
    - ユーザーから`/edit`コマンドを受け取り、`discord/custom_prompt.go` の `SetCustomPromptForUser` または `DeleteCustomPromptForUser` を呼び出してカスタムテンプレートを更新または削除する。
    - 更新または削除の結果をEmbedで表示する。エラーハンドリングは `sendErrorResponse` を使用。

- discord/handler.go:
  - 役割: Discordのイベントハンドリングとコマンドディスパッチ、サービスの初期化を行う。
  - 処理:
    - `setupHandlers(s, geminiAPIKey)`: Bot起動時に呼び出され、ログ設定、`DuckDBHistoryManager` と `ChatService` の初期化、コマンドハンドラの登録を行う。初期化された `HistoryManager` を返す。
    - `interactionCreate` ハンドラ: 受け取ったインタラクションからスレッドID（スレッドでない場合はチャンネルID）を取得し、各コマンドハンドラに渡す。
    - `onReady`: Bot準備完了時のログ出力。

- discord/chat_command.go:
  - 役割: `/chat` コマンドの処理を行う。
  - 処理:
    - `chatCommandHandler(s, i, chatSvc, threadID)`: コマンド実行時に `model.json` を読み込む。カスタムプロンプトを取得し、受け取った `threadID` と共に `chatSvc.GetResponse` を呼び出してLLMからの応答を取得し、結果をEmbedで表示する。

- discord/reset_command.go:
  - 役割: `/reset` コマンドの処理を行う。
  - 処理:
    - `resetCommandHandler(s, i, historyMgr, threadID)`: 受け取った `threadID` を使用して `historyMgr.ClearAllByThreadID` を呼び出し、該当スレッド（またはチャンネル）の全ユーザーの履歴を削除し、結果をEmbedで表示する。

- discord/about_command.go:
  - 役割: `/about` コマンドの処理を行う。
  - 処理:
    - `aboutCommandHandler(s, i)`: コマンド実行時に `model.json` を読み込む。 読み込んだ `modelCfg` を元にBotの説明Embedを作成して表示する。エラーハンドリングは `sendErrorResponse` を使用。

- discord/utils.go: (新規)
  - 役割: Discord関連の共通ユーティリティ関数を提供する。
  - 処理:
    - `sendErrorResponse`: 標準的なエラー応答Embedを送信する。
    - `sendEphemeralErrorResponse`: 本人にのみ見えるエラー応答を送信する。

- discord/embeds.go:
  - 役割: Discord Embed作成に関するヘルパー関数を提供する。
  - 処理:
    - `splitToEmbedFields`: LLMからの応答テキストを受け取り、Discord Embedフィールドの文字数制限(1024文字)に合わせて処理する。1024文字を超える場合は、文字化けしないようにルーン単位で分割し、複数のフィールドに分割して表示する。 フィールド名は空にする。

- discord/types.go: (新規)
  - 役割: `discord` パッケージ内で共通して使用される型定義を行う。
  - 処理:
    - `CustomPromptConfig`: `custom_model.json` の構造体定義。

- discord/custom_prompt.go: (新規)
  - 役割: `json/custom_model.json` の読み書きロジックをカプセル化する。
  - 処理:
    - `GetCustomPromptForUser`, `SetCustomPromptForUser`, `DeleteCustomPromptForUser`: ユーザー固有のプロンプトを取得、設定、削除する。
    - `loadCustomPrompts`, `saveCustomPrompts`: ファイルの読み書きとMutexによる排他制御を行う。

## 今後の展望
- 検索機能を持たせる。GeminiのグラウンディングAPI を使って，ユーザーからの質問を受け付けて、検索結果とGeminiまたはOllamaの応答を組み合わせて、ユーザーに回答する。
