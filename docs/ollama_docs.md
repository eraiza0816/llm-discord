# Ollama 導入タスク指示書

## 目的

本タスクでは、既存の Discord Bot「ぺちこ」に Ollama を使用した LLM との会話機能を追加します。
これにより、Gemini に加えて Ollama を利用できるようになり、LLM の選択肢を広げることができます。

## 前提条件

*   Discord Bot「ぺちこ」が正常に動作していること。
*   json/model.json にある Ollamaと通信が可能かチェックすること。
*   `json/model.json` に Ollama の設定が追加されていること。

## タスク概要

1.  **Ollama 対応:**
    *   `chat/chat.go` を修正し、Ollama と Gemini の両方に対応できるようにします。
        *   `GetResponse` メソッドを修正し、`json/model.json` の `ollama.enabled` の値に基づいて、Ollama または Gemini を使用するように切り替えます。
        *   Ollama との通信に失敗した場合のエラーハンドリングを追加し、Gemini にフォールバックするようにします。
2.  **設定ファイルの読み込み:**
    *   `loader/model.go` を修正し、`json/model.json` から `ollama.api_endpoint` と `ollama.model_name` を読み込むようにします。
3.  **環境変数の読み込み:**
    *   `config/config.go` を修正し、環境変数から Ollama の API キーなどを読み込むようにします (必要な場合)。
4.  **Discord ハンドラーの修正:**
    *   `discord/handler.go` を修正し、Ollama を使用する場合の処理を追加します (必要な場合)。
5.  **初期化処理の追加:**
    *   `main.go` を修正し、Ollama を使用する場合の初期化処理を追加します (必要な場合)。
6.  **設定ファイルの確認:**
    *   `json/model.json` に以下の Ollama の設定が追加されていることを確認します。
        *   `enabled` (bool): Ollama を使用するかどうか。
        *   `api_endpoint` (string): Ollama の API エンドポイント。
        *   `model_name` (string): Ollama のモデル名。

## 実装時の注意

*   ログ出力は Go の標準機能を使ってください (外部ライブラリは使用しないでください)。
*   `json/model.json` は毎回読み込むように実装してください (起動時にのみ読み込むような実装にはしないでください)。
*   `json/model.json` への書き込みは禁止されています。編集してはいけません。
*   `.env` ファイルへの書き込みは禁止されています。
*   実装が完了したら、`go run main.go` を実行して、実行時にエラーがないことを確認してください。
*   エラーがないことの確認が取れたら、`docs/ddd_document.md` の内容をアップデートしてください。

## エラーハンドリング

*   Ollama との通信に失敗した場合、以下の手順でエラーハンドリングを行います。
    1.  エラーログを出力します (Go の標準機能を使用)。
    2.  Gemini にフォールバックします。
    3.  ユーザーにエラーメッセージを表示します (例: "Ollama との通信に失敗したため、Gemini を使用します")。

## テスト
テストは各段階でユーザと一緒に実行します。
1.  `json/model.json` の `ollama.enabled` を `true` に設定し、Ollama が正常に動作することを確認します。
2.  `json/model.json` の `ollama.enabled` を `false` に設定し、Gemini が正常に動作することを確認します。
3.  Ollama が利用できない状態 (例: Ollama サーバーが停止している) で、Gemini に正常にフォールバックすることを確認します。

## 成果物

*   修正された Go のソースコード (`chat/chat.go`, `loader/model.go`, `config/config.go`, `discord/handler.go`, `main.go`)
*   更新されたドキュメント (`docs/ddd_document.md`)
