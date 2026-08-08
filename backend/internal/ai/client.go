package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"ticktask/internal/model"
)

// ErrFunctionCallNotSupported indicates the LLM provider does not support
// function/tool calling (e.g. CLI-based providers, or stubs awaiting impl).
var ErrFunctionCallNotSupported = errors.New("function calling not supported by this provider")

// LLMClient LLM 客户端接口
type LLMClient interface {
	ChatCompletion(ctx context.Context, prompt string) (string, error)
	ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error)
}

// Message is the unified chat message envelope used by both ChatCompletion
// (single user-prompt convenience wrapper) and ChatWithTools (full
// multi-turn, tool-calling form). The ToolCalls/ToolCallID/Name fields are
// omitted when empty so existing ChatCompletion JSON payloads are unchanged.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall is a normalized tool/function invocation produced by the model.
// Args is the raw JSON arguments string from the provider, kept as RawMessage
// so callers can unmarshal into whatever shape they expect.
type ToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// FunctionSpec describes a single function tool's schema.
type FunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolSpec wraps a FunctionSpec with the provider-required "type":"function"
// discriminator. Matches the OpenAI tool-call request shape.
type ToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

// ToolResponse is the normalized result of a ChatWithTools call: the model's
// textual content (possibly empty when the model only emits tool calls), the
// parsed tool calls, and the provider's termination reason.
type ToolResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls"`
	FinishReason string     `json:"finish_reason"`
}

// OpenAIClient OpenAI 客户端实现
type OpenAIClient struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func NewOpenAIClient(apiKey, baseURL, model string) LLMClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &OpenAIClient{
		client:  &http.Client{},
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

// AnthropicClient Anthropic Claude 客户端
type AnthropicClient struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
}

type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type AnthropicContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// CLIClient 通过 claude CLI 调用
type CLIClient struct {
	Binary string
}

func NewCLIClient() LLMClient {
	return &CLIClient{}
}

func (c *CLIClient) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude CLI error: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func NewAnthropicClient(apiKey, baseURL, model string) LLMClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicClient{
		client:  &http.Client{},
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

func (c *AnthropicClient) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}

	req := AnthropicRequest{
		Model:     c.model,
		MaxTokens: 2048,
		Messages: []AnthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp AnthropicResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp AnthropicResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Content) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return chatResp.Content[0].Text, nil
}

func (c *OpenAIClient) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}

	req := ChatRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ChatResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// === Function-calling extension: ChatWithTools implementations ===

// ChatWithTools sends a multi-turn conversation with optional tool/function
// definitions to an OpenAI-compatible chat endpoint. It retries up to 3 times
// on transient network errors or HTTP 5xx responses using exponential backoff
// (1s, 2s, 4s). Non-5xx errors and successful parses are returned immediately.
func (c *OpenAIClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error) {
	body := map[string]any{
		"model":    c.model,
		"messages": messages,
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
		if err != nil {
			return ToolResponse{}, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
			}
			continue
		}

		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
			}
			continue
		}
		if resp.StatusCode != 200 {
			return ToolResponse{}, fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
		}

		var parsed struct {
			Choices []struct {
				Message struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return ToolResponse{}, fmt.Errorf("decode: %w", err)
		}
		if len(parsed.Choices) == 0 {
			return ToolResponse{}, errors.New("no choices")
		}
		ch := parsed.Choices[0]
		out := ToolResponse{
			Content:      ch.Message.Content,
			FinishReason: ch.FinishReason,
		}
		for _, tc := range ch.Message.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: json.RawMessage(tc.Function.Arguments),
			})
		}
		return out, nil
	}
	if lastErr == nil {
		lastErr = errors.New("openai: exhausted retries")
	}
	return ToolResponse{}, lastErr
}

// ChatWithTools on AnthropicClient maps the unified OpenAI-style request to
// the Anthropic Messages API and back. Mapping notes:
//
//   - System message is extracted from the message list and sent as the
//     top-level `system` field (Anthropic separates system from the
//     conversational `messages` array).
//   - Tool definitions are translated to Anthropic's `tools[]` shape, where
//     each tool has `name`, `description`, and `input_schema` (mirroring the
//     OpenAI FunctionSpec.Parameters schema object).
//   - Response content blocks are flattened: `text` blocks concatenate into
//     Content, `tool_use` blocks become ToolCall entries with their `input`
//     re-serialized as the Args JSON. `stop_reason="tool_use"` maps to the
//     OpenAI-style `"tool_calls"` finish reason.
//
// Retries mirror OpenAIClient: up to 3 attempts with exponential backoff on
// transient network errors or HTTP 5xx.
func (c *AnthropicClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error) {
	system, userMsgs := splitSystemMessage(messages)

	body := map[string]any{
		"model":       c.model,
		"max_tokens":  1024,
		"messages":    userMsgs,
		"system":      system,
	}
	if len(tools) > 0 {
		body["tools"] = convertToolsToAnthropic(tools)
		body["tool_choice"] = map[string]string{"type": "auto"}
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return ToolResponse{}, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
			}
			continue
		}

		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(raw))
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
			}
			continue
		}
		if resp.StatusCode != 200 {
			return ToolResponse{}, fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(raw))
		}

		var parsed anthropicMessagesResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return ToolResponse{}, fmt.Errorf("decode: %w", err)
		}

		out := ToolResponse{FinishReason: mapStopReason(parsed.StopReason)}
		for _, blk := range parsed.Content {
			switch blk.Type {
			case "text":
				out.Content += blk.Text
			case "tool_use":
				argsBytes, _ := json.Marshal(blk.Input)
				out.ToolCalls = append(out.ToolCalls, ToolCall{
					ID:   blk.ID,
					Name: blk.Name,
					Args: argsBytes,
				})
			}
		}
		return out, nil
	}
	if lastErr == nil {
		lastErr = errors.New("anthropic: exhausted retries")
	}
	return ToolResponse{}, lastErr
}

// splitSystemMessage separates leading system messages from the conversation.
// Anthropic's Messages API requires the system prompt as a top-level field
// rather than as an entry in `messages`. The unified Message shape may carry
// the system prompt as the first message with Role="system"; we pull it out
// and concatenate any additional system messages by newline.
func splitSystemMessage(messages []Message) (system string, rest []Message) {
	var parts []string
	for _, m := range messages {
		if m.Role == "system" {
			if m.Content != "" {
				parts = append(parts, m.Content)
			}
		} else {
			rest = append(rest, m)
		}
	}
	return strings.Join(parts, "\n\n"), rest
}

// convertToolsToAnthropic translates the OpenAI-style ToolSpec list into
// Anthropic's `tools[]` shape, where each entry carries `input_schema`
// mirroring the OpenAI FunctionSpec.Parameters JSON-schema object.
func convertToolsToAnthropic(tools []ToolSpec) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		entry := map[string]any{
			"name":         t.Function.Name,
			"input_schema": t.Function.Parameters,
		}
		if t.Function.Description != "" {
			entry["description"] = t.Function.Description
		}
		out = append(out, entry)
	}
	return out
}

// mapStopReason converts Anthropic's stop_reason to the OpenAI-style
// FinishReason vocabulary used by the unified ToolResponse. The agent service
// only branches on "no tool calls" vs "has tool calls", so the precise token
// is informational rather than load-bearing.
func mapStopReason(reason string) string {
	switch reason {
	case "tool_use":
		return "tool_calls"
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return reason
	}
}

// anthropicMessagesResponse is the parsed shape of the Anthropic Messages API
// response. Content blocks are polymorphic (text / tool_use / etc.); we model
// only the fields the agent path consumes and leave others as zero values.
type anthropicMessagesResponse struct {
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
}

// anthropicContentBlock models one entry of the Content array. Input is kept
// as a generic map[string]any and re-serialized to JSON when constructing the
// unified ToolCall.Args payload.
type anthropicContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

// ChatWithTools on CLIClient is permanently unsupported: the CLI provider
// surface does not expose structured tool calls. Returns ErrFunctionCallNotSupported.
func (c CLIClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error) {
	return ToolResponse{}, ErrFunctionCallNotSupported
}

// protocolFor maps a vendor id (AISettings.Provider) to one of the three
// implemented client protocols. Known vendors resolve to their protocol;
// unknown values — including "custom" and the empty string — fall back to
// the OpenAI-compatible client, matching historical behavior so existing
// stored settings keep working without migration.
//
// The frontend vendor-preset registry
// (frontend/src/utils/aiVendors.ts) is the display source of truth for vendor
// labels, default base URLs, and model presets; this function is the backend
// routing counterpart and must stay in sync with the preset list.
func protocolFor(provider string) string {
	switch provider {
	case "claude", "cli":
		return "cli"
	case "anthropic", "minimax":
		return "anthropic"
	default: // openai, deepseek, qwen, zhipu, moonshot, custom, "" → OpenAI-compatible
		return "openai"
	}
}

// NewClientFromSettings constructs the appropriate LLMClient for a given
// AISettings, routing by the vendor's protocol (see protocolFor). Returns nil
// if the provider needs an API key and none is set, so callers can distinguish
// "nothing configured" from "configured but rejected" by checking for nil.
//
// Extracted from cmd/server/main.go's constructLLMClient so other packages
// (notably the agent service's TestConnection temp-settings path) can build
// one-shot clients without import cycles through main. main.go's
// constructLLMClient now delegates to this function.
func NewClientFromSettings(settings *model.AISettings) LLMClient {
	if settings == nil {
		return nil
	}
	switch protocolFor(settings.Provider) {
	case "cli":
		return NewCLIClient()
	case "anthropic":
		if settings.APIKey == "" {
			return nil
		}
		return NewAnthropicClient(settings.APIKey, settings.BaseURL, settings.Model)
	default: // openai
		if settings.APIKey == "" {
			return nil
		}
		return NewOpenAIClient(settings.APIKey, settings.BaseURL, settings.Model)
	}
}
