package agent

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ValidateArgs performs a lightweight JSON-schema validation of the raw args
// produced by the LLM against the tool's FunctionSpec.Parameters. It checks:
//   - args is a JSON object (when schema type == "object")
//   - every user-supplied field exists in schema["properties"] and matches the
//     declared type (lenient: tools that declare type=="object" without any
//     properties are skipped, so trivially-typed tools still pass)
//   - every name in schema["required"] is present
//
// It is intentionally permissive about anything it does not understand —
// additional JSON-schema features (oneOf, pattern, etc.) are ignored. The
// goal is to catch obvious LLM mistakes (wrong type, unknown field, missing
// required arg) before the tool is invoked, not to be a fully conformant
// JSON-schema implementation.
func ValidateArgs(schema map[string]any, args json.RawMessage) error {
	if len(schema) == 0 {
		return nil
	}
	var got map[string]any
	if err := json.Unmarshal(args, &got); err != nil {
		return fmt.Errorf("args not a json object: %w", err)
	}
	if t, ok := schema["type"].(string); ok && t == "object" {
		props, _ := schema["properties"].(map[string]any)
		// Lenient: if a tool declares type=="object" but lists no properties,
		// skip field-level validation entirely. This keeps trivially-typed
		// tools (e.g. test fakes that accept anything) usable; real tools that
		// want strict checking declare their properties.
		if len(props) == 0 {
			return nil
		}
		for field, fval := range got {
			ps, ok := props[field].(map[string]any)
			if !ok {
				return errors.New("schema: unknown field " + field)
			}
			wantType, _ := ps["type"].(string)
			if err := checkType(wantType, fval); err != nil {
				return fmt.Errorf("schema: field %s: %w", field, err)
			}
		}
		req, _ := schema["required"].([]any)
		for _, r := range req {
			if rs, ok := r.(string); ok {
				if _, present := got[rs]; !present {
					return errors.New("schema: missing required field " + rs)
				}
			}
		}
	}
	return nil
}

// checkType compares a JSON-unmarshaled value against a JSON-schema type name.
// An empty wantType means "no type declared" — accept anything.
func checkType(want string, v any) error {
	switch want {
	case "string":
		if _, ok := v.(string); !ok {
			return errors.New("expected string")
		}
	case "number", "integer":
		if _, ok := v.(float64); !ok {
			return errors.New("expected number")
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return errors.New("expected boolean")
		}
	case "object":
		if _, ok := v.(map[string]any); !ok {
			return errors.New("expected object")
		}
	case "array":
		if _, ok := v.([]any); !ok {
			return errors.New("expected array")
		}
	}
	return nil
}
