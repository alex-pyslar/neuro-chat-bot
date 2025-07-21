package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alex-pyslar/neuro-chat-bot/internal/domain"
	"github.com/alex-pyslar/neuro-chat-bot/internal/usecases"
	"github.com/alex-pyslar/neuro-chat-bot/pkg/logger"
)

// ChatCompletionMessage представляет сообщение в запросе к API завершения чата.
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionChoice представляет выбор ответа от модели.
type ChatCompletionChoice struct {
	Message ChatCompletionMessage `json:"message"`
}

// ChatCompletionResponse представляет ответ от API завершения чата.
type ChatCompletionResponse struct {
	Choices []ChatCompletionChoice `json:"choices"`
}

// LlamaCppGateway является реализацией usecases.ModelGateway для взаимодействия с llama-server.
type LlamaCppGateway struct {
	httpClient *http.Client
	logger     logger.Logger
	baseURL    string // Базовый URL для llama-server
}

// NewLlamaCppGateway создает новый экземпляр LlamaCppGateway.
func NewLlamaCppGateway(baseURL string, logger logger.Logger, timeout time.Duration) *LlamaCppGateway {
	return &LlamaCppGateway{
		httpClient: &http.Client{Timeout: timeout},
		logger:     logger,
		baseURL:    baseURL,
	}
}

// GetModelResponse отправляет запрос к llama-server и возвращает ответ модели.
func (g *LlamaCppGateway) GetModelResponse(ctx context.Context, messages []domain.ChatMessage, config usecases.ModelConfig) (string, error) {
	// Преобразуем domain.ChatMessage в ChatCompletionMessage для запроса
	apiMessages := make([]ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	requestBody := map[string]interface{}{
		"messages":       apiMessages,
		"temperature":    config.Temperature,
		"top_p":          config.TopP,
		"top_k":          config.TopK,
		"max_tokens":     config.MaxTokens,
		"repeat_penalty": config.RepeatPenalty,
		// "min_p": config.MinP, // Llama.cpp doesn't directly support min_p in this API
		// "presence_penalty": config.PresencePenalty, // Not directly supported
		// "frequency_penalty": config.FrequencyPenalty, // Not directly supported
		// "stop": config.StopSequences, // Uncomment if you add stop sequences to ModelConfig
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		g.logger.Error("Failed to marshal request body: %v", err)
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		g.logger.Error("Failed to create HTTP request: %v", err)
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		g.logger.Error("HTTP Request Error to Llama-server: %v", err)
		return "", fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		g.logger.Error("Llama-server returned non-OK status code: %d, Body: %s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("llama-server returned non-OK status code: %d", resp.StatusCode)
	}

	var result ChatCompletionResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		g.logger.Error("Failed to decode Llama-server response: %v", err)
		return "", fmt.Errorf("failed to decode Llama-server response: %w", err)
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response choices from Llama-server")
}

// Verify that LlamaCppGateway implements usecases.ModelGateway
var _ usecases.ModelGateway = (*LlamaCppGateway)(nil)
