package ai

import (
	"testing"

	"ticktask/internal/model"
)

func TestProtocolFor(t *testing.T) {
	cases := []struct {
		provider string
		want     string
	}{
		{"openai", "openai"},
		{"deepseek", "openai"},
		{"qwen", "openai"},
		{"zhipu", "openai"},
		{"moonshot", "openai"},
		{"custom", "openai"},
		{"", "openai"},
		{"anthropic", "anthropic"},
		{"minimax", "anthropic"},
		{"claude", "cli"},
		{"cli", "cli"},
	}
	for _, c := range cases {
		if got := protocolFor(c.provider); got != c.want {
			t.Errorf("protocolFor(%q) = %q, want %q", c.provider, got, c.want)
		}
	}
}

func TestNewClientFromSettings_Routing(t *testing.T) {
	if got := NewClientFromSettings(nil); got != nil {
		t.Fatalf("nil settings: got %v, want nil", got)
	}

	cases := []struct {
		name     string
		provider string
		apiKey   string
		want     string // "openai" | "anthropic" | "cli" | "nil"
	}{
		{"openai-compatible vendor routes to OpenAIClient", "deepseek", "k", "openai"},
		{"anthropic-compatible vendor routes to AnthropicClient", "minimax", "k", "anthropic"},
		{"cli vendor routes to CLIClient", "claude", "k", "cli"},
		{"http provider with empty key returns nil", "openai", "", "nil"},
		{"anthropic provider with empty key returns nil", "minimax", "", "nil"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &model.AISettings{Provider: c.provider, APIKey: c.apiKey, BaseURL: "http://x", Model: "m"}
			got := NewClientFromSettings(s)
			switch c.want {
			case "openai":
				if _, ok := got.(*OpenAIClient); !ok {
					t.Fatalf("got %T, want *OpenAIClient", got)
				}
			case "anthropic":
				if _, ok := got.(*AnthropicClient); !ok {
					t.Fatalf("got %T, want *AnthropicClient", got)
				}
			case "cli":
				if _, ok := got.(*CLIClient); !ok {
					t.Fatalf("got %T, want *CLIClient", got)
				}
			case "nil":
				if got != nil {
					t.Fatalf("got %T, want nil", got)
				}
			}
		})
	}
}
