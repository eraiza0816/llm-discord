# Ollama連携に関する指示書

## 目的

本指示書は、Discord Bot「ぺちこ」に、Google Geminiに加えて、OllamaをLLMとして利用可能にするための実装に関する詳細を定めるものです。

## 前提

- Discord Bot「ぺちこ」が既にGoogle Geminiと連携し、チャット機能を提供していること。
- Ollamaが指定されたエンドポイントで利用可能であること。
- `json/model.json`にOllamaのエンドポイントが定義されていること。

## 実装内容

1. **Ollama関連の設定項目を`json/model.json`に追加:**
   - Ollamaのエンドポイント (`ollama_endpoint`) を設定ファイルに追加します。
     - 例: `"ollama_endpoint": "http://192.168.200.19/ollama"`
   - Ollamaを使用するかどうかを切り替えるフラグ (`use_ollama`) を設定ファイルに追加します。
     - 例: `"use_ollama": true`

2. **`ollama.go`ファイルの作成:**
   - Ollama APIとの通信を担う`ollama.go`ファイルを新規作成します。
   - `ollama.go`には、以下の機能を持たせること。
     - 利用できる model の一覧を取得
     - Ollama APIへのリクエスト送信
     - Ollama APIからのレスポンス受信
     - エラーハンドリング

3. **既存モジュールとの連携:**
   - ユーザーからのリクエストに応じて、GeminiまたはOllamaをLLMとして選択するロジックを実装します。
   - `json/model.json`の`use_ollama`フラグが`true`の場合、Ollamaを使用します。
   - `chat/chat.go`などの既存モジュールから、`ollama.go`の機能を呼び出せるように実装します。

4. **ログ出力:**
   - Ollama APIとの通信に関するログをGoの標準機能を用いて出力します。
   - ログには、リクエスト内容、レスポンス内容、エラー情報などを記録します。

## 実装時の注意点

- `json/model.json`ファイルへの書き込みは禁止されています。
- `.env`ファイルには正確な認証情報が含まれており、内容の正しさについて確認する必要はありません。
- 実装対象のLLMは Gemini と Ollama のみです。他のLLMは対象外です。
- `json/model.json` は毎回読み込む実装にしてください。起動時にのみ読み込むような実装に変更しないでください。
- ログ出力では外部ライブラリの使用はできません。Goの標準機能のみを使用してください。

## 動作確認

- `go run main.go`を実行して、実行時にエラーが発生しないことを確認してください。
- `use_ollama`フラグを切り替えることで、GeminiとOllamaが正しく切り替わることを確認してください。
- Ollama APIとの通信が正常に行われ、レスポンスが取得できることを確認してください。

## その他

- 不明な点があれば、docs/ddd_document.md を参照してください。
- それでも分からない場合は，ユーザに問い合わせてください。
