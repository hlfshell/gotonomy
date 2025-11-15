package agent

import (
	"context"
	"time"

	arkaineparser "github.com/hlfshell/go-arkaine-parser"
	"github.com/hlfshell/gogentic/pkg/model"
)

// Message represents a message in a conversation with an agent.
type Message struct {
	// Role is the role of the message sender (e.g., "system", "user", "assistant", "tool").
	Role string
	// Content is the text content of the message.
	Content string
	// Timestamp is when the message was created.
	Timestamp time.Time
	// ToolCalls contains any tool calls made in this message.
	ToolCalls []model.ToolCall
	// ToolResults contains the results of any tool calls.
	ToolResults []ToolResultInterface
}

// Conversation represents a conversation between a user and an agent.
type Conversation struct {
	// ID is a unique identifier for the conversation.
	ID string
	// Messages is the list of messages in the conversation.
	Messages []Message
	// Metadata is additional metadata about the conversation.
	Metadata map[string]interface{}
	// CreatedAt is when the conversation was created.
	CreatedAt time.Time
	// UpdatedAt is when the conversation was last updated.
	UpdatedAt time.Time
}

// Tool represents a tool that an agent can use.
type Tool struct {
	// Name is the name of the tool.
	Name string
	// Description is a description of what the tool does.
	Description string
	// Parameters is a map of parameter names to their JSON schema.
	Parameters map[string]interface{}
	// Handler is the function that handles the tool call.
	// Can be either ToolHandler (string return) or ToolHandlerInterface (generic return).
	Handler interface{}
}

// ToolHandler is a function that handles a tool call and returns a string.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)

// ToolHandlerInterface is an interface for tool handlers that can return any type.
// This allows tools to return typed results instead of just strings.
type ToolHandlerInterface interface {
	// Call executes the tool handler and returns a ToolResultInterface.
	Call(ctx context.Context, args map[string]interface{}) (ToolResultInterface, error)
}

// GenericToolHandler wraps a generic handler function.
type GenericToolHandler[T any] struct {
	handler  func(ctx context.Context, args map[string]interface{}) (T, error)
	toolName string
}

// StringToolHandler wraps a legacy ToolHandler (string return) to implement ToolHandlerInterface.
type StringToolHandler struct {
	handler  ToolHandler
	toolName string
}

// AgentConfig represents the configuration for an agent.
type AgentConfig struct {
	// Model is the language model to use.
	Model model.Model
	// SystemPrompt is the system prompt to use.
	SystemPrompt string
	// Tools is the list of tools the agent can use.
	Tools []Tool
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int
	// Temperature controls randomness in the response (0.0 to 1.0).
	Temperature float32
	// MaxIterations is the maximum number of iterations for tool use.
	MaxIterations int
	// Timeout is the timeout for the agent's execution.
	Timeout time.Duration
	// ParserLabels is the list of labels for the parser.
	ParserLabels []arkaineparser.Label
}

// AgentParameters represents input parameters for an agent execution.
type AgentParameters struct {
	// Input is the primary input for the agent (could be a question, task description, etc.)
	Input string

	// AdditionalInputs contains any additional inputs keyed by name
	AdditionalInputs map[string]interface{}

	// Conversation is an optional conversation history
	Conversation *Conversation

	// Options contains execution options that may override agent defaults
	Options AgentOptions

	// StreamHandler is an optional handler for streaming responses.
	// If provided and the model supports streaming, responses will be streamed.
	StreamHandler StreamHandler
}

// AgentOptions contains options for agent execution.
type AgentOptions struct {
	// Temperature controls randomness (0.0 to 1.0)
	Temperature *float32

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens *int

	// Timeout is the execution timeout
	Timeout *time.Duration
}

// ExecutionStats contains statistics about an agent execution.
type ExecutionStats struct {
	// StartTime is when execution started
	StartTime time.Time

	// EndTime is when execution completed
	EndTime time.Time

	// ToolCalls is the number of tool calls made
	ToolCalls int

	// Iterations is the number of reasoning iterations
	Iterations int
}

// AgentResult represents the result of an agent execution.
type AgentResult struct {
	// Output is the primary output text
	Output string

	// AdditionalOutputs contains any additional outputs keyed by name
	AdditionalOutputs map[string]interface{}

	// Conversation is the updated conversation if one was provided
	Conversation *Conversation

	// UsageStats contains token usage statistics
	UsageStats model.UsageStats

	// ExecutionStats contains information about the execution
	ExecutionStats ExecutionStats

	// Message is the final message from the agent
	Message Message

	// ParsedOutput contains the structured output parsed by the agent's parser
	ParsedOutput map[string]interface{}

	// ParseErrors contains any errors that occurred during parsing
	ParseErrors []string
}

// StreamHandler is a function that handles streamed agent responses.
type StreamHandler func(message Message) error

// Agent represents an AI agent that can execute tasks using a language model.
type Agent interface {
	// Execute processes the given parameters and returns a result.
	// This is the core method that all agents must implement.
	// If params.StreamHandler is provided and the model supports streaming,
	// responses will be streamed incrementally.
	Execute(ctx context.Context, params AgentParameters) (AgentResult, error)

	// ID returns the unique identifier for the agent.
	ID() string

	// Name returns the name of the agent.
	Name() string

	// Description returns a description of the agent.
	Description() string

	// GetParser returns the agent's parser.
	GetParser() *arkaineparser.Parser
}

// BaseAgent is a basic implementation of the Agent interface that can be embedded in other agent implementations.
type BaseAgent struct {
	// id is the unique identifier for the agent.
	id string
	// name is the name of the agent.
	name string
	// description is a description of the agent.
	description string
	// config is the agent's configuration.
	config AgentConfig
	// parser is the agent's parser for structured output.
	parser *arkaineparser.Parser
}
