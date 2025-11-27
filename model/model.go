// Package model provides interfaces and types for interacting with language models.
package model

import (
	"context"
	"fmt"

	"github.com/hlfshell/gotonomy/tool"
)

// Role represents the role of a message sender; different providers
// can expect different roles.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ModelDescription contains metadata about a model.
type ModelDescription struct {
	Model            string        `json:"model"` // Needs to be the canonical provider ID of the model
	Provider         string        `json:"provider"`
	MaxContextTokens int           `json:"max_context_tokens"`
	Description      string        `json:"description"`
	Costs            CostsPerToken `json:"costs"`
	// CanUseTools indicates whether the model can use tools/functions
	// If not, you need to use a ReAct wrapper to add it to the model.
	CanUseTools bool `json:"can_use_tools"`
}

// Validate validates the model description.
func (m ModelDescription) Validate() error {
	if m.Model == "" {
		return fmt.Errorf("%w: model name is required", ErrInvalidModelDescription)
	}
	if m.Provider == "" {
		return fmt.Errorf("%w: provider is required", ErrInvalidModelDescription)
	}
	if m.MaxContextTokens <= 0 {
		return fmt.Errorf("%w: MaxContextTokens must be greater than 0", ErrInvalidModelDescription)
	}
	return nil
}

// CostsPerToken contains the costs per token for a model.
type CostsPerToken struct {
	Input     float64 `json:"input"`
	Output    float64 `json:"output"`
	Reasoning float64 `json:"reasoning"` // Only some providers consider reasoning tokens separately
}

// Cost calculates the total cost for the given token counts.
func (c CostsPerToken) Cost(input, output, reasoning int) float64 {
	return c.Input*float64(input) + c.Output*float64(output) + c.Reasoning*float64(reasoning)
}

// Message represents a message in a conversation with a model.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Validate validates the message.
func (m Message) Validate() error {
	if m.Role == "" {
		return fmt.Errorf("%w: role is required", ErrInvalidMessage)
	}
	switch m.Role {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		// Valid role
	default:
		return fmt.Errorf("%w: invalid role %q", ErrInvalidMessage, m.Role)
	}
	return nil
}

// ModelConfig contains configuration for model completion requests.
type ModelConfig struct {
	// Temperature controls randomness in the response (0.0 to 1.0).
	// Higher values make output more random, lower values more deterministic.
	Temperature float32 `json:"temperature"`
}

// Validate validates the model configuration.
func (c ModelConfig) Validate() error {
	if c.Temperature < 0.0 || c.Temperature > 1.0 {
		return fmt.Errorf("%w: temperature must be between 0.0 and 1.0, got %f", ErrInvalidConfig, c.Temperature)
	}
	return nil
}

// CompletionRequest represents a request for a model completion
type CompletionRequest struct {
	Messages []Message   `json:"messages"`
	Tools    []tool.Tool `json:"tools"`
	Config   ModelConfig `json:"config"`
}

// Validate validates the completion request.
func (r CompletionRequest) Validate() error {
	if len(r.Messages) == 0 {
		return fmt.Errorf("%w: at least one message is required", ErrInvalidRequest)
	}
	for i, msg := range r.Messages {
		if err := msg.Validate(); err != nil {
			return fmt.Errorf("message %d: %w", i, err)
		}
	}
	if err := r.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

// ToolCall represents the instance of calling a tool via the model
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments tool.Arguments `json:"arguments"`
}

// CompletionResponse represents a response from a model completion request
type CompletionResponse struct {
	Text       string     `json:"text"`
	ToolCalls  []ToolCall `json:"tool_calls"`
	UsageStats UsageStats `json:"usage_stats"`
}

// UsageStats contains token usage statistics for a model request.
type UsageStats struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens"`
}

// Total returns the total number of tokens used.
func (u UsageStats) Total() int {
	return u.InputTokens + u.OutputTokens + u.ReasoningTokens
}

// Add returns a new UsageStats with the sum of this and the other UsageStats.
func (u UsageStats) Add(other UsageStats) UsageStats {
	return UsageStats{
		InputTokens:     u.InputTokens + other.InputTokens,
		OutputTokens:    u.OutputTokens + other.OutputTokens,
		ReasoningTokens: u.ReasoningTokens + other.ReasoningTokens,
	}
}

// Model is an individual interface to a specific model from a provider;
// each provider will implement this interface for their given subset
// of models.
type Model interface {
	// Description returns information about the model.
	Description() ModelDescription

	// Complete generates a completion for the given request.
	Complete(ctx context.Context, request CompletionRequest) (CompletionResponse, error)
}
