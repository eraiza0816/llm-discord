package chat

// ChatParams はチャット処理に必要なパラメータをカプセル化します。
type ChatParams struct {
	UserID    string
	ThreadID  string
	Username  string
	Message   string
	Timestamp string
	Prompt    string
	IsBot     bool
}

// ChatResponse はチャット処理の結果をカプセル化します。
type ChatResponse struct {
	Text      string
	ElapsedMs float64
	ModelName string
}

// LLMProvider はLLMプロバイダを表します。
type LLMProvider int

const (
	ProviderGemini LLMProvider = iota
	ProviderOllama
	ProviderOpenAI
)

// ModelSelection はLLM選択結果を表します。
type ModelSelection struct {
	Provider  LLMProvider
	OllamaCfg OllamaConfig // ProviderOllama の場合のみ有効
	OpenAICfg OpenAIConfig // ProviderOpenAI の場合のみ有効
	GeminiModelName string // ProviderGemini の場合のモデル名
}

// OpenAIConfig はOpenAI互換APIの設定を表します。
type OpenAIConfig struct {
	Enabled     bool
	APIEndpoint string
	ModelName   string
	APIKey      string
}

// OllamaConfig はOllamaの設定を表します。
type OllamaConfig struct {
	Enabled     bool
	APIEndpoint string
	ModelName   string
}
