package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var errorLogger *log.Logger

type Service interface {
	GetResponse(ctx context.Context, userID, threadID, username, message, timestamp, prompt string, isBot bool) (string, float64, string, error)
	Close()
}

type Chat struct {
	genaiClient *genai.Client
	genaiModel  *genai.GenerativeModel
	historyMgr  history.HistoryManager
	modelConfig *loader.ModelConfig
	config      *config.Config
}

func NewChat(cfg *config.Config, historyMgr history.HistoryManager) (Service, error) {
	errorLogFile, err := os.OpenFile("log/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open error log file: %v", err)
		errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		errorLogger = log.New(errorLogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	initialModelCfg := cfg.Model
	initialGeminiModelName := initialModelCfg.ModelName

	genaiClient, err := genai.NewClient(context.Background(), option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("Geminiクライアントの作成に失敗: %w", err)
	}
	genaiModel := genaiClient.GenerativeModel(initialGeminiModelName)

	return &Chat{
		genaiClient: genaiClient,
		genaiModel:  genaiModel,
		historyMgr:  historyMgr,
		modelConfig: initialModelCfg,
		config:      cfg,
	}, nil
}

func (c *Chat) GetResponse(ctx context.Context, userID, threadID, username, message, timestamp, defaultSystemPrompt string, isBot bool) (string, float64, string, error) {
	if isBot {
		count, err := c.historyMgr.GetBotConversationCount(threadID, userID)
		if err != nil {
			errorLogger.Printf("Failed to get bot conversation count: %v", err)
		}
		if count >= 3 {
			log.Printf("Botとの会話が3回に達したため、応答を中断します。UserID: %s, ThreadID: %s", userID, threadID)
			return "", 0, "", nil
		}
	}

	modelCfg := c.modelConfig
	if isBot && modelCfg.Ollama.Enabled {
		log.Printf("Botとの対話のため、Ollamaモデルを強制的に使用します。UserID: %s", userID)
	}

	currentSystemPrompt := modelCfg.GetPromptByUser(username)
	fullInput := buildFullInput(currentSystemPrompt, message, c.historyMgr, userID, threadID, timestamp)

	if modelCfg.Ollama.Enabled {
		return c.invokeOllama(ctx, userID, threadID, message, fullInput, modelCfg)
	}
	if modelCfg.OpenAI.Enabled {
		return c.invokeOpenAI(ctx, userID, threadID, message, fullInput, modelCfg)
	}
	return c.invokeGemini(ctx, userID, threadID, message, fullInput, modelCfg)
}

func (c *Chat) invokeOllama(ctx context.Context, userID, threadID, message, fullInput string, modelCfg *loader.ModelConfig) (string, float64, string, error) {
	log.Printf("Using Ollama (%s) for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
	responseText, elapsed, err := c.getOllamaResponse(ctx, userID, threadID, message, fullInput, modelCfg.Ollama)
	if err != nil {
		errorLogger.Printf("Ollama API call failed for user %s in thread %s: %v", userID, threadID, err)
		return "", elapsed, modelCfg.Ollama.ModelName, fmt.Errorf("Ollama APIからのエラー: %w", err)
	}
	return responseText, elapsed, modelCfg.Ollama.ModelName, nil
}

func (c *Chat) invokeOpenAI(ctx context.Context, userID, threadID, message, fullInput string, modelCfg *loader.ModelConfig) (string, float64, string, error) {
	log.Printf("Using OpenAI compatible API (%s) for user %s in thread %s", modelCfg.OpenAI.ModelName, userID, threadID)
	responseText, elapsed, err := c.getOpenAIResponse(ctx, userID, threadID, message, fullInput, modelCfg.OpenAI)
	if err != nil {
		errorLogger.Printf("OpenAI API call failed for user %s in thread %s: %v", userID, threadID, err)
		return "", elapsed, modelCfg.OpenAI.ModelName, fmt.Errorf("OpenAI APIからのエラー: %w", err)
	}
	return responseText, elapsed, modelCfg.OpenAI.ModelName, nil
}

func (c *Chat) invokeGemini(ctx context.Context, userID, threadID, message, fullInput string, modelCfg *loader.ModelConfig) (string, float64, string, error) {
	log.Printf("Using Gemini (%s) for user %s", modelCfg.ModelName, userID)
	c.genaiModel = c.genaiClient.GenerativeModel(modelCfg.ModelName)

	start := time.Now()
	resp, err := c.genaiModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		return c.handleGeminiError(ctx, userID, threadID, message, fullInput, modelCfg, elapsed, err)
	}

	return c.processGeminiResponse(ctx, userID, threadID, message, modelCfg, resp, start, elapsed)
}

func (c *Chat) handleGeminiError(ctx context.Context, userID, threadID, message, fullInput string, modelCfg *loader.ModelConfig, elapsed float64, err error) (string, float64, string, error) {
	errorLogger.Printf("Initial Gemini API call failed for model %s: %v", modelCfg.ModelName, err)

	gapiErr, isQuotaExceeded := err.(*googleapi.Error)
	if !isQuotaExceeded || gapiErr.Code != 429 {
		errorLogger.Printf("Gemini API error: input=%q err=%v", fullInput, err)
		return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIからのエラー: %w", err)
	}

	log.Printf("Quota exceeded for model %s. Attempting fallback...", modelCfg.ModelName)

	if modelCfg.SecondaryModelName != "" {
		secResp, secElapsed, secErr := c.invokeSecondaryModel(ctx, fullInput, modelCfg)
		if secErr == nil {
			return c.processGeminiResponse(ctx, userID, threadID, message, modelCfg, secResp, time.Now(), secElapsed)
		}
		elapsed = secElapsed
		err = secErr
	} else {
		log.Println("Secondary model name not configured.")
	}

	if modelCfg.Ollama.Enabled {
		log.Printf("Falling back to Ollama (%s) for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
		responseText, ollamaElapsed, ollamaErr := c.getOllamaResponse(ctx, userID, threadID, message, fullInput, modelCfg.Ollama)
		if ollamaErr != nil {
			errorLogger.Printf("Ollama fallback failed for user %s in thread %s: %v", userID, threadID, ollamaErr)
			return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIクォータ超過後、Ollamaフォールバックも失敗: (Gemini: %w), (Ollama: %v)", err, ollamaErr)
		}
		log.Printf("Successfully generated content with Ollama fallback: %s for user %s in thread %s", modelCfg.Ollama.ModelName, userID, threadID)
		return responseText, ollamaElapsed, modelCfg.Ollama.ModelName, nil
	}

	log.Println("Ollama is not enabled, cannot fallback.")
	return "", elapsed, modelCfg.ModelName, fmt.Errorf("Gemini APIクォータ超過、フォールバック先なし: %w", err)
}

func (c *Chat) invokeSecondaryModel(ctx context.Context, fullInput string, modelCfg *loader.ModelConfig) (*genai.GenerateContentResponse, float64, error) {
	log.Printf("Attempting retry with secondary model: %s", modelCfg.SecondaryModelName)
	secondaryModel := c.genaiClient.GenerativeModel(modelCfg.SecondaryModelName)

	startSecondary := time.Now()
	resp, err := secondaryModel.GenerateContent(ctx, genai.Text(fullInput))
	elapsed := float64(time.Since(startSecondary).Milliseconds())

	if err == nil {
		log.Printf("Successfully generated content with secondary model: %s", modelCfg.SecondaryModelName)
	} else {
		errorLogger.Printf("Secondary Gemini API call failed for model %s: %v", modelCfg.SecondaryModelName, err)
	}
	return resp, elapsed, err
}

func (c *Chat) processGeminiResponse(ctx context.Context, userID, threadID, message string, modelCfg *loader.ModelConfig, resp *genai.GenerateContentResponse, start time.Time, elapsed float64) (string, float64, string, error) {
	if resp.Candidates == nil || len(resp.Candidates) == 0 {
		errorLogger.Println("Gemini response candidates are empty.")
		return "応答を取得できませんでした。", elapsed, modelCfg.ModelName, nil
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		errorLogger.Println("Gemini response candidate content or parts are empty.")
		return "応答を取得できませんでした。", elapsed, modelCfg.ModelName, nil
	}

	var functionCallProcessed bool
	var llmIntroText strings.Builder
	var toolResult string

	for i, part := range candidate.Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			llmIntroText.WriteString(string(v))
		case genai.FunctionCall:
			functionCallProcessed = true
			errorLogger.Printf("Unknown function call: %s", v.Name)
			toolResult = fmt.Sprintf("不明な関数呼び出し: %s", v.Name)
		default:
			errorLogger.Printf("Part %d is an unexpected type: %T", i, v)
		}
	}

	if functionCallProcessed {
		return c.handleFunctionCall(ctx, userID, threadID, message, modelCfg, candidate, toolResult, llmIntroText, start, elapsed)
	}

	responseText := llmIntroText.String()
	if responseText == "" {
		responseText = getResponseText(resp)
	}

	if responseText != "" {
		c.historyMgr.Add(userID, threadID, message, responseText)
	} else {
		errorLogger.Printf("Skipping history add for user %s in thread %s because responseText is empty.", userID, threadID)
	}
	return responseText, elapsed, modelCfg.ModelName, nil
}

func (c *Chat) handleFunctionCall(ctx context.Context, userID, threadID, message string, modelCfg *loader.ModelConfig, candidate *genai.Candidate, toolResult string, llmIntroText strings.Builder, start time.Time, elapsed float64) (string, float64, string, error) {
	var calledFuncName string
	for _, part := range candidate.Content.Parts {
		if fc, ok := part.(genai.FunctionCall); ok {
			calledFuncName = fc.Name
			break
		}
	}

	if calledFuncName == "" {
		errorLogger.Printf("Could not determine called function name from candidate parts.")
		return "関数呼び出し名の取得に失敗しました。", elapsed, modelCfg.ModelName, fmt.Errorf("関数呼び出し名の取得に失敗")
	}

	var partsForNextTurn []genai.Part
	partsForNextTurn = append(partsForNextTurn, genai.Text(message))

	var functionCallPart genai.FunctionCall
	for _, part := range candidate.Content.Parts {
		if fc, ok := part.(genai.FunctionCall); ok {
			functionCallPart = fc
			break
		}
	}

	if functionCallPart.Name == "" {
		errorLogger.Printf("Could not extract FunctionCall from candidate.Content.Parts for second call. Using full candidate.Content.Parts.")
		partsForNextTurn = append(partsForNextTurn, candidate.Content.Parts...)
	} else {
		if llmIntroText.Len() > 0 {
			partsForNextTurn = append(partsForNextTurn, genai.Text(llmIntroText.String()))
		}
		partsForNextTurn = append(partsForNextTurn, functionCallPart)
	}

	const maxToolResultForLLM = 1800
	toolResultForLLM := toolResult
	if len(toolResultForLLM) > maxToolResultForLLM {
		toolResultForLLM = toolResultForLLM[:maxToolResultForLLM] + "..."
	}

	partsForNextTurn = append(partsForNextTurn, genai.FunctionResponse{
		Name: calledFuncName,
		Response: map[string]interface{}{
			"content": toolResultForLLM,
		},
	})

	secondResp, err := c.genaiModel.GenerateContent(ctx, partsForNextTurn...)
	elapsed += float64(time.Since(start).Milliseconds())

	if err != nil {
		errorLogger.Printf("Error in second GenerateContent call after function execution: %v", err)
		return fmt.Sprintf("ツールの実行結果: %s (LLMによる最終応答生成に失敗: %v)", toolResult, err), elapsed, modelCfg.ModelName, nil
	}

	finalResponseText := getResponseText(secondResp)
	if finalResponseText == "" {
		finalResponseText = "ツールは実行されましたが、LLMからの追加の応答はありませんでした。"
		if toolResult != "" {
			finalResponseText += fmt.Sprintf(" ツールの結果: %s", toolResult)
		}
	}

	if finalResponseText != "" {
		c.historyMgr.Add(userID, threadID, message, finalResponseText)
	} else {
		errorLogger.Printf("Skipping history add for user %s in thread %s because finalResponseText is empty after function call.", userID, threadID)
	}
	return finalResponseText, elapsed, modelCfg.ModelName, nil
}

func (c *Chat) Close() {
	c.genaiClient.Close()
}

func GetErrorLogger() *log.Logger {
	return errorLogger
}
