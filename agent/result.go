package agent

import (
	"encoding/json"
	"fmt"
)

// ResultInterface is the interface that all ToolResult types implement.
// This allows us to store different ToolResult types without generics cascading.
type ResultInterface interface {
	Errored() bool
	GetError() error
	GetResult() interface{}
	ToJSON() ([]byte, error)
	MarshalJSON() ([]byte, error)
	// String is possibly the most important, as its assumed that
	// You will want to convert the data to a human readable string
	// for easier consumption by the LLM for calling it.
	String() (string, error)
}

// Result is a generic type that can hold any JSON-serializable value.
// T can be a primitive type (string, int, float, bool, etc.) or any struct
// that can be serialized/deserialized to JSON.
type Result[T any] struct {
	// Result is the result of the tool call.
	Result T
	// Error is any error that occurred during the tool call.
	Error error
}

// Errored returns true if the result has an error.
func (r Result[T]) Errored() bool {
	return r.Error != nil
}

// GetError returns the error string.
func (r Result[T]) GetError() error {
	return r.Error
}

// GetResult returns the result as interface{}.
func (r Result[T]) GetResult() interface{} {
	return r.Result
}

// ToJSON marshals the result to JSON bytes.
func (r Result[T]) ToJSON() ([]byte, error) {
	if r.Errored() {
		return nil, r.Error
	}
	return json.Marshal(r.Result)
}

// String converts the result to a string representation.
// For primitives, returns the string value.
// For objects, returns JSON string.
func (r Result[T]) String() (string, error) {
	if r.Errored() {
		return "", r.Error
	}

	// Handle primitives directly
	switch v := any(r.Result).(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%g", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	default:
		// For complex types, marshal to JSON
		jsonBytes, err := json.Marshal(r.Result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// MarshalJSON implements json.Marshaler for Result.
func (r Result[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Result)
}

// UnmarshalJSON implements json.Unmarshaler for ToolResult.
func (r *Result[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Result)
}

func BlankResult(result interface{}, err error) ResultInterface {
	return Result[interface{}]{
		Result: result,
		Error:  err,
	}
}
