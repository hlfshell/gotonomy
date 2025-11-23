// Package agent provides interfaces and implementations for building AI agents
// that can use language models to accomplish tasks.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	agentcontext "github.com/hlfshell/gogentic/context"
	"github.com/hlfshell/gogentic/model"
	"github.com/hlfshell/gogentic/tool"
)

// Agent is a simple expandable type of tool that utilizes an LLM
// to accomplish a task. They are a basic implementation of the GotonomyTool
// interface, meaning they can be in turn handed off to other agents.
type Agent[T any] struct {
	// name is the name of the agent. When used as a tool, this must be globally unique.
	name string
	// description is a human readable description of the agent
	description string
	// arguments is the arguments the agent can use.
	arguments Arguments
	// model is the language model to use. (Required)
	model *model.Model
	// tools is the list of tools the agent can use.
	tools map[string]tool.GotonomyTool
	// argumentsParser is a function that converts the arguments an agent
	// receives to map[string]string. If nil, DefaultArgumentsToPrompt is used.
	argumentsParser ArgumentsToPrompt
	// parser is the parser to use for structured output (optional)
	parser ResponseParserInterface
}

// AgentOption is a functional option for configuring an Agent.
type AgentOption[T any] func(*Agent[T])

// DefaultArgumentsToPrompt just json marshals the arguments to a string
func DefaultArgumentsToPrompt(args Arguments) (map[string]string, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}
	return map[string]string{
		"input": string(data),
	}, nil
}

// DefaultResponseParser just returns the text output as is
func DefaultResponseParser(input string) ResultInterface {
	return Result[string]{
		Result: input,
		Error:  nil,
	}
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
func NewAgent[T any](
	name, description string,
	model *model.Model,
	opts ...AgentOption[T],
) *Agent[T] {
	// Create agent with sensible defaults
	a := &Agent[T]{
		name:            name,
		description:     description,
		model:           model,
		tools:           make(map[string]GotonomyTool),
		argumentsParser: DefaultArgumentsToPrompt,
		parser:          DefaultResponseParser,
	}

	// Apply all options
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// WithPrompt sets the system prompt for the agent, utilizing the
// SimplePromptTemplate function. Will overwrite any existing
// arguments parser.
func WithPrompt[T any](prompt string) AgentOption[T] {
	return func(a *Agent[T]) {
		a.argumentsParser = SimplePromptTemplate(prompt, a.arguments)
	}
}

// WithArgumentsParser sets a custom arguments parser for the agent.
// The parser converts tool arguments into a map[string]string for prompting.
func WithArgumentsParser[T any](parser ArgumentsToPrompt) AgentOption[T] {
	return func(a *Agent[T]) {
		a.argumentsParser = parser
	}
}

// WithParser sets a custom output parser for the agent.
// The parser extracts structured data from the agent's text output.
func WithParser[T any](parser ResponseParserInterface) AgentOption[T] {
	return func(a *Agent[T]) {
		a.parser = parser
	}
}

// WithTool adds a tool to the agent.
func WithTool[T any](tool GotonomyTool) AgentOption[T] {
	return func(a *Agent[T]) {
		a.tools[tool.Name()] = tool
	}
}

// WithTools adds multiple tools to the agent.
func WithTools[T any](tools []GotonomyTool) AgentOption[T] {
	return func(a *Agent[T]) {
		for _, tool := range tools {
			a.tools[tool.Name()] = tool
		}
	}
}

// Name returns the name of the agent. When used as a tool, this serves as the unique identifier.
func (a *Agent[T]) Name() string {
	return a.name
}

// Description returns a description of the agent.
func (a *Agent[T]) Description() string {
	return a.description
}

// Parameters returns the JSON schema for the agent's parameters when used as a tool.
// By default, agents accept an "input" parameter.
// This method is part of the GotonomyTool interface.
func (a *Agent[T]) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "The input for the agent",
			},
		},
		"required": []string{"input"},
	}
}

// Execute executes the agent with the given arguments and returns a result.
// This method implements the GotonomyTool interface, allowing agents to be used as tools.
// Errors are returned as part of the ResultInterface, not as a separate error.
func (a *Agent[T]) Execute(ctx context.Context, args Arguments) ResultInterface {
	parsedArgs, err := a.argumentsParser(args)
	if err != nil {
		return BlankResult(nil, err)
	}

	// Convert the parsed arguments map to a JSON string for the prompt
	inputData, err := json.Marshal(parsedArgs)
	if err != nil {
		return BlankResult(nil, fmt.Errorf("failed to marshal parsed arguments: %w", err))
	}
	input := string(inputData)

	// Execute the agent with the provided arguments (non-streaming)
	// Tools don't pass options - use agent defaults
	result, err := a.executeInternal(ctx, input, nil, nil)

	if err != nil {
		return BlankResult(nil, err)
	}

	// Return the agent result wrapped in a Result
	return Result[T]{
		Result: result,
		Error:  err,
	}
}

// TODO WORK ON THESE PARTS:

// // buildModelMessages converts a conversation to model messages.
// func (a *Agent) buildModelMessages(conversation *Conversation) []model.Message {
// 	model_messages := []model.Message{}

// 	// Add the system prompt if it exists
// 	if a.prompt != "" {
// 		model_messages = append(model_messages, model.Message{
// 			Role: "system",
// 			Content: []model.Content{
// 				{
// 					Type: model.TextContent,
// 					Text: a.prompt,
// 				},
// 			},
// 		})
// 	}

// 	// Add the conversation messages
// 	for _, msg := range conversation.Messages {
// 		model_msg := model.Message{
// 			Role: msg.Role,
// 			Content: []model.Content{
// 				{
// 					Type: model.TextContent,
// 					Text: msg.Content,
// 				},
// 			},
// 		}
// 		model_messages = append(model_messages, model_msg)
// 	}

// 	return model_messages
// }

// // buildCompletionRequest builds a completion request from model messages and options.
// func (a *Agent) buildCompletionRequest(model_messages []model.Message, temperature float32, max_tokens int, stream bool) model.CompletionRequest {
// 	request := model.CompletionRequest{
// 		Messages:       model_messages,
// 		Temperature:    temperature,
// 		MaxTokens:      max_tokens,
// 		StreamResponse: stream,
// 	}

// 	// Add tools if they exist
// 	if len(a.tools) > 0 {
// 		model_tools := []model.Tool{}
// 		for _, tool := range a.tools {
// 			model_tool := model.Tool{
// 				Name:        tool.Name(),
// 				Description: tool.Description(),
// 				Parameters:  tool.Parameters(),
// 			}
// 			model_tools = append(model_tools, model_tool)
// 		}
// 		request.Tools = model_tools
// 	}

// 	return request
// }

// processToolCalls processes a list of tool calls and returns the results.
func (a *Agent[T]) processToolCalls(
	ctx context.Context,
	tool_calls []model.ToolCall,
) ([]ResultInterface, error) {
	tool_results := []ResultInterface{}

	// Get ExecutionContext if available
	execCtx, hasExecCtx := agentcontext.AsExecutionContext(ctx)

	for _, tool_call := range tool_calls {
		// Create tool call node if ExecutionContext is available
		var toolNode *agentcontext.Node
		if hasExecCtx {
			var err error
			toolNode, err = execCtx.CreateChildNode(nil, "tool", tool_call.Name, tool_call.Arguments)
			if err == nil {
				_ = toolNode
			}
		}

		// Find the tool
		if tool, exists := a.tools[tool_call.Name]; !exists {
			tool_result := NewToolResultError(tool_call.Name, fmt.Errorf("tool not found: %s", tool_call.Name))
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				execCtx.SetError(fmt.Errorf("tool not found: %s", tool_call.Name))
			}
			continue
		} else {
			tool_result := tool.Execute(ctx, toolArgs)
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				agentcontext.SetOutput(execCtx, tool_result.String())
				agentcontext.SetData(execCtx, "tool_result", tool_result.GetResult())
			}
		}

		// Convert tool call arguments to Arguments
		// Just use the map directly - no need to separate input
		toolArgs := Arguments(tool_call.Arguments)

		// Execute the tool directly - Tool interface has Execute() method
		tool_result := tool.Execute(ctx, toolArgs)

		// Check if the tool result contains an error
		if tool_result.Errored() {
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				execCtx.SetError(fmt.Errorf(tool_result.GetError()))
			}
			continue
		}

		// Tool call succeeded - set output in execution context
		if hasExecCtx {
			agentcontext.SetOutput(execCtx, tool_result.String())
			agentcontext.SetData(execCtx, "tool_result", tool_result.GetResult())
		}

		// Tool call succeeded
		tool_results = append(tool_results, tool_result)
	}

	return tool_results, nil
}

// addToolResultsToConversation adds tool results as messages to the conversation.
func (a *Agent) addToolResultsToConversation(conversation *Conversation, tool_results []ResultInterface) {
	for _, tool_result := range tool_results {
		tool_message := Message{
			Role:      "tool",
			Content:   tool_result.String(),
			Timestamp: time.Now(),
		}
		conversation.Messages = append(conversation.Messages, tool_message)
	}
	conversation.UpdatedAt = time.Now()
}

// executeInternal is the internal implementation used by both ExecuteAgent and ExecuteStream.
func (a *Agent) executeInternal(
	ctx context.Context,
	input string,
) (AgentResult, error) {
	// Get or create ExecutionContext
	execCtx := agentcontext.InitContext(ctx)
	ctx = execCtx // Use ExecutionContext as the context going forward

	// Create agent execution node and set as current
	agentNode, err := execCtx.CreateChildNode(nil, "agent", a.name, map[string]interface{}{
		"input":      input,
		"agent_id":   a.id,
		"agent_type": "base",
	})
	if err != nil {
		return AgentResult{}, fmt.Errorf("failed to create agent node: %w", err)
	}
	if err := execCtx.SetCurrentNode(agentNode); err != nil {
		return AgentResult{}, fmt.Errorf("failed to set current node: %w", err)
	}
	_ = agentNode

	// Set execution-level data
	agentcontext.SetExecutionData(execCtx, "agent_id", a.id)
	agentcontext.SetExecutionData(execCtx, "agent_name", a.name)
	agentcontext.SetExecutionData(execCtx, "agent_type", "base")

	// Record execution start time
	start_time := time.Now()

	// Apply options if provided
	temperature := a.temperature
	if options != nil && options.Temperature != nil {
		temperature = *options.Temperature
	}

	max_tokens := a.maxTokens
	if options != nil && options.MaxTokens != nil {
		max_tokens = *options.MaxTokens
	}

	timeout := a.timeout
	if options != nil && options.Timeout != nil {
		timeout = *options.Timeout
	}

	// Create a timeout context
	timeout_ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Initialize conversation (always create new for now - conversation management can be added later)
	conversation := &Conversation{
		ID:        uuid.New().String(),
		Messages:  []Message{},
		Metadata:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a user message
	user_message := Message{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}

	// Add the message to the conversation
	conversation.Messages = append(conversation.Messages, user_message)
	conversation.UpdatedAt = time.Now()

	// Initialize usage stats
	usage_stats := model.UsageStats{}

	// Start the tool loop
	iterations := 0
	tool_calls_count := 0
	for iterations < a.maxIterations {
		// Create iteration node
		iterationNode, nodeErr := execCtx.CreateChildNode(nil, "iteration", fmt.Sprintf("iteration_%d", iterations+1), map[string]interface{}{
			"iteration_number": iterations + 1,
		})
		if nodeErr == nil {
			_ = iterationNode
		}

		// Check if the context is done
		select {
		case <-timeout_ctx.Done():
			execCtx.SetError(fmt.Errorf("agent execution timed out: %w", timeout_ctx.Err()))
			return AgentResult{}, fmt.Errorf("agent execution timed out: %w", timeout_ctx.Err())
		default:
			// Continue processing
		}

		// Convert the conversation to model messages and build the request
		model_messages := a.buildModelMessages(conversation)

		// Check if streaming is requested and supported
		use_streaming := streamHandler != nil
		request := a.buildCompletionRequest(model_messages, temperature, max_tokens, use_streaming)

		// Create the agent message
		agent_message := Message{
			Role:      "assistant",
			Content:   "",
			Timestamp: time.Now(),
		}

		var response model.CompletionResponse
		var err error

		if use_streaming {
			// Stream the completion from the model
			var tool_calls []model.ToolCall
			err = a.model.CompleteStream(timeout_ctx, request, func(chunk model.StreamedCompletionChunk) error {
				// Update the agent message
				agent_message.Content += chunk.Text

				// Add tool calls if they exist
				if len(chunk.ToolCalls) > 0 {
					tool_calls = chunk.ToolCalls
					agent_message.ToolCalls = tool_calls
				}

				// Call the stream handler if provided
				if streamHandler != nil {
					return streamHandler(agent_message)
				}
				return nil
			})
			if err != nil {
				return AgentResult{}, err
			}

			// Create a response object from the streamed data
			response = model.CompletionResponse{
				Text:       agent_message.Content,
				ToolCalls:  agent_message.ToolCalls,
				UsageStats: model.UsageStats{}, // Streaming doesn't provide usage stats incrementally
			}
		} else {
			// Get a completion from the model (non-streaming)
			response, err = a.model.Complete(timeout_ctx, request)
			if err != nil {
				return AgentResult{}, err
			}

			// Update agent message with response
			agent_message.Content = response.Text
			agent_message.ToolCalls = response.ToolCalls
		}

		// Update usage stats
		usage_stats.PromptTokens += response.UsageStats.PromptTokens
		usage_stats.CompletionTokens += response.UsageStats.CompletionTokens
		usage_stats.TotalTokens += response.UsageStats.TotalTokens

		// Add the agent message to the conversation
		conversation.Messages = append(conversation.Messages, agent_message)
		conversation.UpdatedAt = time.Now()

		// Check if there are tool calls
		if len(agent_message.ToolCalls) == 0 {
			// No tool calls, we're done
			// Record execution end time
			end_time := time.Now()

			// Set output in execution context
			agentcontext.SetOutput(execCtx, agent_message.Content)
			agentcontext.SetData(execCtx, "iterations", iterations+1)
			agentcontext.SetData(execCtx, "tool_calls_count", tool_calls_count)
			agentcontext.SetData(execCtx, "usage_stats", usage_stats)

			// Parse the response text using the agent's parser if one is configured
			var parsed_output map[string]interface{}
			var parse_errors []string
			if a.parser != nil {
				parsed_output, parse_errors = a.parser.Parse(agent_message.Content)
			}

			return AgentResult{
				Output:            agent_message.Content,
				AdditionalOutputs: map[string]interface{}{},
				Conversation:      conversation,
				UsageStats:        usage_stats,
				ExecutionStats: ExecutionStats{
					StartTime:  start_time,
					EndTime:    end_time,
					ToolCalls:  tool_calls_count,
					Iterations: iterations + 1,
				},
				Message:      agent_message,
				ParsedOutput: parsed_output,
				ParseErrors:  parse_errors,
			}, nil
		}

		// Process tool calls
		tool_calls_count += len(agent_message.ToolCalls)
		agentcontext.SetData(execCtx, "current_iteration_tool_calls", len(agent_message.ToolCalls))
		tool_results, err := a.processToolCalls(timeout_ctx, agent_message.ToolCalls)
		if err != nil {
			execCtx.SetError(err)
			return AgentResult{}, err
		}

		// Add the tool results to the agent message
		agent_message.ToolResults = tool_results

		// Add tool results to the conversation
		a.addToolResultsToConversation(conversation, tool_results)

		// Call stream handler for tool results if streaming
		if streamHandler != nil {
			for _, tool_result := range tool_results {
				tool_message := Message{
					Role:      "tool",
					Content:   tool_result.String(),
					Timestamp: time.Now(),
				}
				if err := streamHandler(tool_message); err != nil {
					return AgentResult{}, err
				}
			}
		}

		// Increment the iteration counter
		iterations++
	}

	// We've reached the maximum number of iterations
	return AgentResult{}, fmt.Errorf("reached maximum number of iterations without completing the task")
}
