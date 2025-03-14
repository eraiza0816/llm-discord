# DDDドキュメント

## プロジェクトの目的と概要

このプロジェクトは、Discord上でGoogleのGeminiとチャットできるDiscord Botです。

## 主要な機能

- Geminiとのチャット機能
- チャット履歴のリセット機能
- BOTの説明表示機能

## ドメインに関する用語（ユビキタス言語）

- Bot: Discord Bot
- Gemini: GoogleのGemini
- チャット履歴: ユーザーごとのチャットの履歴
- コマンド: Botへの指示

## コンテキストマップ（サブドメイン、境界づけられたコンテキスト、関係）

- サブドメイン:
  - チャット
  - コマンド
  - ユーザー管理
- 境界づけられたコンテキスト:
  - Discord Bot
  - Gemini API
- 関係:
  - Discord BotはGemini APIを利用してチャット機能を提供します。
  - Discord Botはユーザーからのコマンドを受け付けます。
  - Discord Botはユーザーのチャット履歴を管理します。

## ドメインモデルの概要（エンティティ、値オブジェクト、ドメインサービスなど）

- エンティティ:
  - ユーザー: Discordのユーザー
  - メッセージ: ユーザーまたはBotからのメッセージ
- 値オブジェクト:
  - タイムスタンプ: メッセージの送信時刻
- ドメインサービス:
  - Chat: Gemini APIとのやり取りを行うサービス
  - JsonLoad: JSONファイルの読み込みを行うサービス

## ディレクトリ構成

```
.
├── .env
├── go.mod
├── go.sum
├── main_test.go
├── main.go
├── README.md
├── docs
│   └── docs.md （このファイル）
├── json
│   ├── command.json
│   ├── model.json.sample
│   └── embed
│       └── about.json.sample
└── log （ログ置き場）