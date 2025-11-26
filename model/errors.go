// Package model provides interfaces and types for interacting with language models.
package model

import "errors"

// Common model-related errors.
var (
	// ErrInvalidModelDescription is returned when a model description is invalid.
	ErrInvalidModelDescription = errors.New("invalid model description")

	// ErrInvalidMessage is returned when a message is invalid.
	ErrInvalidMessage = errors.New("invalid message")

	// ErrInvalidRequest is returned when a completion request is invalid.
	ErrInvalidRequest = errors.New("invalid completion request")

	// ErrInvalidConfig is returned when model configuration is invalid.
	ErrInvalidConfig = errors.New("invalid model configuration")

	// ErrModelNotFound is returned when a requested model is not found.
	ErrModelNotFound = errors.New("model not found")

	// ErrRateLimitExceeded is returned when the rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrContextTooLong is returned when the context exceeds the model's maximum tokens.
	ErrContextTooLong = errors.New("context too long")

	// ErrInvalidToolCall is returned when a tool call is invalid.
	ErrInvalidToolCall = errors.New("invalid tool call")
)

