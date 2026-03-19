package ai

import (
	"audiotranscrib/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		SystemTokens     int `json:"system_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type GigaChatClient struct {
	authKey     string
	accessToken string
	expiresAt   time.Time
	client      *http.Client
	logger      *zap.Logger
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func NewGigaChatClient(cfg *config.Config, logger *zap.Logger) *GigaChatClient {
	return &GigaChatClient{
		authKey: cfg.GigaChatKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *GigaChatClient) getToken(ctx context.Context) (string, error) {
	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		strings.NewReader("scope=GIGACHAT_API_PERS"),
	)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", uuid.New().String())
	req.Header.Set("Authorization", "Basic "+c.authKey)

	start := time.Now()
	resp, err := c.client.Do(req)
	c.logger.Info("gigachat token request", zap.Duration("took", time.Since(start)))

	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("token request failed",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("body", body),
		)
		return "", fmt.Errorf("token request failed: %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	c.accessToken = tr.AccessToken
	c.expiresAt = time.UnixMilli(tr.ExpiresAt).Add(-time.Minute)

	c.logger.Info("token received",
		zap.Time("expires_at", c.expiresAt),
	)

	return c.accessToken, nil
}

func (c *GigaChatClient) GetSummary(ctx context.Context, text string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	reqBody := ChatRequest{
		Model: "GigaChat",
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "Сделай краткое резюме текста",
			},
			{
				Role:    "user",
				Content: text,
			},
		},
		Stream: false,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://gigachat.devices.sberbank.ru/api/v1/chat/completions",
		bytes.NewReader(b),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := c.client.Do(req)
	c.logger.Info("gigachat request", zap.Duration("took", time.Since(start)))

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("gigachat API error",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("body", body),
		)
		return "", fmt.Errorf("gigachat API error: %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		c.logger.Warn("empty response from gigachat",
			zap.ByteString("body", body),
		)
		return "", fmt.Errorf("empty response")
	}

	result := chatResp.Choices[0].Message.Content

	c.logger.Info("gigachat success",
		zap.String("content", result),
		zap.Int("total_tokens", chatResp.Usage.TotalTokens),
	)

	return result, nil
}
