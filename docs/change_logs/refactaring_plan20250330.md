# リファクタリング計画

このドキュメントは、`llm-discord-go` プロジェクトのリファクタリング計画を示すものです。目的は、コードの可読性、保守性、テスト容易性を向上させることです。

## 1. パッケージ構成の見直し

### 1.1. `discord` パッケージの共通処理を分離

**現状:**
`discord` パッケージ内の各コマンドハンドラ (`chat_command.go`, `edit_command.go` など) に、Embed の作成やエラー応答処理などの共通ロジックが散見されます。

**指示:**
- `discord` パッケージ内に `utils.go` ファイルを作成します。
- 各コマンドハンドラで共通して使用されているエラー応答処理（例: `s.InteractionResponseEdit` でエラーメッセージを返す部分）を `utils.go` に共通関数として切り出します。
- `discord` パッケージ内に `embeds.go` ファイルを作成します。
- `chat_command.go` にある `splitToEmbedFields` 関数を `embeds.go` に移動します。
- 各コマンドハンドラで使用されている Embed の作成ロジックを分析し、共通化できる部分があれば `embeds.go` にヘルパー関数として切り出します。

### 1.2. `chat` パッケージの履歴管理ロジックを分離

**現状:**
`chat/chat.go` 内の `Chat` 構造体と関連メソッドで、LLM との対話履歴 (`userHistories`) の管理が行われています。

**指示:**
- 新しいパッケージ `history` を作成します (`history/history.go`)。
- `chat/chat.go` から `userHistories` フィールド、`userHistoriesMutex` フィールド、`ClearHistory` メソッド、および関連する履歴操作ロジックを `history` パッケージに移動します。
- `history` パッケージは、ユーザーID ごとに履歴を管理する機能（追加、取得、クリア）を提供するインターフェースと実装を提供します。
- `chat.Chat` 構造体は `history` パッケージのインターフェースに依存するように変更します。

## 2. 設定ファイルの読み込み処理の改善

### 2.1. `model.json` の読み込み共通化

**現状:**
`.clinerules` の指示に従い、`discord/chat_command.go` と `discord/about_command.go` で `loader.LoadModelConfig("json/model.json")` が個別に呼び出されています。

**指示:**
- `discord/handler.go` の `setupHandlers` 関数内で `loader.LoadModelConfig("json/model.json")` を呼び出すように変更します。
- 読み込んだ `modelCfg` を、各コマンドハンドラ関数 (`chatCommandHandler`, `aboutCommandHandler` など) に引数として渡すように、ハンドラの登録部分と関数シグネチャを変更します。
- 各コマンドハンドラ内の個別の `loader.LoadModelConfig` 呼び出しを削除します。
- **注意:** `.clinerules` の「毎回読み込む」という要件は、リクエストごとにハンドラが呼ばれる際に `model.json` の内容が最新であるべき、という意味合いと解釈し、起動時に1回読み込むのではなく、リクエスト処理の起点に近い `setupHandlers` で読み込むことで、ある程度その意図を汲みます。（もし厳密に *毎回* ファイルを読む必要がある場合は、各ハンドラ内で読む現状維持が正しいですが、パフォーマンスへの影響を考慮すると、この共通化が現実的です。）

### 2.2. `custom_model.json` の読み書きロジック共通化

**現状:**
`discord/chat_command.go` と `discord/edit_command.go` で、`json/custom_model.json` の読み込みと書き込みを行うロジックが重複しています。

**指示:**
- `discord` パッケージ内に `custom_prompt.go` ファイルを作成します。
- `json/custom_model.json` から特定のユーザーのプロンプトを読み込む関数を作成します。
- `json/custom_model.json` に特定のユーザーのプロンプトを書き込む（または削除する）関数を作成します。
- `discord/chat_command.go` と `discord/edit_command.go` 内の該当ロジックを、作成した共通関数を呼び出すように置き換えます。

## 3. エラーハンドリングの統一

**現状:**
各コマンドハンドラでのエラー発生時の Discord への応答メッセージ形式やログ出力が統一されていません。また、`log.Fatalf` が不適切な箇所で使用されています。

**指示:**
- `discord/utils.go` に、エラーを受け取り、標準的なエラー Embed を生成して Discord に応答する共通関数（例: `sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error)`）を作成します。
- 各コマンドハンドラ内のエラーハンドリング部分を、この共通関数を呼び出すように修正します。
- `config/config.go` や `discord/handler.go` 内の `log.Fatalf` を見直し、エラーを呼び出し元に返すように修正します。プログラムの終了は `main` 関数でのみ行うようにします。

## 4. LLM クライアント処理の整理

**現状:**
`chat/chat.go` の `getOllamaResponse` メソッド内で、Ollama API との通信とストリーミングレスポンスの解析処理が行われています。

**指示:**
- `getOllamaResponse` 内のストリーミングレスポンス（`bufio.NewReader` で読み込んでいるループ部分）の解析ロジックを、別のプライベート関数（例: `parseOllamaStreamResponse(reader *bufio.Reader) (string, error)`）に切り出します。
- `getOllamaResponse` はこの新しい関数を呼び出すように修正します。

## 5. 型定義の整理

**現状:**
`discord/chat_command.go` で定義されている `CustomModelConfig` 構造体が `discord/edit_command.go` でも使用されています。

**指示:**
- `discord` パッケージ内に `types.go` ファイルを作成します。
- `CustomModelConfig` 構造体の定義を `discord/types.go` に移動します。
- `discord/chat_command.go` と `discord/edit_command.go` は `discord/types.go` で定義された型を使用するように修正します。

## 6. テストの追加

**現状:**
`main_test.go` 以外にユニットテストが存在しません。

**指示:**
- `chat` パッケージの `GetResponse`, `ClearHistory` などの主要なロジックに対するユニットテストを作成します。
- `loader` パッケージの `LoadModelConfig`, `GetPromptByUser` に対するユニットテストを作成します。
- `history` パッケージ（新規作成）の関数に対するユニットテストを作成します。
- `discord` パッケージ内の `utils.go`, `embeds.go`, `custom_prompt.go`（新規作成）のヘルパー関数に対するユニットテストを作成します。
- 可能であれば、各コマンドハンドラの主要なロジック部分（LLM サービス呼び出しや Embed 生成部分を除く）に対するユニットテストを作成します（`discordgo` のモックが必要になる場合があります）。
