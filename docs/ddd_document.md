# ドメイン駆動設計 ドキュメント

## プロジェクトの目的と概要

このプロジェクトは、Discord上でLLMのGeminiとOllamaとチャットできるDiscord Bot「ぺちこ」を開発しています。
ユーザーはDiscordのインターフェースを通じてGeminiまたはOllamaと対話できます。

## 主要な機能

- LLMとのチャット機能: ユーザーはテキストメッセージを送信し、GeminiまたはOllamaからの応答を受信できます。スレッド内での会話の場合、スレッド単位で履歴が管理されます。
- チャット履歴のリセット機能: ユーザーは`/reset`コマンドを使用して、コマンドを実行したスレッド（またはチャンネル）のチャット履歴をクリアできます。
- BOTの説明表示機能: `/about`コマンドで、`json/model.json` に定義されたBOTの説明を表示します。
- プロンプト編集機能: ユーザーは`/edit`コマンドを使用してプロンプトを編集できます。
  - `/edit`コマンドで"delete"と送信された場合、`json/custom_model.json` から該当ユーザーの行を削除します。
- DM応答機能: BotへのDM送信に対しても応答します。
- 天気情報提供機能: LLMがFunction Calling (`getWeather`) を利用して、ユーザーがメッセージに「天気」と地名を含む場合に `zu2l` API (`GetWeatherPoint`, `GetWeatherStatus`) を呼び出して天気情報を返します。
- 頭痛情報提供機能: LLMがFunction Calling (`getPainStatus`) を利用して、ユーザーがメッセージに「頭痛」または「ずつう」と地名を含む場合に `zu2l` API (`GetWeatherPoint`, `GetPainStatus`) を呼び出して頭痛情報を返します。
- Otenki ASP情報提供機能: LLMがFunction Calling (`getOtenkiAspInfo`) を利用して、ユーザーがメッセージに「asp情報」と地点コードを含む場合に `zu2l` API (`GetOtenkiASP`) を呼び出してOtenki ASP情報を返します。
- 地点検索機能: LLMがFunction Calling (`searchWeatherPoint`) を利用して、ユーザーがメッセージに「地点検索」とキーワードを含む場合に `zu2l` API (`GetWeatherPoint`) を呼び出して地点情報を返します。
- URL内容理解機能: ユーザーが会話中にURLを提示すると、LLMがFunction Calling (`get_url_content`) を利用して `URLReaderService` を呼び出し、URLの主要なテキストコンテンツを取得し、内容を理解した上で応答します。
- メッセージ監査ログ機能: Discordでのメッセージ作成、更新、削除イベントをログファイル (`log/audit.log`) に記録します。

## ドメインに関する用語（ユビキタス言語）

- Bot: Discord Bot
- Gemini: GoogleのLLM
- Ollama: ローカルで動作するLLM
- LLM: GeminiやOllamaなどの大規模言語モデル
- チャット履歴 (History): スレッドIDとユーザーIDごとのLLMとの対話の記録。DuckDBに永続化される。
- コマンド: Botへの指示 (例: `/chat`, `/reset`, `/about`, `/edit`)
- プロンプト: GeminiまたはOllamaへの指示文。ユーザーごと、またはデフォルトの指示文が設定可能。
- ぺちこ: このBotの名前
- 設定 (Config): `.env` ファイル、`json/model.json`、`json/custom_model.json` から読み込まれるBotの動作に必要な設定情報。(`config/config.go` で一元管理)
- モデル設定 (ModelConfig): `json/model.json` から読み込まれるBotのLLMモデルや表示に関する基本設定。(`loader/model.go` で定義、`config/config.go` で読み込み)
- カスタムプロンプト設定 (CustomPromptConfig): `json/custom_model.json` から読み書きされるユーザー固有のプロンプト設定。(`config/config.go` で定義・読み込み、`discord/custom_prompt.go` で書き込み処理)
- スレッドID (ThreadID): Discordのスレッドまたはチャンネルの一意な識別子。履歴管理やコマンド処理の単位となる。
- 監査ログ (AuditLog): メッセージの作成、更新、削除などのイベントを記録するログ。

## コンテキストマップ

- サブドメイン:
  - チャット: ユーザーとLLMとの対話の管理
  - コマンド処理: ユーザーからのコマンドの解析と実行
  - 設定管理: Botの設定の読み込みと管理 (`.env`, `json/model.json`, `json/custom_model.json`)
  - 履歴管理: Discordのスレッド（またはチャンネル）単位およびユーザー単位でのチャット履歴の永続化と取得 (DuckDBを使用)
  - 監査ログ管理: Discordメッセージイベントの記録 (`log/audit.log`)
- 境界づけられたコンテキスト:
  - Discord Bot: Discord APIとのインターフェース (`discord` パッケージ)。スレッドIDの取得、イベントハンドリング、コマンドディスパッチを行う。
  - LLMクライアント: LLM API (Gemini, Ollama) とのインターフェース (`chat` パッケージ)。Function Callingの処理、履歴の取得と保存を行う。
  - 設定ローダー: 設定ファイルの読み込み (`config`, `loader` パッケージ)
  - 履歴マネージャー: チャット履歴の管理 (`history` パッケージ)。DuckDBによる永続化、スレッドIDとユーザーIDに基づいた履歴操作。
  - 監査ロガー: メッセージイベントのロギング (`history` パッケージ)。
- 関係:
  - Discord BotはLLMクライアントを利用してチャット機能を提供します。
  - Discord Botはユーザーからのコマンドを受け付け、適切な処理を呼び出します。
  - LLMクライアントは履歴マネージャーを利用して対話履歴を取得・保存します。
  - Discord Botは設定ローダーから設定を読み込みます。
  - Discord Botは履歴マネージャーを利用して履歴をリセットします。
  - Discord Botは監査ロガーを利用してメッセージイベントを記録します。

## ドメインモデル

### エンティティ

- ユーザー (User)
  - ID: DiscordのユーザーID (userID)。Discord APIから取得。
  - ユーザー名: Discordのユーザー名 (username)。Discord APIから取得。

- チャット履歴エントリ (HistoryMessage) (`history/history.go`)
  - ロール (Role): メッセージの送信者 ("user" または "model")。
  - 内容 (Content): メッセージの内容。

### 値オブジェクト

- タイムスタンプ (Timestamp)
  - メッセージの送信時刻やイベント発生時刻。

- 設定 (Config) (`config/config.go`)
  - DiscordBotToken: Discord Botのトークン。
  - GeminiAPIKey: Gemini APIのキー。
  - Model: `loader.ModelConfig` へのポインタ。
  - CustomModel: `config.CustomPromptConfig` へのポインタ。

- モデル設定 (ModelConfig) (`loader/model.go`)
  - Name: Botの名前。
  - モデル名 (ModelName): 使用するLLMのモデル名。
  - セカンダリモデル名 (SecondaryModelName): Gemini API のクォータ超過 (429エラー) 時にフォールバックとして試行するモデル名 (任意)。
  - アイコン (Icon): BotのアイコンURL。
  - プロンプト (Prompts): LLMへの指示文。ユーザー名をキーとしたマップ。
  - BOTの説明 (About): `/about` コマンドで表示されるBotの説明 (`loader.About` 型)。
  - チャット履歴の最大サイズ (MaxHistorySize): `InMemoryHistoryManager` で保持するチャット履歴の最大ペア数。
  - Ollama設定 (OllamaConfig): Ollama APIに関する設定 (`loader.OllamaConfig` 型)。

- カスタムプロンプト設定 (CustomPromptConfig) (`config/config.go`)
  - ユーザープロンプト (Prompts): ユーザー名をキーとしたカスタムプロンプトのマップ。

### ドメインサービス / アプリケーションサービス / インフラストラクチャサービス

- ChatService (`chat/chat.go`, `chat/prompt.go`, `chat/utils.go`, `chat/ollama.go`, `chat/weather.go`)
  - 役割: LLM (Gemini, Ollama) とのやり取り、および Gemini の Function Calling 機能のディスパッチを担当する。天気関連の Function Calling 処理は `WeatherService` に、URL内容取得関連の Function Calling 処理は `URLReaderService` に委譲する。
  - 処理:
    - `NewChat(cfg *config.Config, historyMgr history.HistoryManager)` (`chat/chat.go`): Geminiクライアント、`HistoryManager`、`WeatherService`、`URLReaderService` を初期化する。`cfg.Model` から `ModelConfig` を取得し保持する。`WeatherService` および `URLReaderService` から取得した Function Declaration を含む `Tool` を定義し、初期 Gemini モデルに設定する。
    - `GetResponse(userID, threadID, username, message, timestamp, prompt)` (`chat/chat.go`):
      1. 保持している `ModelConfig` を使用する。
      2. `buildFullInput` (`chat/prompt.go`) を呼び出して、プロンプト、現在日時情報、ツール指示（`get_url_content`、天気関連を含む）、履歴、ユーザーメッセージを結合した入力文字列を生成する。
      3. `ModelConfig.Ollama.Enabled` が `true` の場合、`getOllamaResponse` (`chat/ollama.go`) を呼び出して Ollama にリクエストを送信する。
      4. `ModelConfig.Ollama.Enabled` が `false` の場合、`ModelConfig.ModelName` を使用して Gemini にリクエストを送信する (`genaiModel.GenerateContent`)。
      5. Gemini API から 429 (Quota Exceeded) エラーが返された場合:
         - `ModelConfig.SecondaryModelName` が設定されていれば、そのモデルで Gemini API に再リクエストする。
         - 再リクエストも失敗した場合、または `SecondaryModelName` が未設定の場合で、かつ `ModelConfig.Ollama.Enabled` が `true` であれば、`getOllamaResponse` を呼び出して Ollama にフォールバックする。
         - 上記いずれのフォールバックも実行できない場合は、エラーを返す。
      6. Gemini からの応答 (`resp.Candidates[0].Content.Parts`) をループで確認する。
      7. 応答パーツの中に `genai.Text` 型が見つかった場合、その内容を `llmIntroText` バッファに追記する。
      8. 応答パーツの中に `genai.FunctionCall` 型が見つかった場合:
         - `fn.Name` が `get_url_content` の場合、`urlReaderService.GetURLContentAsText` を呼び出してURLの内容を取得する。URLは `fn.Args["url"]` から取得する。
         - それ以外の場合（天気関連など）、`weatherService.HandleFunctionCall` を呼び出して処理を委譲する。
         - `toolResult` に結果を格納し、`functionCallProcessed` フラグを立てる。（エラーハンドリングも含む）
      9. ループ終了後、`functionCallProcessed` が `true` の場合:
         - `toolResult` を含む `FunctionResponse` を作成し、再度 `genaiModel.GenerateContent` を呼び出してLLMに最終的な応答を生成させる。
         - 生成された応答を `historyMgr.Add` で履歴に追加する。
         - 生成された応答を `return` する。
      10. `functionCallProcessed` が `false` の場合 (通常のテキスト応答):
          - `llmIntroText` を取得する (空の場合は `getResponseText` (`chat/utils.go`) でフォールバック)。
          - 応答を `historyMgr.Add` で履歴に追加する。
          - テキスト応答を `return` する (source は実際に使用されたモデル名)。
    - `Close()` (`chat/chat.go`): Geminiクライアントを閉じる。
    - `buildFullInput` (`chat/prompt.go`): LLMへの入力文字列を構築する。履歴のロール名 "model" を "assistant" に変換する。
    - `getResponseText` (`chat/utils.go`): 応答テキスト抽出ヘルパー。
    - `getOllamaResponse` (`chat/ollama.go`): Ollama API との通信処理。
    - `parseOllamaStreamResponse` (`chat/ollama.go`): Ollama のストリーミング応答の解析。

- WeatherService (`chat/weather.go`)
  - 役割: 天気関連の Function Calling 処理と `zu2l` API との連携を担当する。
  - 処理:
    - `NewWeatherService`: `zutoolapi.Client` を初期化する。
    - `GetFunctionDeclarations`: 天気関連の Function Declaration (`getWeather`, `getPainStatus`, `searchWeatherPoint`, `getOtenkiAspInfo`) のリストを返す。
    - `HandleFunctionCall`: 受け取った `genai.FunctionCall` の名前 (`fn.Name`) に基づいて、対応する内部ハンドラ (`handleGetWeather` など) を呼び出す。
    - `handleGetWeather`, `handleGetPainStatus`, `handleSearchWeatherPoint`, `handleGetOtenkiAspInfo`: 各 Function Call の具体的な処理。`zutoolapi.Client` を使用して `zu2l` API を呼び出し、結果を整形して文字列として返す。地点情報の取得や天気コードの絵文字変換はヘルパー関数を利用する。
    - `getLocationInfo` (ヘルパー関数): `GetWeatherPoint` API を呼び出し、地点コードと地点名を取得する共通処理。
    - `getWeatherEmoji` (ヘルパー関数): 天気コード（数値または文字列）を受け取り、100の位で丸めて `weatherEmojiMap` を参照して対応する絵文字を返す共通処理。
    - `weatherEmojiMap`: 天気コードに対応する絵文字を定義する。

- URLReaderService (`chat/url_reader_service.go`)
  - 役割: URLの内容取得とテキスト抽出、および `get_url_content` Function Declaration の提供を担当する。
  - 処理:
    - `NewURLReaderService`: サービスの初期化。
    - `GetURLContentAsText(url string) (string, error)`: 指定されたURLの内容を `http.Get` で取得し、`goquery` でHTMLをパースして主要なテキストを抽出する。抽出テキストは最大2000文字に制限される。
    - `GetURLReaderFunctionDeclaration() *genai.FunctionDeclaration`: `get_url_content` 関数の定義 (`FunctionDeclaration`) を返す。

- HistoryManager (`history/history.go`, `history/duckdb_manager.go`)
  - 役割: Discordのスレッド（またはチャンネル）単位およびユーザー単位でのチャット履歴を管理する。DuckDBデータベースを使用して履歴を永続化する。
  - インターフェース (`history.HistoryManager`):
    - `Add(userID, threadID, message, response) error`: 指定されたユーザーとスレッドの履歴にメッセージと応答を追加する。`DuckDBHistoryManager` では履歴は全て保存され、古いものは削除されない。
    - `Get(userID, threadID) ([]HistoryMessage, error)`: 指定されたユーザーとスレッドの履歴を取得する。`DuckDBHistoryManager` ではデータベースから全ての履歴を取得し、最新20ペアを返す。
    - `Clear(userID, threadID) error`: 指定されたユーザーとスレッドの履歴をクリアする。
    - `ClearAllByThreadID(threadID) error`: 指定されたスレッドの全ユーザーの履歴をクリアする。
    - `Close() error`: データベース接続などのリソースを解放する。
  - 実装 (`history.DuckDBHistoryManager`):
    - DuckDBデータベースへの接続、テーブル作成、CRUD操作を実装。
    - `thread_histories` テーブル (thread_id, user_id, history_json, last_updated_at) を使用。
  - 実装 (`history.InMemoryHistoryManager`):
    - ユーザーID単位のメモリ上履歴管理。スレッドIDはダミー引数として受け取るが使用しない。履歴は最大サイズを超えると古いものからペアで削除される。テスト用などに利用。

- 設定サービス (`config/config.go`, `loader/model.go`, `discord/custom_prompt.go`)
  - 役割: 設定ファイル (`.env`, `json/model.json`, `json/custom_model.json`) の読み込みと管理を行う。
  - 処理:
    - `config.LoadConfig()`: `.env` ファイル、`json/model.json`、`json/custom_model.json` を読み込み、`Config` 構造体を返す。(`config/config.go`)
    - `loader.LoadModelConfig(filepath)`: `json/model.json` を読み込む。(`loader/model.go`)
    - `discord/custom_prompt.go` 内の関数群: `json/custom_model.json` の書き込み処理（設定、削除）を行う。Mutexによる排他制御を含む。カスタムプロンプトの取得は `config.Config` 経由で行う。

- 監査ログサービス (`history/audit_log.go`)
  - 役割: Discordメッセージイベント（作成、更新、削除）をログファイル (`log/audit.log`) に記録する。
  - 処理:
    - `InitAuditLog`: 監査ログファイルを初期化する。
    - `LogMessageCreate`, `LogMessageUpdate`, `LogMessageDelete`: 各イベントをログに書き込む。
    - `CloseAuditLog`: 監査ログファイルを閉じる。

### ドメインイベント

ドメインイベントとは、ドメイン内で発生した過去の重要な出来事を表すオブジェクトです。例えば、「ユーザーがチャットを開始した」「プロンプトが更新された」などが考えられます。
これらのイベントは、システムの他の部分に影響を与えたり、特定のビジネスロジックをトリガーしたりする可能性があります。

現状のコードベースでは、このようなドメインイベントを明示的に定義し、発行・購読するような仕組みは見受けられません。そのため、「現状、明示的なドメインイベントは定義されていません。」と記載しています。

## プログラムファイルの詳細

- main.go:
  - 役割: Discord Botの起動と設定、監査ログの初期化、シグナルハンドリングを行う。
  - 処理:
    - `history.InitAuditLog()`: 監査ログを初期化する。
    - `config.LoadConfig()`: 環境変数、`json/model.json`、`json/custom_model.json` を読み込み、設定をロードする。
    - `discord.StartBot(cfg)`: Discord Botを起動する。
    - アプリケーション終了のためのシグナルハンドリングを行い、`history.CloseAuditLog()` を呼び出す。

- config/config.go:
  - 役割: `.env` ファイル、`json/model.json`、`json/custom_model.json` の読み込みと管理を一元的に行う。
  - 処理:
    - `LoadConfig()`: `.env`ファイルを読み込み環境変数を設定し、`loader.LoadModelConfig` を呼び出して `json/model.json` を、内部関数 `loadCustomPrompts` で `json/custom_model.json` を読み込み、`Config` 構造体を構築して返す。エラー発生時はエラーを返す。
    - `CustomPromptConfig` 構造体を定義。

- chat/chat.go:
  - 役割: `Service` インターフェースの主要な実装。LLM (Gemini, Ollama) API との通信、Gemini の Function Calling のディスパッチ（`WeatherService` および `URLReaderService` への委譲を含む）、応答生成のコアロジック、エラー時のフォールバック処理を担当。
  - 処理:
    - `NewChat(cfg *config.Config, historyMgr history.HistoryManager)`: サービスと依存関係 (`WeatherService`, `URLReaderService` を含む) を初期化。`cfg.Model` から `ModelConfig` を取得して保持する。`URLReaderService` から `get_url_content` の `FunctionDeclaration` を取得し、`WeatherService` のものと合わせて `genaiModel.Tools` に設定する。
    - `GetResponse(userID, threadID, username, message, timestamp, prompt)`: 保持している `ModelConfig` を使用する。ユーザーからのメッセージ、スレッドID、タイムスタンプを受け取り、`ModelConfig` の設定に基づいて使用するLLMを決定。`historyMgr` を使用してスレッドIDに基づいた履歴を取得・保存する。Function Calling が発生した場合、`fn.Name` が `get_url_content` であれば `urlReaderService` に処理を委譲する。Gemini API で 429 エラーが発生した場合のフォールバック処理などを行う。
    - `Close()`: Geminiクライアントを閉じる。

- chat/prompt.go:
  - 役割: LLM に送信するプロンプト（入力文字列）の構築ロジックを担当。
  - 処理:
    - `buildFullInput(systemPrompt, userMessage, historyMgr, userID, threadID, timestamp)`: システムプロンプト、現在日時情報、ツール指示（`get_url_content`、天気関連を含む）、会話履歴（スレッドIDとユーザーIDで取得、ロール名 "model" を "assistant" に変換）、ユーザーメッセージを結合する。

- chat/weather.go:
  - 役割: `WeatherService` の実装。天気関連の Function Calling 処理と `zu2l` API との連携を担当。
  - 処理:
    - `WeatherService` インターフェースと `weatherServiceImpl` 構造体を定義。
    - `NewWeatherService`: サービスの初期化。
    - `GetFunctionDeclarations`: 天気関連ツールの定義を返す。
    - `HandleFunctionCall` および `handle*` メソッド群: 各 Function Call の具体的な処理と `zu2l` API 呼び出し。地点情報取得や絵文字変換はヘルパー関数を利用。
    - `getLocationInfo`, `getWeatherEmoji` (ヘルパー関数): コードの共通化と可読性向上のための内部ヘルパー。`getWeatherEmoji` は天気コードを100の位で丸めて絵文字を検索する。
    - `weatherEmojiMap`: 天気コードと絵文字のマッピング。

- chat/utils.go:
  - 役割: `chat` パッケージ内で共通して使用されるヘルパー関数を提供する。
  - 処理:
    - `getResponseText`: Gemini の応答からテキスト部分を抽出する。

- chat/url_reader_service.go:
  - 役割: `URLReaderService` の実装。URLの内容取得とテキスト抽出、および `get_url_content` Function Declaration の提供を担当する。
  - 処理:
    - `NewURLReaderService`: サービスの初期化。
    - `GetURLContentAsText(url string) (string, error)`: 指定されたURLの内容を `http.Get` で取得し、`goquery` でHTMLをパースして主要なテキストを抽出する。抽出テキストは最大2000文字に制限される。
    - `GetURLReaderFunctionDeclaration() *genai.FunctionDeclaration`: `get_url_content` 関数の定義 (`FunctionDeclaration`) を返す。

- chat/ollama.go:
  - 役割: Ollama LLM との連携に関するロジックを担当。
  - 処理:
    - `(c *Chat) getOllamaResponse(userID, threadID, message, fullInput, ollamaCfg loader.OllamaConfig)`: 引数で受け取った `OllamaConfig` を使用して Ollama API にリクエストを送信し、応答を取得してスレッドIDとユーザーIDに基づいて履歴に追加する。
    - `parseOllamaStreamResponse`: Ollama からのストリーミング応答を1行ずつ解析し、`response` フィールドを連結する。

- history/history.go:
  - 役割: チャット履歴管理のインターフェース (`HistoryManager`) とインメモリ実装 (`InMemoryHistoryManager`)、および履歴メッセージの構造体 (`HistoryMessage`) を提供する。
  - 処理:
    - `HistoryMessage` 構造体: `Role` ("user" or "model") と `Content` を持つ。
    - `HistoryManager` インターフェース: `Add`, `Get`, `Clear`, `ClearAllByThreadID`, `Close` メソッドを定義。各メソッドは `error` を返す。`Get` は `[]HistoryMessage` を返す。
    - `InMemoryHistoryManager`: ユーザーID単位のメモリ上履歴管理。スレッドIDはダミー引数。履歴は最大サイズを超えると古いものからペアで削除。`Close` は何もしない。

- history/duckdb_manager.go:
  - 役割: `HistoryManager` インターフェースのDuckDB実装 (`DuckDBHistoryManager`) を提供する。
  - 処理:
    - `NewDuckDBHistoryManager`: DuckDBデータベースへの接続とテーブル (`thread_histories`) の初期化を行う。
    - `Add`: 既存履歴を読み込み、新しいメッセージと応答を追加後、全履歴をJSONで保存。古い履歴は削除しない。
    - `Get`: DBから全履歴を取得し、最新20ペアを返す。
    - `Clear`, `ClearAllByThreadID`: スレッドIDとユーザーIDに基づいてDuckDBデータベース内の履歴を操作する。
    - `Close`: DuckDBデータベース接続を閉じる。

- history/audit_log.go:
  - 役割: Discordメッセージイベント（作成、更新、削除）をログファイル (`log/audit.log`) に記録する。
  - 処理:
    - `InitAuditLog`: 監査ログファイルを初期化する。
    - `LogMessageCreate`, `LogMessageUpdate`, `LogMessageDelete`: 各イベントをログに書き込む。
    - `CloseAuditLog`: 監査ログファイルを閉じる。

- discord/discord.go:
  - 役割: Discord APIとのインターフェースを提供し、Botの起動、コマンド登録、リソース管理を行う。
  - 処理:
    - `StartBot(cfg *config.Config)`: Discord Botを起動し、`setupHandlers` を呼び出してハンドラとサービスを初期化する。起動シーケンス中の各ステップでエラーハンドリングとログ出力を行う。`HistoryManager`, `ChatService`, Discordセッションの `Close` メソッドが `defer` で確実に呼び出されるようにする。main.goからのシグナルで終了処理が行われるまでブロックする。

- loader/model.go:
  - 役割: `json/model.json` の読み込み処理と構造体定義 (`ModelConfig`, `OllamaConfig`, `About`) を行う。
  - 処理:
    - `ModelConfig` 構造体: `Name`, `ModelName`, `SecondaryModelName`, `Icon`, `MaxHistorySize`, `Prompts`, `About`, `Ollama` フィールドを持つ。
    - `LoadModelConfig(filepath)`: `json/model.json`ファイルを読み込み、`ModelConfig`構造体にマッピングする。
    - `GetPromptByUser(username)`: `ModelConfig` からユーザー固有またはデフォルトのプロンプトを取得する。

- json/custom_model.json:
  - 役割: ユーザーがプロンプトをカスタマイズするための設定ファイル。`discord/custom_prompt.go` によって管理される。
  - 処理:
    - ユーザーは、`/edit` コマンドを通じてこのファイルの内容を変更できる。

- discord/edit_command.go:
  - 役割: `/edit`コマンドの処理を行う。
  - 処理:
    - ユーザーから`/edit`コマンドを受け取り、`discord/custom_prompt.go` の `SetCustomPromptForUser` または `DeleteCustomPromptForUser` を呼び出してカスタムテンプレートを更新または削除する。
    - 更新または削除の結果をEmbedで表示する。エラーハンドリングは `sendErrorResponse` を使用。引数として `*config.Config` を受け取るように変更（以前は `chat.Service` で未使用だった）。

- discord/handler.go:
  - 役割: Discordのイベントハンドリングとコマンドディスパッチ、サービスの初期化を行う。
  - 処理:
    - `setupHandlers(s *discordgo.Session, cfg *config.Config, chatSvc chat.Service, historyMgr history.HistoryManager)`: Bot起動時に呼び出され、ログ設定、`DuckDBHistoryManager` と `ChatService` の初期化、コマンドハンドラの登録を行う。初期化された `HistoryManager` と `ChatService` を返す。`chat` パッケージとエラーロガーを共有する。
  - `interactionCreate` ハンドラ: 受け取ったインタラクションからスレッドID（スレッドでない場合はチャンネルID）を取得し、各コマンドハンドラ (`chatCommandHandler`, `resetCommandHandler`, `aboutCommandHandler`, `editCommandHandler`) に `cfg` や必要なサービスを渡す。`editCommandHandler` への引数が `cfg` に変更された。DMからのインタラクションの場合は、`i.User` からユーザー情報を取得します。
  - `onReady`: Bot準備完了時のログ出力。
  - `messageCreateHandler`: メッセージ作成イベントを処理します。DMの場合は `m.GuildID` が空文字列になることを利用してDMを判定し、`chatSvc.GetResponse` を呼び出して応答を生成し、DMで返信します。サーバー内メッセージの場合は監査ログに記録します。
  - `messageUpdateHandler`, `messageDeleteHandler`: メッセージの更新、削除イベントを `history.LogMessage*` を使って監査ログに記録する。

- discord/chat_command.go:
  - 役割: `/chat` コマンドの処理を行う。
  - 処理:
    - `chatCommandHandler(s, i, chatSvc, threadID, cfg)`: `cfg.Model` から `ModelConfig` を取得する。
    - `discord.GetCustomPromptForUser(cfg, username)` を使用してカスタムプロンプトを取得し、受け取った `threadID` と共に `chatSvc.GetResponse` を呼び出してLLMからの応答を取得し、結果をEmbedで表示する。

- discord/reset_command.go:
  - 役割: `/reset` コマンドの処理を行う。
  - 処理:
    - `resetCommandHandler(s, i, historyMgr, threadID)`: 受け取った `threadID` を使用して `historyMgr.ClearAllByThreadID` を呼び出し、該当スレッド（またはチャンネル）の全ユーザーの履歴を削除し、結果をEmbedで表示する。

- discord/about_command.go:
  - 役割: `/about` コマンドの処理を行う。
  - 処理:
    - `aboutCommandHandler(s, i, cfg)`: `cfg.Model` から `ModelConfig` を取得する。
    - 読み込んだ `modelCfg.About` を元にBotの説明Embedを作成して表示する。エラーはログに出力。

- discord/utils.go:
  - 役割: Discord関連の共通ユーティリティ関数を提供する。
  - 処理:
    - `errorLogger` (グローバル変数), `SetErrorLogger`: エラーロガーの設定。
    - `sendErrorResponse`: エラーメッセージをContentに設定して応答する。HTTP 500 Internal Server Errorの場合、「返事の文章が長すぎたみたい…」というメッセージを追加する。
    - `sendEphemeralErrorResponse`: 本人にのみ見えるエラー応答を送信する。

- discord/embeds.go:
  - 役割: Discord Embed作成に関するヘルパー関数を提供する。
  - 処理:
    - `SplitToEmbedFields`: LLMからの応答テキストを受け取り、Discord Embedフィールドの制約に合わせて分割するヘルパー関数。テキストをルーン単位で処理し、各フィールドが1024文字を超えないように分割する。フィールドの最大数(5)や合計文字数の上限(5120)に達した場合は、最後のフィールドの末尾に省略記号 "..." を付与して処理を終える。

- discord/custom_prompt.go:
  - 役割: `json/custom_model.json` の書き込み処理（設定、削除）と、ファイル直接読み込みのヘルパー関数を提供する。
  - 処理:
    - `GetCustomPromptForUser(cfg *config.Config, username string)`: `config.Config` を引数に取り、そこからカスタムプロンプトを取得する。
    - `SetCustomPromptForUser`, `DeleteCustomPromptForUser`: ユーザー固有のプロンプトを設定、削除する。内部で `loadCustomPromptsFromFile` を呼び出して最新のファイル内容を読み込んでから保存する。
    - `loadCustomPromptsFromFile`: ファイルを直接読み込むヘルパー関数。通常は `config.LoadConfig` で読み込まれた設定を使用する。
    - `saveCustomPrompts`: ファイルへの書き込みとMutexによる排他制御を行う。
