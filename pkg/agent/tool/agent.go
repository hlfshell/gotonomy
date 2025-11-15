// Package tool provides a tool-using agent implementation.
package tool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hlfshell/gogentic/pkg/model"

	"github.com/hlfshell/gogentic/pkg/agent"
)

// ToolAgent is an implementation of the Agent interface that can use
// tools to accomplish tasks.
type ToolAgent struct {
	*agent.BaseAgent
}

// NewToolAgent creates a new tool agent with the given configuration.
func NewToolAgent(id, name, description string, config agent.AgentConfig) *ToolAgent {
	// Set default values if not provided
	if config.MaxIterations <= 0 {
		config.MaxIterations = 10
	}

	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}

	// Create the base agent
	base_agent := agent.NewBaseAgent(id, name, description, config)

	// Create and return the tool agent
	return &ToolAgent{
		BaseAgent: base_agent,
	}
}

// buildModelMessages converts a conversation to model messages.
func (a *ToolAgent) buildModelMessages(conversation *agent.Conversation) []model.Message {
	model_messages := []model.Message{}

	// Add the system prompt if it exists
	config := a.BaseAgent.Config()
	if config.SystemPrompt != "" {
		model_messages = append(model_messages, model.Message{
			Role: "system",
			Content: []model.Content{
				{
					Type: model.TextContent,
					Text: config.SystemPrompt,
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

	return model_messages
}

// buildCompletionRequest builds a completion request from model messages and options.
func (a *ToolAgent) buildCompletionRequest(model_messages []model.Message, temperature float32, max_tokens int, stream bool) model.CompletionRequest {
	request := model.CompletionRequest{
		Messages:       model_messages,
		Temperature:    temperature,
		MaxTokens:      max_tokens,
		StreamResponse: stream,
	}

	// Add tools if they exist
	config := a.BaseAgent.Config()
	if len(config.Tools) > 0 {
		model_tools := []model.Tool{}
		for _, tool := range config.Tools {
			model_tool := model.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			}
			model_tools = append(model_tools, model_tool)
		}
		request.Tools = model_tools
	}

	return request
}

// processToolCalls processes a list of tool calls and returns the results.
func (a *ToolAgent) processToolCalls(ctx context.Context, tool_calls []model.ToolCall) ([]agent.ToolResultInterface, error) {
	tool_results := []agent.ToolResultInterface{}

	// Get ExecutionContext if available
	execCtx, hasExecCtx := agent.AsExecutionContext(ctx)

	config := a.Config()
	for _, tool_call := range tool_calls {
		// Create tool call node if ExecutionContext is available
		var toolNode *agent.Node
		if hasExecCtx {
			var err error
			toolNode, err = execCtx.CreateChildNode("tool", tool_call.Name, tool_call.Arguments)
			if err == nil {
				_ = toolNode
			}
		}

		// Find the tool
		var tool agent.Tool
		found := false
		for _, t := range config.Tools {
			if t.Name == tool_call.Name {
				tool = t
				found = true
				break
			}
		}

		if !found {
			// Tool not found
			tool_result := agent.NewToolResultError(tool_call.Name, fmt.Errorf("tool not found: %s", tool_call.Name))
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				execCtx.SetError(fmt.Errorf("tool not found: %s", tool_call.Name))
			}
			continue
		}

		// Call the tool - handle both legacy ToolHandler and new ToolHandlerInterface
		var tool_result agent.ToolResultInterface
		var err error

		switch handler := tool.Handler.(type) {
		case agent.ToolHandlerInterface:
			// New generic handler
			tool_result, err = handler.Call(ctx, tool_call.Arguments)
		case agent.ToolHandler:
			// Legacy string handler - wrap it
			stringHandler := agent.NewStringToolHandler(tool_call.Name, handler)
			tool_result, err = stringHandler.Call(ctx, tool_call.Arguments)
		default:
			// Unknown handler type
			tool_result = agent.NewToolResultError(tool_call.Name, fmt.Errorf("unknown handler type"))
			err = fmt.Errorf("unknown handler type")
		}

		if err != nil {
			// Tool call failed - tool_result should already have error set, but ensure it does
			if tool_result == nil {
				tool_result = agent.NewToolResultError(tool_call.Name, err)
			}
			tool_results = append(tool_results, tool_result)
			if hasExecCtx {
				execCtx.SetError(err)
			}
			continue
		}

		// Tool call succeeded - set output in execution context
		if hasExecCtx {
			agent.SetOutput(execCtx, tool_result.String())
			agent.SetData(execCtx, "tool_result", tool_result.GetResult())
		}

		// Tool call succeeded
		tool_results = append(tool_results, tool_result)
	}

	return tool_results, nil
}

// addToolResultsToConversation adds tool results as messages to the conversation.
func (a *ToolAgent) addToolResultsToConversation(conversation *agent.Conversation, tool_results []agent.ToolResultInterface) {
	for _, tool_result := range tool_results {
		tool_message := agent.Message{
			Role:      "tool",
			Content:   tool_result.String(),
			Timestamp: time.Now(),
		}
		conversation.Messages = append(conversation.Messages, tool_message)
	}
	conversation.UpdatedAt = time.Now()
}

// Execute processes the given parameters and returns a result.
// This implementation adds a tool loop for tool usage.
func (a *ToolAgent) Execute(ctx context.Context, params agent.AgentParameters) (agent.AgentResult, error) {
	// Get or create ExecutionContext
	execCtx := agent.GetOrCreateExecutionContext(ctx)
	ctx = execCtx // Use ExecutionContext as the context going forward

	// Create agent execution node
	agentNode, err := execCtx.CreateChildNode("agent", a.BaseAgent.Name(), map[string]interface{}{
		"input":      params.Input,
		"agent_id":   a.BaseAgent.ID(),
		"agent_type": "tool",
	})
	if err != nil {
		return agent.AgentResult{}, fmt.Errorf("failed to create agent node: %w", err)
	}
	_ = agentNode

	// Set execution-level data
	agent.SetExecutionData(execCtx, "agent_id", a.BaseAgent.ID())
	agent.SetExecutionData(execCtx, "agent_name", a.BaseAgent.Name())
	agent.SetExecutionData(execCtx, "agent_type", "tool")

	// Record execution start time
	start_time := time.Now()

	config := a.BaseAgent.Config()

	// Apply options if provided
	temperature := config.Temperature
	if params.Options.Temperature != nil {
		temperature = *params.Options.Temperature
	}

	max_tokens := config.MaxTokens
	if params.Options.MaxTokens != nil {
		max_tokens = *params.Options.MaxTokens
	}

	timeout := config.Timeout
	if params.Options.Timeout != nil {
		timeout = *params.Options.Timeout
	}

	// Create a timeout context
	timeout_ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Initialize or use provided conversation
	var conversation *agent.Conversation
	if params.Conversation != nil {
		conversation = params.Conversation
	} else {
		conversation = &agent.Conversation{
			ID:        uuid.New().String(),
			Messages:  []agent.Message{},
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Create a user message
	user_message := agent.Message{
		Role:      "user",
		Content:   params.Input,
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
	for iterations < config.MaxIterations {
		// Create iteration node
		iterationNode, nodeErr := execCtx.CreateChildNode("iteration", fmt.Sprintf("iteration_%d", iterations+1), map[string]interface{}{
			"iteration_number": iterations + 1,
		})
		if nodeErr == nil {
			_ = iterationNode
		}

		// Check if the context is done
		select {
		case <-timeout_ctx.Done():
			execCtx.SetError(fmt.Errorf("agent execution timed out: %w", timeout_ctx.Err()))
			return agent.AgentResult{}, fmt.Errorf("agent execution timed out: %w", timeout_ctx.Err())
		default:
			// Continue processing
		}

		// Convert the conversation to model messages and build the request
		model_messages := a.buildModelMessages(conversation)

		// Check if streaming is requested and supported
		use_streaming := params.StreamHandler != nil
		request := a.buildCompletionRequest(model_messages, temperature, max_tokens, use_streaming)

		// Create the agent message
		agent_message := agent.Message{
			Role:      "assistant",
			Content:   "",
			Timestamp: time.Now(),
		}

		var response model.CompletionResponse
		var err error

		if use_streaming {
			// Stream the completion from the model
			var tool_calls []model.ToolCall
			err = config.Model.CompleteStream(timeout_ctx, request, func(chunk model.StreamedCompletionChunk) error {
				// Update the agent message
				agent_message.Content += chunk.Text

				// Add tool calls if they exist
				if len(chunk.ToolCalls) > 0 {
					tool_calls = chunk.ToolCalls
					agent_message.ToolCalls = tool_calls
				}

				// Call the stream handler if provided
				if params.StreamHandler != nil {
					return params.StreamHandler(agent_message)
				}
				return nil
			})
			if err != nil {
				return agent.AgentResult{}, err
			}

			// Create a response object from the streamed data
			response = model.CompletionResponse{
				Text:       agent_message.Content,
				ToolCalls:  agent_message.ToolCalls,
				UsageStats: model.UsageStats{}, // Streaming doesn't provide usage stats incrementally
			}
		} else {
			// Get a completion from the model (non-streaming)
			response, err = config.Model.Complete(timeout_ctx, request)
			if err != nil {
				return agent.AgentResult{}, err
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
			agent.SetOutput(execCtx, agent_message.Content)
			agent.SetData(execCtx, "iterations", iterations+1)
			agent.SetData(execCtx, "tool_calls_count", tool_calls_count)
			agent.SetData(execCtx, "usage_stats", usage_stats)

			// Parse the response text using the agent's parser
			parsed_output, parse_errors := a.BaseAgent.GetParser().Parse(agent_message.Content)

			return agent.AgentResult{
				Output:            agent_message.Content,
				AdditionalOutputs: map[string]interface{}{},
				Conversation:      conversation,
				UsageStats:        usage_stats,
				ExecutionStats: agent.ExecutionStats{
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
		agent.SetData(execCtx, "current_iteration_tool_calls", len(agent_message.ToolCalls))
		tool_results, err := a.processToolCalls(timeout_ctx, agent_message.ToolCalls)
		if err != nil {
			execCtx.SetError(err)
			return agent.AgentResult{}, err
		}

		// Add the tool results to the agent message
		agent_message.ToolResults = tool_results

		// Add tool results to the conversation
		a.addToolResultsToConversation(conversation, tool_results)

		// Call stream handler for tool results if streaming
		if params.StreamHandler != nil {
			for _, tool_result := range tool_results {
				tool_message := agent.Message{
					Role:      "tool",
					Content:   tool_result.String(),
					Timestamp: time.Now(),
				}
				if err := params.StreamHandler(tool_message); err != nil {
					return agent.AgentResult{}, err
				}
			}
		}

		// Increment the iteration counter
		iterations++
	}

	// We've reached the maximum number of iterations
	return agent.AgentResult{}, errors.New("reached maximum number of iterations without completing the task")
}
