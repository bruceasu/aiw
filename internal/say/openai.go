package say

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// OpenAIProvider calls the Chat Completions API without storing request content.
type OpenAIProvider struct {
	apiKey   string
	endpoint string
	client   httpDoer
	wait     func(context.Context, time.Duration) error
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: os.Getenv("OPENAI_API_KEY"), endpoint: configuredEndpoint(),
		client: &http.Client{}, wait: waitContext,
	}
}

func configuredEndpoint() string {
	base := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if base == "" { return openAIEndpoint }
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return ""
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path == "" { parsed.Path = "/v1" }
	if !strings.HasSuffix(parsed.Path, "/chat/completions") { parsed.Path += "/chat/completions" }
	return parsed.String()
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") { return true }
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (p *OpenAIProvider) Translate(ctx context.Context, request Request) (string, error) {
	if p.client == nil { p.client = &http.Client{} }
	if p.wait == nil { p.wait = waitContext }
	if strings.TrimSpace(p.apiKey) == "" {
		return "", errors.New("OPENAI_API_KEY is required")
	}
	if strings.TrimSpace(request.Model) == "" {
		return "", errors.New("model is required; set [say.llm].model or --model")
	}
	if p.endpoint == "" {
		return "", errors.New("OPENAI_BASE_URL must be an HTTPS URL; HTTP is allowed only for loopback testing")
	}
	body, err := json.Marshal(map[string]any{
		"model": request.Model,
		"messages": []map[string]string{
			{"role": "system", "content": SystemPrompt(request)},
			{"role": "user", "content": request.Text},
		},
		"stream": false,
	})
	if err != nil {
		return "", errors.New("encode translation request")
	}
	for attempt := 0; attempt <= 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
		if err != nil {
			return "", errors.New("create OpenAI request")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
		response, err := p.client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return "", errors.New("translation request timed out or was canceled")
			}
			if attempt == 2 {
				return "", errors.New("OpenAI request failed after retries")
			}
			if waitErr := p.wait(ctx, retryDelay(attempt)); waitErr != nil {
				return "", errors.New("translation request timed out or was canceled")
			}
			continue
		}
		if shouldRetryStatus(response.StatusCode) && attempt < 2 {
			response.Body.Close()
			if waitErr := p.wait(ctx, retryDelay(attempt)); waitErr != nil {
				return "", errors.New("translation request timed out or was canceled")
			}
			continue
		}
		translation, err := decodeResponse(response)
		if err != nil {
			return "", err
		}
		return translation, nil
	}
	return "", errors.New("OpenAI request failed after retries")
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500 && status <= 599
}

func retryDelay(attempt int) time.Duration {
	return time.Duration(200<<attempt) * time.Millisecond
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func decodeResponse(response *http.Response) (string, error) {
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "", fmt.Errorf("OpenAI authentication failed (HTTP %d)", response.StatusCode)
		case http.StatusTooManyRequests:
			return "", errors.New("OpenAI rate limit reached after retries")
		default:
			return "", fmt.Errorf("OpenAI request failed (HTTP %d)", response.StatusCode)
		}
	}
	var result struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	limited := io.LimitReader(response.Body, 2*1024*1024+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", errors.New("read OpenAI response")
	}
	if len(data) > 2*1024*1024 {
		return "", errors.New("OpenAI response exceeds size limit")
	}
	if !utf8.Valid(data) || json.Unmarshal(data, &result) != nil {
		return "", errors.New("OpenAI returned an invalid response")
	}
	if len(result.Choices) != 1 || result.Choices[0].FinishReason != "stop" {
		return "", errors.New("OpenAI response was incomplete")
	}
	text := strings.TrimSpace(result.Choices[0].Message.Content)
	if text == "" || !utf8.ValidString(text) {
		return "", errors.New("OpenAI response contains no valid translation")
	}
	return text, nil
}
