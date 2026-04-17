package speech

import (
	"audiotranscrib/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Client struct {
	baseURL string
	token   string
	client  *http.Client
	logger  *zap.Logger

	accessToken string
	expiresAt   time.Time

	mu sync.Mutex
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type CreateTaskRequest struct {
	RequestFileID string  `json:"request_file_id"`
	Options       Options `json:"options"`
}

type Options struct {
	Model                    string                   `json:"model"`
	AudioEncoding            string                   `json:"audio_encoding"`
	SampleRate               int                      `json:"sample_rate"`
	Language                 string                   `json:"language"`
	EnableProfanityFilter    bool                     `json:"enable_profanity_filter"`
	HypothesesCount          int                      `json:"hypotheses_count"`
	NoSpeechTimeout          string                   `json:"no_speech_timeout"`
	MaxSpeechTimeout         string                   `json:"max_speech_timeout"`
	Hints                    Hints                    `json:"hints"`
	ChannelsCount            int                      `json:"channels_count"`
	SpeakerSeparationOptions SpeakerSeparationOptions `json:"speaker_separation_options"`
	InsightModels            []string                 `json:"insight_models"`
}

type Hints struct {
	Words         []string `json:"words"`
	EnableLetters bool     `json:"enable_letters"`
	EOUTimeout    string   `json:"eou_timeout"`
}

type SpeakerSeparationOptions struct {
	Enable                bool `json:"enable"`
	EnableOnlyMainSpeaker bool `json:"enable_only_main_speaker"`
	Count                 int  `json:"count"`
}

func NewClient(cfg *config.Config, logger *zap.Logger) *Client {
	return &Client{
		baseURL: "https://smartspeech.sber.ru/rest/v1",
		token:   cfg.SaluteSpeechKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func detectAudioParams(mime string) (string, int) {
	mime = strings.ToLower(mime)

	switch {
	case strings.Contains(mime, "ogg"):
		return "OPUS", 16000
	case strings.Contains(mime, "mpeg"):
		return "MP3", 44100
	case strings.Contains(mime, "wav"):
		return "PCM_S16LE", 16000
	default:
		return "PCM_S16LE", 16000
	}
}

func (c *Client) uploadFile(ctx context.Context, data []byte, mime string) (string, error) {
	start := time.Now()

	contentType := "application/octet-stream"

	if strings.Contains(mime, "ogg") {
		contentType = "audio/ogg;codecs=opus"
	} else if strings.Contains(mime, "mpeg") {
		contentType = "audio/mpeg"
	} else if strings.Contains(mime, "wav") {
		contentType = "audio/wav"
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/data:upload",
		bytes.NewReader(data),
	)
	if err != nil {
		return "", fmt.Errorf("create upload request: %w", err)
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	c.logger.Info("upload audio",
		zap.Int("size_bytes", len(data)),
		zap.String("content_type", contentType),
		zap.Duration("took", time.Since(start)),
	)

	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("upload failed",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("body", body),
		)
		return "", fmt.Errorf("upload failed: %d", resp.StatusCode)
	}

	var result struct {
		Result struct {
			RequestFileID string `json:"request_file_id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}

	return result.Result.RequestFileID, nil
}

func (c *Client) createTask(ctx context.Context, fileID string, encoding string, sampleRate int) (string, error) {
	reqBody := CreateTaskRequest{
		RequestFileID: fileID,
		Options: Options{
			Model:                 "general",
			AudioEncoding:         encoding,
			SampleRate:            sampleRate,
			Language:              "ru-RU",
			EnableProfanityFilter: true,
			HypothesesCount:       1,
			NoSpeechTimeout:       "2s",
			MaxSpeechTimeout:      "2s",
			Hints: Hints{
				Words:         []string{"карту", "гуакамоле"},
				EnableLetters: false,
				EOUTimeout:    "1s",
			},
			ChannelsCount: 1,
			SpeakerSeparationOptions: SpeakerSeparationOptions{
				Enable:                false,
				EnableOnlyMainSpeaker: false,
				Count:                 1,
			},
			InsightModels: []string{"csi", "call_features"},
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal createTask: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/speech:async_recognize",
		bytes.NewReader(b),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := c.client.Do(req)
	c.logger.Info("create task",
		zap.String("encoding", encoding),
		zap.Int("sample_rate", sampleRate),
		zap.Duration("took", time.Since(start)),
	)

	if err != nil {
		return "", fmt.Errorf("create task request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("create task failed",
			zap.Int("status", resp.StatusCode),
			zap.ByteString("body", body),
		)
		return "", fmt.Errorf("create task failed: %d", resp.StatusCode)
	}

	var result struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode createTask response: %w", err)
	}

	return result.Result.ID, nil
}

func (c *Client) waitResult(ctx context.Context, taskID string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return "", fmt.Errorf("recognition timeout")

		case <-ticker.C:
			req, err := http.NewRequestWithContext(timeoutCtx, "GET", c.baseURL+"/task:get?id="+taskID, nil)
			if err != nil {
				return "", fmt.Errorf("create status request: %w", err)
			}

			token, err := c.getToken(timeoutCtx)
			if err != nil {
				return "", err
			}

			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/octet-stream")

			resp, err := c.client.Do(req)
			if err != nil {
				return "", fmt.Errorf("status request failed: %w", err)
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var tmp struct {
				Result struct {
					Status         string `json:"status"`
					ResponseFileID string `json:"response_file_id"`
				} `json:"result"`
			}

			if err := json.Unmarshal(body, &tmp); err != nil {
				c.logger.Warn("failed to decode status", zap.Error(err))
				continue
			}

			switch tmp.Result.Status {
			case "NEW", "RUNNING":
				continue

			case "DONE":
				if tmp.Result.ResponseFileID == "" {
					c.logger.Warn("empty response_file_id")
					continue
				}

				data, err := c.downloadResult(timeoutCtx, tmp.Result.ResponseFileID)
				if err != nil {
					return "", err
				}

				var result struct {
					Result []struct {
						Results []struct {
							NormalizedText string `json:"normalized_text"`
						} `json:"results"`
					} `json:"result"`
				}

				if err := json.Unmarshal(data, &result); err != nil {
					return "", fmt.Errorf("decode final result: %w", err)
				}

				var texts []string
				for _, r := range result.Result {
					for _, res := range r.Results {
						texts = append(texts, res.NormalizedText)
					}
				}

				if len(texts) == 0 {
					return "[не удалось распознать речь]", nil
				}

				return strings.Join(texts, " "), nil

			case "ERROR", "CANCELED":
				return "", fmt.Errorf("recognition failed: %s", tmp.Result.Status)
			}
		}
	}
}

func (c *Client) downloadResult(ctx context.Context, responseFileID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		c.baseURL+"/data:download?response_file_id="+responseFileID,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/octet-stream")

	start := time.Now()
	resp, err := c.client.Do(req)
	c.logger.Info("download result", zap.Duration("took", time.Since(start)))

	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		strings.NewReader("scope=SALUTE_SPEECH_PERS"),
	)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", uuid.New().String())
	req.Header.Set("Authorization", "Basic "+c.token)

	start := time.Now()
	resp, err := c.client.Do(req)
	c.logger.Info("speech token request", zap.Duration("took", time.Since(start)))

	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}

	c.accessToken = result.AccessToken
	c.expiresAt = time.Unix(result.ExpiresAt, 0).Add(-time.Minute)

	c.logger.Info("speech token received",
		zap.Time("expires_at", c.expiresAt),
	)

	return c.accessToken, nil
}

func (c *Client) Recognize(ctx context.Context, data []byte, mime string) (string, error) {
	encoding, sampleRate := detectAudioParams(mime)

	c.logger.Info("audio params",
		zap.String("mime", mime),
		zap.String("encoding", encoding),
		zap.Int("sample_rate", sampleRate),
	)

	fileID, err := c.uploadFile(ctx, data, mime)
	if err != nil {
		return "", err
	}

	taskID, err := c.createTask(ctx, fileID, encoding, sampleRate)
	if err != nil {
		return "", err
	}

	return c.waitResult(ctx, taskID)
}
