# DDDドキュメント

## プロジェクトの目的と概要

このプロジェクトは、Discord上でGoogleのGeminiとチャットできるDiscord Botです。
ユーザーはDiscordのインターフェースを通じてGeminiと対話できます。

## 主要な機能

- Geminiとのチャット機能: ユーザーはテキストメッセージを送信し、Geminiからの応答を受信できます。
- チャット履歴のリセット機能: ユーザーは`/reset`コマンドを使用してチャット履歴をクリアできます。
- BOTの説明表示機能: `/about`コマンドで、json/model.json に定義されたBOTの説明を表示します。

## ドメインに関する用語（ユビキタス言語）

- Bot: Discord Bot
- Gemini: Googleの大規模言語モデル Gemini
- チャット履歴: ユーザーごとのGeminiとのチャットの履歴
- コマンド: Botへの指示 (例: `/chat`, `/reset`, `/about`)
- プロンプト: Geminiへの指示文。ユーザーごと、またはデフォルトの指示文が設定可能。

## コンテキストマップ

- サブドメイン:
  - チャット: ユーザーとGeminiの対話の管理
  - コマンド処理: ユーザーからのコマンドの解析と実行
  - 設定管理: Botの設定の読み込みと管理
  - ユーザー管理: ユーザーの識別と履歴の管理
- 境界づけられたコンテキスト:
  - Discord Bot: Discord APIとのインターフェース
  - Gemini API: Google Geminiとのインターフェース
  - 設定ファイル: (json/model.json)
- 関係:
  - Discord BotはGemini APIを利用してチャット機能を提供します。
  - Discord Botはユーザーからのコマンドを受け付けます。
  - Discord Botはユーザーのチャット履歴を管理します。
  - Discord Botは設定ファイルから設定を読み込みます。

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

- **設定 (ModelConfig)**
  - モデル名: 使用するGeminiのモデル名 (model_name)。
  - アイコン: Botのアイコン (icon)。
  - プロンプト: Geminiへの指示文 (prompts)。ユーザーごと、またはデフォルトの指示文が設定可能。
  - BOTの説明: BOTの説明 (about)。

### ドメインサービス

- **ChatService**
  - 役割: Gemini APIとのやり取りを行う。
  - メソッド:
    - `GetResponse(userID, username, message, timestamp, prompt)`: Gemini APIを呼び出し、応答を取得する。
    - `ClearHistory(userID)`: ユーザーのチャット履歴をクリアする。

- **ConfigService**
  - 役割: 設定ファイルの読み込みと管理を行う。
  - メソッド:
    - `LoadModelConfig(filepath)`: jsonファイルからModelConfigを読み込む。

### ドメインイベント

- **メッセージ送信イベント (MessageSentEvent)**
  - 概要: ユーザーがメッセージを送信したときに発生するイベント。
  - イベントハンドラー:
    - チャット履歴の更新
    - Gemini APIへのメッセージ送信

- **チャット履歴クリアイベント (ChatHistoryClearedEvent)**
  - 概要: ユーザーがチャット履歴をクリアしたときに発生するイベント。
  - イベントハンドラー:
    - チャット履歴の削除

## プログラムファイルの詳細

- **main.go:**
  - 役割: Discord Botの起動と設定を行う。
  - 処理:
    - `config.LoadConfig()`: 環境変数を読み込み、設定をロードする。
    - `discord.StartBot(cfg)`: Discord Botを起動する。

- **config/config.go:**
  - 役割: 環境変数の読み込みと管理を行う。
  - 処理:
    - `LoadConfig()`: `.env`ファイルを読み込み、環境変数を設定する。

- **chat/chat.go:**
  - 役割: Gemini APIとの通信処理を行う。
  - 処理:
    - `NewChat(token, model, defaultPrompt)`: Gemini APIクライアントを初期化する。
    - `GetResponse(userID, username, message, timestamp, prompt)`: Gemini APIを呼び出し、応答を取得する。
    - `ClearHistory(userID)`: チャット履歴をクリアする。

- **discord/discord.go:**
  - 役割: Discord APIとのインターフェースを提供し、Botの起動、コマンド登録を行う。
  - 処理:
    - `StartBot(cfg)`: Discord Botを起動し、コマンドハンドラーを登録する。

- **loader/model.go:**
  - 役割: json/model.jsonの読み込み処理を行う。
  - 処理:
    - `LoadModelConfig(filepath)`: `json/model.json`ファイルを読み込み、`ModelConfig`構造体にマッピングする。

- **discord/handler.go:**
  - 役割: Discordのコマンドハンドラーを定義する。
  - 処理:
    - `interactionCreate(chatSvc, modelCfg)`: コマンドハンドラーを登録する。
    - `interactionHandler(s, i, chatSvc, modelCfg)`: `/chat`、`/reset`、`/about`などのコマンド処理を個別のハンドラー関数に委譲する。
    - `chatCommandHandler(s, i, chatSvc, modelCfg)`: `/chat`コマンドの処理を行う。
    - `resetCommandHandler(s, i, chatSvc)`: `/reset`コマンドの処理を行う。
    - `aboutCommandHandler(s, i, modelCfg)`: `/about`コマンドの処理を行う。

## 今後の展望
- ユーザから /edit を受け取るとプロンプトテンプレートを受け取った内容に変更する機能が欲しい
