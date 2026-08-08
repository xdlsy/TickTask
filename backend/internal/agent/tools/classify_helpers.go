package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// classifyResult is the parsed JSON returned by the LLM for a classify_task
// call. The Quadrant field is recomputed locally from Important/Urgent so we
// trust the model's booleans over any quadrant it returns.
type classifyResult struct {
	Important bool   `json:"important"`
	Urgent    bool   `json:"urgent"`
	Reason    string `json:"reason"`
}

// parseClassifyResponse extracts the JSON object the LLM embedded in its reply
// and unmarshals it into a classifyResult. LLMs sometimes wrap JSON in prose
// or code fences; extractJSON finds the outermost {...} block.
//
// Ported from service/ai_service.go (parseClassifyResponse) so the tools
// package can stand alone once AIService is removed in Task 15.
func parseClassifyResponse(response string) (*classifyResult, error) {
	jsonStr := extractJSON(response)
	var r classifyResult
	if err := json.Unmarshal([]byte(jsonStr), &r); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}
	return &r, nil
}

// extractJSON returns the substring of response starting at the first '{' and
// ending at the last '}'. If neither brace is present the response is returned
// unchanged (and the caller's json.Unmarshal will surface the parse error).
func extractJSON(response string) string {
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || start > end {
		return response
	}
	return response[start : end+1]
}
