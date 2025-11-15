// Package agent provides interfaces and implementations for building AI agents
// that can use language models to accomplish tasks.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	arkaineparser "github.com/hlfshell/go-arkaine-parser"
	"github.com/hlfshell/gogentic/pkg/model"
)

// ToolResultInterface is the interface that all ToolResult types implement.
// This allows us to store different ToolResult types without generics cascading.
type ToolResultInterface interface {
	GetToolName() string
	GetError() string
	GetResult() interface{}
	ToJSON() ([]byte, error)
	String() string
	MarshalJSON() ([]byte, error)
}

// ToolResult is a generic type that can hold any JSON-serializable value.
// T can be a primitive type (string, int, float, bool, etc.) or any struct
// that can be serialized/deserialized to JSON.
type ToolResult[T any] struct {
	// ToolName is the name of the tool that was called.
	ToolName string
	// Result is the result of the tool call.
	Result T
	// Error is any error that occurred during the tool call.
	Error string
}

// Ensure ToolResult implements ToolResultInterface
var _ ToolResultInterface = ToolResult[string]{}

// GetToolName returns the tool name.
func (tr ToolResult[T]) GetToolName() string {
	return tr.ToolName
}

// GetError returns the error string.
func (tr ToolResult[T]) GetError() string {
	return tr.Error
}

// GetResult returns the result as interface{}.
func (tr ToolResult[T]) GetResult() interface{} {
	return tr.Result
}

// ToJSON marshals the result to JSON bytes.
func (tr ToolResult[T]) ToJSON() ([]byte, error) {
	if tr.Error != "" {
		return json.Marshal(map[string]string{"error": tr.Error})
	}
	return json.Marshal(tr.Result)
}

// String converts the result to a string representation.
// For primitives, returns the string value.
// For objects, returns JSON string.
func (tr ToolResult[T]) String() string {
	if tr.Error != "" {
		return tr.Error
	}

	// Handle primitives directly
	switch v := any(tr.Result).(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		// For complex types, marshal to JSON
		jsonBytes, err := json.Marshal(tr.Result)
		if err != nil {
			return fmt.Sprintf("%v", tr.Result)
		}
		return string(jsonBytes)
	}
}

// MarshalJSON implements json.Marshaler for ToolResult.
func (tr ToolResult[T]) MarshalJSON() ([]byte, error) {
	result := map[string]interface{}{
		"tool_name": tr.ToolName,
	}

	if tr.Error != "" {
		result["error"] = tr.Error
		result["result"] = nil
	} else {
		result["result"] = tr.Result
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements json.Unmarshaler for ToolResult.
func (tr *ToolResult[T]) UnmarshalJSON(data []byte) error {
	var aux struct {
		ToolName string          `json:"tool_name"`
		Result   json.RawMessage `json:"result"`
		Error    string          `json:"error"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	tr.ToolName = aux.ToolName
	tr.Error = aux.Error

	if aux.Error == "" && len(aux.Result) > 0 {
		return json.Unmarshal(aux.Result, &tr.Result)
	}

	return nil
}

// NewToolResult creates a new ToolResult with type safety.
func NewToolResult[T any](toolName string, result T) ToolResultInterface {
	return ToolResult[T]{
		ToolName: toolName,
		Result:   result,
	}
}

// NewToolResultError creates a ToolResult with an error.
func NewToolResultError(toolName string, err error) ToolResultInterface {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	return ToolResult[string]{
		ToolName: toolName,
		Error:    errStr,
	}
}

// NewGenericToolHandler creates a new generic tool handler.
func NewGenericToolHandler[T any](toolName string, handler func(ctx context.Context, args map[string]interface{}) (T, error)) ToolHandlerInterface {
	return &GenericToolHandler[T]{
		handler:  handler,
		toolName: toolName,
	}
}

// Call implements ToolHandlerInterface.
func (h *GenericToolHandler[T]) Call(ctx context.Context, args map[string]interface{}) (ToolResultInterface, error) {
	result, err := h.handler(ctx, args)
	if err != nil {
		return NewToolResultError(h.toolName, err), err
	}
	return NewToolResult(h.toolName, result), nil
}

// NewStringToolHandler creates a wrapper for legacy string-returning handlers.
func NewStringToolHandler(toolName string, handler ToolHandler) ToolHandlerInterface {
	return &StringToolHandler{
		handler:  handler,
		toolName: toolName,
	}
}

// Call implements ToolHandlerInterface.
func (h *StringToolHandler) Call(ctx context.Context, args map[string]interface{}) (ToolResultInterface, error) {
	result, err := h.handler(ctx, args)
	if err != nil {
		return NewToolResultError(h.toolName, err), err
	}
	return NewToolResult(h.toolName, result), nil
}

// NewBaseAgent creates a new base agent with the given configuration.
func NewBaseAgent(id, name, description string, config AgentConfig) *BaseAgent {
	// Set default values if not provided
	if id == "" {
		id = uuid.New().String()
	}

	if config.MaxTokens <= 0 {
		config.MaxTokens = 1000
	}

	if config.Temperature <= 0 {
		config.Temperature = 0.7
	}

	if config.MaxIterations <= 0 {
		config.MaxIterations = 5
	}

	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}

	// Create a parser with provided labels or use default labels
	var parser_labels []arkaineparser.Label
	if len(config.ParserLabels) > 0 {
		parser_labels = config.ParserLabels
	} else {
		// Default labels for ReAct-style agents
		parser_labels = []arkaineparser.Label{
			{Name: "Reasoning", IsBlockStart: true},
			{Name: "Action"},
			{Name: "Action Input", IsJSON: true},
		}
	}
	parser, _ := arkaineparser.NewParser(parser_labels)

	return &BaseAgent{
		id:          id,
		name:        name,
		description: description,
		config:      config,
		parser:      parser,
	}
}

// ID returns the unique identifier for the agent.
func (a *BaseAgent) ID() string {
	return a.id
}

// Name returns the name of the agent.
func (a *BaseAgent) Name() string {
	return a.name
}

// Description returns a description of the agent.
func (a *BaseAgent) Description() string {
	return a.description
}

// GetParser returns the agent's parser.
func (a *BaseAgent) GetParser() *arkaineparser.Parser {
	return a.parser
}

// Config returns the agent's configuration.
func (a *BaseAgent) Config() AgentConfig {
	return a.config
}

// Execute processes the given parameters and returns a result.
// This is a basic implementation that should be overridden by specific agent types.
func (a *BaseAgent) Execute(ctx context.Context, params AgentParameters) (AgentResult, error) {
	// Get or create ExecutionContext
	execCtx := InitContext(ctx)
	ctx = execCtx // Use ExecutionContext as the context going forward

	// Create agent execution node and set as current
	agentNode, err := execCtx.PushCurrentNode("agent", a.name, map[string]interface{}{
		"input":    params.Input,
		"agent_id": a.id,
	})
	if err != nil {
		return AgentResult{}, fmt.Errorf("failed to create agent node: %w", err)
	}
	_ = agentNode // Use agentNode to avoid unused variable warning

	// Set execution-level data
	SetExecutionData(execCtx, "agent_id", a.id)
	SetExecutionData(execCtx, "agent_name", a.name)

	// Record execution start time
	start_time := time.Now()

	// Apply options if provided
	temperature := a.config.Temperature
	if params.Options.Temperature != nil {
		temperature = *params.Options.Temperature
	}

	max_tokens := a.config.MaxTokens
	if params.Options.MaxTokens != nil {
		max_tokens = *params.Options.MaxTokens
	}

	timeout := a.config.Timeout
	if params.Options.Timeout != nil {
		timeout = *params.Options.Timeout
	}

	// Create a timeout context if needed
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Initialize or use provided conversation
	var conversation *Conversation
	if params.Conversation != nil {
		conversation = params.Conversation
	} else {
		conversation = &Conversation{
			ID:        uuid.New().String(),
			Messages:  []Message{},
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Create a user message
	user_message := Message{
		Role:      "user",
		Content:   params.Input,
		Timestamp: time.Now(),
	}

	// Add the message to the conversation
	conversation.Messages = append(conversation.Messages, user_message)
	conversation.UpdatedAt = time.Now()

	// Convert the conversation to a model request
	model_messages := []model.Message{}

	// Add the system prompt if it exists
	if a.config.SystemPrompt != "" {
		model_messages = append(model_messages, model.Message{
			Role: "system",
			Content: []model.Content{
				{
					Type: model.TextContent,
					Text: a.config.SystemPrompt,
				},
			},
		})
	}

	// Add the conversation messages
	for _, msg := range conversation.Messages {
		model_msg := model.Message{
			Role: msg.Role,
			Content: []model.Content{
				{
					Type: model.TextContent,
					Text: msg.Content,
				},
			},
		}
		model_messages = append(model_messages, model_msg)
	}

	// Create the model request
	request := model.CompletionRequest{
		Messages:    model_messages,
		Temperature: temperature,
		MaxTokens:   max_tokens,
	}

	// Add tools if they exist
	if len(a.config.Tools) > 0 {
		model_tools := []model.Tool{}
		for _, tool := range a.config.Tools {
			model_tool := model.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			}
			model_tools = append(model_tools, model_tool)
		}
		request.Tools = model_tools
	}

	// Get a completion from the model
	response, err := a.config.Model.Complete(ctx, request)
	if err != nil {
		execCtx.SetError(err)
		return AgentResult{}, err
	}

	// Create the agent message
	agent_message := Message{
		Role:      "assistant",
		Content:   response.Text,
		Timestamp: time.Now(),
		ToolCalls: response.ToolCalls,
	}

	// Add the agent message to the conversation
	conversation.Messages = append(conversation.Messages, agent_message)
	conversation.UpdatedAt = time.Now()

	// Record execution end time
	end_time := time.Now()

	// Parse the response text using the agent's parser
	parsed_output, parse_errors := a.parser.Parse(response.Text)

	// Set output in execution context
	if err := SetOutput(execCtx, agent_message.Content); err == nil {
		SetData(execCtx, "tool_calls_count", len(response.ToolCalls))
		SetData(execCtx, "usage_stats", response.UsageStats)
	}

	// Return the agent result
	result := AgentResult{
		Output:            response.Text,
		AdditionalOutputs: map[string]interface{}{},
		Conversation:      conversation,
		UsageStats:        response.UsageStats,
		ExecutionStats: ExecutionStats{
			StartTime:  start_time,
			EndTime:    end_time,
			ToolCalls:  len(response.ToolCalls),
			Iterations: 1,
		},
		Message:      agent_message,
		ParsedOutput: parsed_output,
		ParseErrors:  parse_errors,
	}
	return result, nil
}
