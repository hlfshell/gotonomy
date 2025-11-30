// Package agent provides interfaces and implementations for building AI agents
// that can use language models to accomplish tasks.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hlfshell/gogentic/context"
	"github.com/hlfshell/gogentic/model"
	"github.com/hlfshell/gogentic/tool"
)

// Agent is a simple expandable type of tool that utilizes an LLM
// to accomplish a task. They are a basic implementation of the tool.Tool
// interface, meaning they can be in turn handed off to other agents.
type Agent[I any, O any] struct {
	// name is the name of the agent. When used as a tool, this must be globally unique.
	name string
	// description is a human readable description of the agent
	description string
	// parameters is the list of parameters the agent accepts.
	parameters []tool.Parameter
	// model is the language model to use. (Required)
	model *model.Model
	// tools is the list of tools the agent can use.
	tools map[string]tool.Tool
	// prepareInput is a function that converts the arguments an agent
	// receives a prompt the agent uses. If nil, DefaultArgumentsToPrompt
	// is used.
	prepareInput PrepareInput
	// parser is the parser to use for structured output (optional)
	parser ParseResponse[O]
}

// AgentOption is a functional option for configuring an Agent.
type AgentOption[I any, O any] func(*Agent[I, O])

// DefaultArgumentsToPrompt marshals the entire arguments map to a single JSON string
// under the "input" key. This produces nested JSON when later embedded in prompts.
// If you want per-field templating, provide a custom ArgumentsToPrompt.
func DefaultArgumentsToPrompt(args tool.Arguments) (map[string]string, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}
	return map[string]string{
		"input": string(data),
	}, nil
}

// DefaultResponseParser returns the text output unchanged for string T.
// Limitation: Only supports T=string. For non-string types, returns zero-value and a parse error.
func DefaultResponseParser[I any, O any](input string) (O, []string) {
	var zero O
	if _, ok := any(zero).(string); ok {
		return any(input).(O), nil
	}
	return zero, []string{"default parser only supports string type"}
}

// NewAgent creates a new agent with the given name, description, and model.
// Additional configuration can be provided via AgentOption functions.
//
// Example:
//
//	agent := agent.NewAgent(
//	    "my_agent",
//	    "Performs calculations",
//	    myModel,
//	    agent.WithPrompt("You are a helpful calculator"),
//	    agent.WithTools(tool1, tool2),
//	    agent.WithTemperature(0.3),
//	)
func NewAgent[I any, O any](
	name, description string,
	model *model.Model,
	opts ...AgentOption[I, O],
) *Agent[I, O] {
	// Create agent with sensible defaults - default parameter is "input"
	defaultParams := []tool.Parameter{
		tool.NewParameter[string](
			"input",
			"The input for the agent",
			true,
			"",
			func(v string) (string, error) { return v, nil },
		),
	}

	a := &Agent[I, O]{
		name:         name,
		description:  description,
		parameters:   defaultParams,
		model:        model,
		tools:        make(map[string]tool.Tool),
		prepareInput: DefaultArgumentsToPrompt,
		parser:       DefaultResponseParser[I, O],
	}

	// Apply all options
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// WithPrompt is currently a no-op placeholder for future prompt templating.
// It exists to avoid breaking API, but does not alter behavior yet.
// Future versions will wire this into a prompt template system.
func WithPrompt[I any, O any](prompt string) AgentOption[I, O] {
	return func(a *Agent[I, O]) {
		// intentionally no-op
		_ = prompt
	}
}

// WithArgumentsParser sets a custom arguments parser for the agent.
// The parser converts tool arguments into a map[string]string for prompting.
func WithArgumentsParser[I any, O any](parser PrepareInput) AgentOption[I, O] {
	return func(a *Agent[I, O]) {
		a.prepareInput = parser
	}
}

// WithParser sets a custom output parser for the agent.
// The parser extracts structured data from the agent's text output.
func WithParser[I any, O any](parser ResponseParser[O]) AgentOption[I, O] {
	return func(a *Agent[I, O]) {
		a.parser = parser
	}
}

// WithParameters sets the parameters for the agent.
func WithParameters[I any, O any](parameters []tool.Parameter) AgentOption[I, O] {
	return func(a *Agent[I, O]) {
		a.parameters = parameters
	}
}

// WithTool adds a tool to the agent.
func WithTool[I any, O any](t tool.Tool) AgentOption[I, O] {
	return func(a *Agent[I, O]) {
		a.tools[t.Name()] = t
	}
}

// WithTools adds multiple tools to the agent.
func WithTools[I any, O any](tools []tool.Tool) AgentOption[I, O] {
	return func(a *Agent[I, O]) {
		for _, t := range tools {
			a.tools[t.Name()] = t
		}
	}
}

// Name returns the name of the agent. When used as a tool, this serves as the unique identifier.
func (a *Agent[I, O]) Name() string {
	return a.name
}

// Description returns a description of the agent.
func (a *Agent[I, O]) Description() string {
	return a.description
}

// Parameters returns the list of parameters for the agent.
func (a *Agent[I, O]) Parameters() []tool.Parameter {
	// Preserve declaration order; return a shallow copy for encapsulation
	result := make([]tool.Parameter, len(a.parameters))
	copy(result, a.parameters)
	return result
}

// Run accepts a given input, prepares it to a model.Call, gets a response,
// handles tool calls, then attempts to parse what it means until the agent
// declares itself done, all teh while tracing context.
func (a *Agent[I, O]) Run(ctx context.Context, input I) (O, error) {
	session := NewSession()

	for !session.Finished() {

	}
}

// Execute executes the agent with the given arguments and returns a result.
// This method implements the tool.Tool interface, allowing agents to be used as tools.
// Errors are returned as part of the ResultInterface, not as a separate error.
func (a *Agent[I, O]) Execute(ctx context.Context, args tool.Arguments) tool.ResultInterface {

	ectx, hasExecCtx := agentcontext.AsExecutionContext(ctx)
	if !hasExecCtx {
		return tool.NewError(fmt.Errorf("execution context not found"))
	}

	// Convert tool.Arguments to map[string]string via argumentsParser
	model_input, err := a.prepareInput(ectx, args)
	if err != nil {
		return tool.NewError(err)
	}

	// Call the model
	response, err := a.model.Complete(
		ctx,
		model.CompletionRequest{
			Messages: model_input,
			Tools:    a.tools,
			Config:   model.ModelConfig{},
		},
	)

	if err != nil {
		return tool.NewError(err)
	}

	// Parse the result using the agent's parser
	parsed, parseErrors := a.parser(result.Output)
	if len(parseErrors) > 0 {
		// If parsing failed, return the raw output as string
		// This allows the agent to still work even if parsing fails
		return tool.NewOK(result.Output)
	}

	// Return the parsed result
	return tool.NewOK(parsed)
}

// buildModelMessages converts a conversation to model messages.
func (a *Agent[I, O]) buildModelMessages(conversation *History) []model.Message {
	model_messages := []model.Message{}

	// TODO: Add system prompt if configured
	// if a.systemPrompt != "" {
	// 	model_messages = append(model_messages, model.Message{
	// 		Role: "system",
	// 		Content: []model.Content{
	// 			{
	// 				Type: model.TextContent,
	// 				Text: a.systemPrompt,
	// 			},
	// 		},
	// 	})
	// }

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

	return model_messages
}

// buildCompletionRequest builds a completion request from model messages and options.
func (a *Agent[I, O]) buildCompletionRequest(model_messages []model.Message, temperature float32, max_tokens int, stream bool) model.CompletionRequest {
	request := model.CompletionRequest{
		Messages:       model_messages,
		Temperature:    temperature,
		MaxTokens:      max_tokens,
		StreamResponse: stream,
	}

	// Add tools if they exist, using ParametersToJSONSchema to convert parameters
	if len(a.tools) > 0 {
		model_tools := []model.Tool{}
		// Stabilize tool listing order by name
		names := make([]string, 0, len(a.tools))
		for name := range a.tools {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			t := a.tools[name]
			model_tool := model.Tool{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  tool.ParametersToJSONSchema(t.Parameters()),
			}
			model_tools = append(model_tools, model_tool)
		}
		request.Tools = model_tools
	}

	return request
}

// processToolCalls processes a list of tool calls and returns the results.
func (a *Agent[I, O]) processToolCalls(
	ctx context.Context,
	tool_calls []model.ToolCall,
) ([]tool.ResultInterface, error) {
	tool_results := []tool.ResultInterface{}

	// Get ExecutionContext if available
	execCtx, hasExecCtx := agentcontext.AsExecutionContext(ctx)

	for _, tool_call := range tool_calls {
		// Create tool call node if ExecutionContext is available
		if hasExecCtx {
			toolNode, err := execCtx.CreateChildNode(nil, "tool", tool_call.Name, tool_call.Arguments)
			if err == nil {
				_ = toolNode
			}
		}

		// Find the tool
		t, exists := a.tools[tool_call.Name]
		if !exists {
			tool_result := tool.NewError(fmt.Errorf("tool not found: %s", tool_call.Name))
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				execCtx.SetError(fmt.Errorf("tool not found: %s", tool_call.Name))
			}
			continue
		}

		// Convert tool call arguments to tool.Arguments
		toolArgs := tool.Arguments(tool_call.Arguments)

		// Execute the tool
		tool_result := t.Execute(ctx, toolArgs)

		// Check if the tool result contains an error
		if tool_result.Errored() {
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				execCtx.SetError(tool_result.GetError())
			}
			continue
		}

		// Tool call succeeded - set output in execution context
		if hasExecCtx {
			str, serr := tool_result.String()
			if serr != nil {
				execCtx.SetError(serr)
			}
			agentcontext.SetOutput(execCtx, str)
			agentcontext.SetData(execCtx, "tool_result", tool_result.GetResult())
		}

		tool_results = append(tool_results, tool_result)
	}

	return tool_results, nil
}

// addToolResultsToConversation adds tool results as messages to the conversation.
func (a *Agent[I, O]) addToolResultsToConversation(conversation *History, tool_results []tool.ResultInterface) {
	for _, tool_result := range tool_results {
		// Use the string view of the result for conversation
		content, err := tool_result.String()
		if err != nil {
			// Surface stringification errors in the conversation
			content = err.Error()
		}
		tool_message := Message{
			Role:      "tool",
			Content:   content,
			Timestamp: time.Now(),
		}
		conversation.Messages = append(conversation.Messages, tool_message)
	}
	conversation.UpdatedAt = time.Now()
}

// agentExecutionResult holds the result of an agent execution.
type agentExecutionResult struct {
	Output string
}
