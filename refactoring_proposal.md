# discord/handler.go のリファクタリング提案

## 1. `interactionCreate` 関数の分割

`interactionCreate` 関数が `chat` と `reset` の処理を両方持っているため、関数が長くなっています。
それぞれの処理を別の関数に分割することで、可読性と保守性を向上させることができます。

```go
func handleChatInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, modelCfg *loader.ModelConfig, ctx context.Context) {
    // chat の処理
}

func handleResetInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, chatSvc chat.Service, modelCfg *loader.ModelConfig, ctx context.Context) {
    // reset の処理
}

func interactionCreate(chatSvc chat.Service, modelCfg *loader.ModelConfig, ctx context.Context) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "chat":
			handleChatInteraction(s, i, chatSvc, modelCfg, ctx)
		case "reset":
			handleResetInteraction(s, i, chatSvc, modelCfg, ctx)
		}
	}
}
```

## 2. エラーハンドリングの共通化

`GetResponse` のエラーハンドリングが複数箇所で同じように記述されています。
共通のエラーハンドリング関数を作成することで、コードの重複を減らし、保守性を向上させることができます。

```go
func handleGetResponseError(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	log.Printf("GetResponse error: %v", err)
	content := fmt.Sprintf("エラーが発生しました: %v", err)
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		log.Printf("InteractionResponseEdit error: %v", err)
	}
}
```

## 3. ログメッセージの作成

ログメッセージの作成をもっとスマートにできる可能性があります。
例えば、構造体を使ってログメッセージを作成することで、可読性を向上させることができます。

```go
type LogMessage struct {
	Username  string
	Message   string
	Timestamp string
}

func (l LogMessage) String() string {
	return fmt.Sprintf("User %s sent message: %s at %s", l.Username, l.Message, l.Timestamp)
}

// ...

logMessage := LogMessage{
	Username:  username,
	Message:   message,
	Timestamp: timestamp,
}
log.Printf(logMessage.String())
```

# discord/discord.go のリファクタリング提案

### 1. `StartBot` 関数の分割

`StartBot` 関数がボットの起動処理、コマンドの登録処理、シグナルハンドリングなどをすべて行っているため、関数が長くなっています。
それぞれの処理を別の関数に分割することで、可読性と保守性を向上させることができます。

```go
func setupDiscordSession(cfg *config.Config) (*discordgo.Session, error) {
	// セッションの作成
}

func registerCommands(s *discordgo.Session, commands []*discordgo.ApplicationCommand) ([]*discordgo.ApplicationCommand, error) {
	// コマンドの登録
}

func handleSignals(sc chan os.Signal) {
	// シグナルハンドリング
}

func StartBot(cfg *config.Config) error {
	// 各関数の呼び出し
}
```

### 2. コマンドの登録処理

コマンドの登録処理でエラーが発生した場合に `continue` している箇所があります。
エラーの内容によっては処理を中断した方がいいかもしれません。

### 3. `session.AddHandler` の中でコマンドを登録している

`session.Open()` の前にコマンドを登録した方が、ボットの起動が早くなるかもしれません。

# chat/chat.go のリファクタリング提案

### 1. `/reset` コマンドの処理

`GetResponse` 関数の中で `/reset` コマンドを処理しているけど、これは別の関数に分けた方が、関数の役割が明確になるかも。

### 2. メッセージ履歴の保持

メッセージ履歴を保持する件数を固定値 (20件) にしているけど、設定ファイルから変更できるようにした方が、柔軟性が高まるかも。

### 3. `getResponseText` 関数

`getResponseText` 関数の中で、APIからの応答がない場合のメッセージを固定値にしているけど、これも設定ファイルから変更できるようにした方が、柔軟性が高まるかも。
