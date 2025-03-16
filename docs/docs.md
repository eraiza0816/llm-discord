# DDDドキュメント

## プロジェクトの目的と概要

このプロジェクトは、Discord上でGoogleのGeminiとチャットできるDiscord Botです。
ユーザーはDiscordのインターフェースを通じてGeminiと対話できます。

## 主要な機能

- Geminiとのチャット機能: ユーザーはテキストメッセージを送信し、Geminiからの応答を受信できます。
- チャット履歴のリセット機能: ユーザーは`/reset`コマンドを使用してチャット履歴をクリアできます。
- BOTの説明表示機能:  json/embed/about.json.sample を読み込むことで実現できる。

## ドメインに関する用語（ユビキタス言語）

- Bot: Discord Bot
- Gemini: GoogleのGemini
- チャット履歴: ユーザーごとのチャットの履歴
- コマンド: Botへの指示 (例: `/chat`, `/reset`)
- プロンプト: Geminiへの指示文

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

## ドメインモデルの概要

- エンティティ:
  - ユーザー: Discordのユーザー (userID, username)。Discord APIから取得。
  - メッセージ: ユーザーまたはBotからのメッセージ (content, timestamp, userID)。チャット履歴として保存。
- 値オブジェクト:
  - タイムスタンプ: メッセージの送信時刻。
  - 設定:  ModelConfig (name, model_name, icon, prompts)。json/model.jsonから読み込み。
- ドメインサービス:
  - Chat: Gemini APIとのやり取りを行うサービス (GetResponse, ClearHistory)。Gemini APIを呼び出し、プロンプトを生成。
  - 設定ローダー:  ModelConfigをjsonファイルから読み込むサービス (LoadModelConfig)。json/model.jsonを読み込み、ModelConfigを生成。

## プログラムファイルの詳細

- **main.go:**
  - Discord Botの起動と設定を行います。
  - `discord.New`でDiscord Botのクライアントを作成し、`discord.AddHandlers`でコマンドハンドラーを登録します。
  - `chat.New`でGemini APIとの連携を初期化し、`config.LoadEnv`で環境変数を読み込みます。
  - コマンドハンドラーを登録し、Gemini APIとの連携やチャット履歴の管理を行います。

- **config/config.go:**
  - Botの設定を読み込み、環境変数を管理します。
  - `LoadEnv`関数で`.env`ファイルを読み込み、環境変数を設定します。

- **chat/chat.go:**
  - Gemini APIとの通信処理を行います。
  - `GetResponse`関数でGemini APIを呼び出し、応答を取得します。
  - `ClearHistory`関数でチャット履歴をクリアします。
  - プロンプトの生成を行います。

- **discord/discord.go:**
  - Discord APIとのインターフェースを提供します。
  - `New`関数でDiscord Botのクライアントを作成します。
  - `AddHandlers`関数でコマンドハンドラーを登録します。
  - メッセージの送受信を行います。

- **json/model.json:**
  - Botの設定ファイルです。
  - モデル名、アイコン、プロンプトなどの設定を定義します。

- **loader/model.go:**
  - json/model.jsonの読み込み処理を行います。
  - `LoadModelConfig`関数で`json/model.json`ファイルを読み込み、`ModelConfig`構造体にマッピングします。

- **discord/handler.go:**
  - Discordのコマンドハンドラーです。
  - `/chat`、`/reset`などのコマンド処理を行います。
  - `ChatCommand`関数で`/chat`コマンドを処理し、Gemini APIにメッセージを送信します。
  - `ResetCommand`関数で`/reset`コマンドを処理し、チャット履歴をクリアします。
