package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	defaultTimeout   = 60 * time.Second
	defaultMaxTokens = 8192
)

// AnthropicClient calls the Anthropic Messages API.
// It implements both Client and Caller interfaces.
type AnthropicClient struct {
	apiKey string
	model  string
	http   *http.Client
}

// NewAnthropicClient creates a client with a 60-second timeout.
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: defaultTimeout},
	}
}

// Call sends raw system and user prompts and returns the text response.
func (c *AnthropicClient) Call(systemPrompt, userPrompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":      c.model,
		"max_tokens": defaultMaxTokens,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("anthropic: create request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("anthropic: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("anthropic: API error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return "", fmt.Errorf("anthropic: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", fmt.Errorf("anthropic: parse response: %w", err)
	}
	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response content")
	}
	return anthropicResp.Content[0].Text, nil
}

// Generate sends a GenerateRequest to Claude and returns parsed file contents.
func (c *AnthropicClient) Generate(req GenerateRequest) (*GenerateResponse, error) {
	rawText, err := c.Call(BuildSystemPrompt(), BuildUserPrompt(req))
	if err != nil {
		return nil, err
	}
	return ParseResponse(rawText)
}
