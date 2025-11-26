package agent

import (
	"time"

	"github.com/hlfshell/gogentic/model"
	"github.com/hlfshell/gogentic/tool"
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
	ToolResults []tool.ResultInterface
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

// ResponseParser is a generic function type that parses agent output text into type T.
// It returns the parsed value and a list of parse errors (if any).
type ResponseParser[T any] func(input string) (T, []string)

type ArgumentsToPrompt func(args tool.Arguments) (map[string]string, error)

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
