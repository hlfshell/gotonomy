package agent

import (
	"encoding/json"
	"fmt"
)

// Arguments represents input arguments for tool or agent execution.
// It's a map where all arguments are key-value pairs.
// Common keys include "input" for the primary input, but any key can be used.
type Arguments map[string]interface{}

// NewArguments creates an Arguments map from any value by marshaling it to JSON
// and then unmarshaling it into a map. This allows you to create Arguments
// from structs, maps, or any JSON-serializable value.
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
// This provides type-safe extraction of arguments for tool implementations.
// The target must be a pointer to a struct with JSON tags.
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
func (a Arguments) UnmarshalArgs(target interface{}) error {
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
