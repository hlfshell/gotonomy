package tool

import (
	"encoding/json"
	"fmt"
)

// ResultInterface is the interface that all tool results implement.
type ResultInterface interface {
	Errored() bool
	GetError() error
	GetResult() any
	ToJSON() ([]byte, error)
	// MarshalJSON encodes only the underlying result value to JSON, even if the Result is errored.
	// Use ToJSON when you want error propagation for errored Results.
	MarshalJSON() ([]byte, error)
	// String converts the result to a string representation for LLM consumption
	// Returns an error if the Result is errored or if serialization fails.
	String() (string, error)
}

// Result is a generic type that can hold any JSON-serializable value.
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

// GetError returns the error.
func (r Result[T]) GetError() error {
	return r.Error
}

// GetResult returns the result as any.
func (r Result[T]) GetResult() any {
	return r.Result
}

// ToJSON marshals the result to JSON bytes.
// If the Result is errored, the error is returned (no JSON is produced).
func (r Result[T]) ToJSON() ([]byte, error) {
	if r.Errored() {
		return nil, r.Error
	}
	return json.Marshal(r.Result)
}

// String converts the result to a string representation.
// For primitives, returns the string value directly.
// For complex types, returns JSON string.
// If the Result is errored, the error is returned.
func (r Result[T]) String() (string, error) {
	if r.Errored() {
		return "", r.Error
	}

	// Use JSON marshaling for all types - it handles primitives and complex types uniformly
	jsonBytes, err := json.Marshal(r.Result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(jsonBytes), nil
}

// MarshalJSON implements json.Marshaler for Result.
// Note: This encodes the underlying result value regardless of error state.
// Prefer ToJSON if you need error propagation semantics.
func (r Result[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Result)
}

// UnmarshalJSON implements json.Unmarshaler for Result.
func (r *Result[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Result)
}

// NewOK creates a successful result with the given value.
func NewOK[T any](value T) ResultInterface {
	return Result[T]{
		Result: value,
		Error:  nil,
	}
}

// NewError creates an error result with the given error.
func NewError(err error) ResultInterface {
	return Result[any]{
		Result: nil,
		Error:  err,
	}
}

// BlankResult creates a result with the given value and error.
// This is a convenience function for creating results when you don't need type safety.
func BlankResult(result any, err error) ResultInterface {
	return Result[any]{
		Result: result,
		Error:  err,
	}
}
