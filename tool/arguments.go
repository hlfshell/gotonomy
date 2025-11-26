package tool

import (
	"encoding/json"
	"fmt"
)

// NewArguments creates an Arguments map from any value.
//
// Conversion rules:
// - If v is already Arguments or map[string]any, it is used directly (no JSON round-trip).
// - Otherwise, v is marshaled via encoding/json and then unmarshaled into a map.
//   Struct fields follow json tags and exported field names. Ensure tags/names match.
//
// This makes the JSON round-trip explicit and allows predictable struct-to-Arguments
// conversion, while preserving performance when callers already provide maps.
//
// Example:
//
//	type MyArgs struct {
//		Input    string `json:"input"`
//		Location string `json:"location"`
//	}
//
//	args := NewArguments(MyArgs{
//		Input:    "What's the weather?",
//		Location: "New York",
//	})
//
// Or from a map:
//
//	args := NewArguments(map[string]interface{}{
//		"input": "Hello",
//		"key":   "value",
//	})
func NewArguments(v interface{}) (Arguments, error) {
	if v == nil {
		return Arguments{}, nil
	}

	// If it's already an Arguments map, return it directly
	if args, ok := v.(Arguments); ok {
		return args, nil
	}

	// If it's already a map[string]interface{}, convert it
	if m, ok := v.(map[string]interface{}); ok {
		return Arguments(m), nil
	}

	// Marshal to JSON, then unmarshal into a map
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value to JSON: %w", err)
	}

	var args Arguments
	if err := json.Unmarshal(jsonBytes, &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to Arguments: %w", err)
	}

	return args, nil
}

// UnmarshalArgs unmarshals the entire Arguments map into the target struct.
//
// Details:
// - Uses encoding/json under the hood: the target must be a pointer to a struct
//   with appropriate json tags or exported field names that match keys.
// - Passing Arguments or map[string]any into NewArguments avoids any JSON encoding.
//
// Example:
//
//	type WeatherArgs struct {
//		Input    string `json:"input"`
//		Location string `json:"location"`
//		Unit     string `json:"unit"`
//	}
//
//	var args WeatherArgs
//	if err := arguments.UnmarshalArgs(&args); err != nil {
//		return nil, err
//	}
func UnmarshalArgs(a Arguments, target interface{}) error {
	if len(a) == 0 {
		return nil
	}

	// Convert Arguments map to JSON, then unmarshal to target type
	jsonBytes, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target type: %w", err)
	}

	return nil
}


