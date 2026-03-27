package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/ai"
	"testing"
	"time"
)

func TestAIService_IsConfigured(t *testing.T) {
	// Without client
	service := &AIService{client: nil}
	if service.IsConfigured() {
		t.Error("expected IsConfigured to be false when client is nil")
	}

	// With client
	service = &AIService{client: ai.NewOpenAIClient("test-key", "", "")}
	if !service.IsConfigured() {
		t.Error("expected IsConfigured to be true when client is set")
	}
}

func TestAIService_CalculateQuadrant(t *testing.T) {
	tests := []struct {
		important bool
		urgent    bool
		expected  int
	}{
		{true, true, 1},
		{true, false, 2},
		{false, true, 3},
		{false, false, 4},
	}

	for _, tt := range tests {
		result := calculateQuadrant(tt.important, tt.urgent)
		if result != tt.expected {
			t.Errorf("calculateQuadrant(%v, %v) = %d, expected %d",
				tt.important, tt.urgent, result, tt.expected)
		}
	}
}

func TestAIService_ParseClassifyResponse(t *testing.T) {
	jsonResponse := `{"important": true, "urgent": false, "reason": "Test reason"}`

	result, err := parseClassifyResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseClassifyResponse failed: %v", err)
	}

	if result.Important != true {
		t.Errorf("expected Important to be true, got %v", result.Important)
	}

	if result.Urgent != false {
		t.Errorf("expected Urgent to be false, got %v", result.Urgent)
	}

	if result.Reason != "Test reason" {
		t.Errorf("expected Reason 'Test reason', got %s", result.Reason)
	}
}

func TestAIService_ParseClassifyResponse_WithExtraContent(t *testing.T) {
	// Test with markdown code blocks
	jsonResponse := "```json\n{\"important\": true, \"urgent\": true, \"reason\": \"Test\"}\n```"

	result, err := parseClassifyResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseClassifyResponse failed: %v", err)
	}

	if result.Important != true {
		t.Errorf("expected Important to be true")
	}
}

func TestAIService_ParseScheduleResponse(t *testing.T) {
	jsonResponse := `{
		"schedule": [
			{
				"task_id": "task-1",
				"title": "Task 1",
				"start_time": "09:00",
				"end_time": "10:30",
				"pomodoro_count": 3
			}
		]
	}`

	result, err := parseScheduleResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseScheduleResponse failed: %v", err)
	}

	if len(result.Schedule) != 1 {
		t.Fatalf("expected 1 schedule item, got %d", len(result.Schedule))
	}

	if result.Schedule[0].TaskID != "task-1" {
		t.Errorf("expected task_id 'task-1', got %s", result.Schedule[0].TaskID)
	}

	if result.Schedule[0].PomodoroCount != 3 {
		t.Errorf("expected pomodoro_count 3, got %d", result.Schedule[0].PomodoroCount)
	}
}

func TestAIService_ParsePriorityResponse(t *testing.T) {
	jsonResponse := `{"priority_order": ["task-1", "task-2", "task-3"]}`

	result, err := parsePriorityResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parsePriorityResponse failed: %v", err)
	}

	if len(result.PriorityOrder) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.PriorityOrder))
	}

	if result.PriorityOrder[0] != "task-1" {
		t.Errorf("expected first item 'task-1', got %s", result.PriorityOrder[0])
	}
}

func TestAIService_ExtractJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			input:    `Here is the result: {"key": "value"} and more text`,
			expected: `{"key": "value"}`,
		},
		{
			input:    `Some text before {"a": 1, "b": 2} some text after`,
			expected: `{"a": 1, "b": 2}`,
		},
	}

	for _, tt := range tests {
		result := extractJSON(tt.input)
		if result != tt.expected {
			t.Errorf("extractJSON(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// Integration test with mock HTTP server
func TestAIService_ChatCompletion(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong Authorization header")
		}

		// Return mock response
		resp := ai.ChatResponse{
			Choices: []ai.Choice{
				{
					Message: ai.Message{
						Role:    "assistant",
						Content: `{"important": true, "urgent": false, "reason": "Test"}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with mock server URL
	client := ai.NewOpenAIClient("test-key", server.URL, "gpt-4o-mini")

	// Make request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.ChatCompletion(ctx, "Test prompt")
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if response == "" {
		t.Error("expected non-empty response")
	}
}

func TestAIService_ChatCompletion_Error(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := ai.ChatResponse{
			Error: &ai.APIError{
				Message: "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ai.NewOpenAIClient("invalid-key", server.URL, "gpt-4o-mini")

	ctx := context.Background()
	_, err := client.ChatCompletion(ctx, "Test prompt")
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestAIService_ChatCompletion_Timeout(t *testing.T) {
	// Create mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := ai.NewOpenAIClient("test-key", server.URL, "gpt-4o-mini")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.ChatCompletion(ctx, "Test prompt")
	if err == nil {
		t.Error("expected timeout error")
	}
}