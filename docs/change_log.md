## 変更履歴
- 2025/05/02: `discord/embeds.go` の `splitToEmbedFields` を修正し、マルチバイト文字が分割されて文字化けする問題を解消。
- 2025/04/02: `/chat`, `/about` コマンド実行時に `model.json` を読み込むように変更。
- 2025/04/02: Gemini API 429 エラー時に `secondary_model_name` で再試行し、Ollama へフォールバックする機能を追加。`loader.ModelConfig` に `SecondaryModelName` フィールドを追加。
- 2025/04/02: `discord/embeds.go` の `splitToEmbedFields` を修正し、1024文字を超える場合に複数のフィールドに分割して表示するように変更。
- 2025/04/02: `chat/chat.go` に Ollama と Gemini の処理分岐を追加。`chat/ollama.go` の履歴追加処理を有効化。
- 2025/04/02: `discord/embeds.go` の `splitToEmbedFields` を修正し、1024文字を超える場合に省略表示するように変更。Discordの文字数制限エラーに対応。
- 2025/04/02: `discord/embeds.go` の `splitToEmbedFields` を修正し、Embedフィールド名を空にして非表示に変更。
- 2025/04/01: `chat/chat.go` 内の冗長なデバッグログ出力をコメントアウトし、ログ量を削減。
- 2025/04/01: コード中のコメントの表現を丁寧語から簡潔な記述に修正。