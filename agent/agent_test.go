package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hlfshell/gogentic/pkg/model"
)

// mockModel is a mock implementation of model.Model for testing
type mockModel struct {
	info         model.ModelInfo
	completeFunc func(context.Context, model.CompletionRequest) (model.CompletionResponse, error)
	streamFunc   func(context.Context, model.CompletionRequest, model.StreamHandler) error
}

func (m *mockModel) GetInfo() model.ModelInfo {
	return m.info
}

func (m *mockModel) Complete(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, request)
	}
	return model.CompletionResponse{}, nil
}

func (m *mockModel) CompleteStream(ctx context.Context, request model.CompletionRequest, handler model.StreamHandler) error {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, request, handler)
	}
	return nil
}

func (m *mockModel) SupportsContentType(contentType model.ContentType) bool {
	return true // Mock supports all content types
}

func TestNewToolResult(t *testing.T) {
	result := NewToolResult("test_tool", "test result")

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetResult() != "test result" {
		t.Errorf("Expected result 'test result', got %q", result.GetResult())
	}

	if result.GetError() != "" {
		t.Errorf("Expected no error, got %q", result.GetError())
	}
}

func TestNewToolResultError(t *testing.T) {
	testErr := errors.New("test error")
	result := NewToolResultError("test_tool", testErr)

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetError() != "test error" {
		t.Errorf("Expected error 'test error', got %q", result.GetError())
	}
}

func TestToolResultString(t *testing.T) {
	// Test string result
	stringResult := NewToolResult("test_tool", "simple string")
	if stringResult.String() != "simple string" {
		t.Errorf("Expected 'simple string', got %q", stringResult.String())
	}

	// Test int result
	intResult := NewToolResult("test_tool", 42)
	if intResult.String() != "42" {
		t.Errorf("Expected '42', got %q", intResult.String())
	}

	// Test complex result
	complexResult := NewToolResult("test_tool", map[string]interface{}{"key": "value"})
	expected := `{"key":"value"}`
	if complexResult.String() != expected {
		t.Errorf("Expected %q, got %q", expected, complexResult.String())
	}

	// Test error result
	errResult := NewToolResultError("test_tool", errors.New("error message"))
	if errResult.String() != "error message" {
		t.Errorf("Expected 'error message', got %q", errResult.String())
	}
}

func TestToolResultMarshalJSON(t *testing.T) {
	result := NewToolResult("test_tool", map[string]string{"key": "value"})
	jsonBytes, err := result.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded["tool_name"] != "test_tool" {
		t.Errorf("Expected tool_name 'test_tool', got %v", decoded["tool_name"])
	}
}

func TestToolResultUnmarshalJSON(t *testing.T) {
	jsonStr := `{"tool_name":"test_tool","result":"test value","error":""}`
	var result Result[string]

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if result.ToolName != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.ToolName)
	}

	if result.Result != "test value" {
		t.Errorf("Expected result 'test value', got %q", result.Result)
	}
}

func TestToolResultToJSON(t *testing.T) {
	result := NewToolResult("test_tool", "test value")
	jsonBytes, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var decoded string
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded != "test value" {
		t.Errorf("Expected 'test value', got %q", decoded)
	}
}

func TestNewGenericToolHandler(t *testing.T) {
	handler := NewGenericToolHandler("test_tool", "A test tool", map[string]interface{}{}, func(ctx context.Context, params Arguments) (int, error) {
		return 42, nil
	})

	ctx := context.Background()
	result, err := handler.Execute(ctx, Arguments{})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetResult() != 42 {
		t.Errorf("Expected result 42, got %v", result.GetResult())
	}
}

func TestGenericToolHandlerWithError(t *testing.T) {
	testErr := errors.New("test error")
	handler := NewGenericToolHandler("test_tool", "A test tool", map[string]interface{}{}, func(ctx context.Context, params Arguments) (string, error) {
		return "", testErr
	})

	ctx := context.Background()
	result, err := handler.Execute(ctx, Arguments{})

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	if result.GetError() != "test error" {
		t.Errorf("Expected error 'test error', got %q", result.GetError())
	}
}

func TestNewBaseAgent(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	if agent.ID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got %q", agent.ID())
	}

	if agent.Name() != "test-agent" {
		t.Errorf("Expected name 'test-agent', got %q", agent.Name())
	}

	if agent.Description() != "Test description" {
		t.Errorf("Expected description 'Test description', got %q", agent.Description())
	}
}

func TestNewBaseAgentWithDefaults(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
	}

	config := AgentConfig{
		Model: mockModel,
	}

	agent := NewAgent("", "test-agent", "Test description", config)

	// Check defaults
	if agent.Config().MaxTokens != 1000 {
		t.Errorf("Expected default MaxTokens 1000, got %d", agent.Config().MaxTokens)
	}

	if agent.Config().Temperature != 0.7 {
		t.Errorf("Expected default Temperature 0.7, got %f", agent.Config().Temperature)
	}

	if agent.Config().MaxIterations != 5 {
		t.Errorf("Expected default MaxIterations 5, got %d", agent.Config().MaxIterations)
	}

	if agent.Config().Timeout != 60*time.Second {
		t.Errorf("Expected default Timeout 60s, got %v", agent.Config().Timeout)
	}
}

func TestBaseAgentExecute(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			return model.CompletionResponse{
				Text: "Test response",
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	params := Arguments{
		"input": "Test input",
	}

	ctx := context.Background()
	result, err := agent.ExecuteAgent(ctx, params, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Output != "Test response" {
		t.Errorf("Expected output 'Test response', got %q", result.Output)
	}

	if result.UsageStats.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", result.UsageStats.TotalTokens)
	}
}

func TestBaseAgentExecuteWithExistingConversation(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Verify that we have 2 messages: system and new user message
			// (Conversations are now always created fresh)
			if len(request.Messages) != 2 {
				t.Errorf("Expected 2 messages, got %d", len(request.Messages))
			}
			return model.CompletionResponse{
				Text: "Response to message",
			}, nil
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	params := Arguments{
		"input": "Test message",
	}
	// Note: Conversation management is now handled separately - conversations are always created fresh

	ctx := context.Background()
	result, err := agent.ExecuteAgent(ctx, params, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should have 2 messages: user message and assistant response
	if len(result.Conversation.Messages) != 2 {
		t.Errorf("Expected 2 messages in conversation, got %d", len(result.Conversation.Messages))
	}
}

func TestBaseAgentExecuteWithOptions(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Verify that options were applied
			if request.Temperature != 0.9 {
				t.Errorf("Expected temperature 0.9, got %f", request.Temperature)
			}
			if request.MaxTokens != 500 {
				t.Errorf("Expected max tokens 500, got %d", request.MaxTokens)
			}
			return model.CompletionResponse{
				Text: "Response",
			}, nil
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	temp := float32(0.9)
	maxTokens := 500
	params := Arguments{
		"input": "Test input",
	}
	options := &AgentOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	ctx := context.Background()
	_, err := agent.ExecuteAgent(ctx, params, options)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestBaseAgentExecuteWithTools(t *testing.T) {
	callCount := 0
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			callCount++

			// Verify tools are included
			if len(request.Tools) != 1 {
				t.Errorf("Expected 1 tool in request, got %d", len(request.Tools))
			}
			if request.Tools[0].Name != "test_tool" {
				t.Errorf("Expected tool name 'test_tool', got %q", request.Tools[0].Name)
			}

			// First call: return a tool call
			// Second call (after tool execution): return final response with no tool calls
			if callCount == 1 {
				return model.CompletionResponse{
					Text: "Calling test tool",
					ToolCalls: []model.ToolCall{
						{
							Name:      "test_tool",
							Arguments: map[string]interface{}{"arg1": "value1"},
						},
					},
					UsageStats: model.UsageStats{
						PromptTokens:     10,
						CompletionTokens: 5,
						TotalTokens:      15,
					},
				}, nil
			}

			// Second call after tool execution
			return model.CompletionResponse{
				Text:      "Final response after tool execution",
				ToolCalls: []model.ToolCall{},
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			}, nil
		},
	}

	// Add a tool handler so the tool can actually be executed
	toolCalled := false

	// Create a tool using NewGenericToolHandler
	tool := NewGenericToolHandler("test_tool", "A test tool", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"arg1": map[string]interface{}{"type": "string"},
		},
	}, func(ctx context.Context, params Arguments) (string, error) {
		toolCalled = true
		return "tool result", nil
	})

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
		Tools:       []GotonomyTool{tool},
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	params := Arguments{
		"input": "Test input",
	}

	ctx := context.Background()
	result, err := agent.ExecuteAgent(ctx, params, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !toolCalled {
		t.Error("Expected tool to be called")
	}

	if result.ExecutionStats.ToolCalls != 1 {
		t.Errorf("Expected tool calls count 1, got %d", result.ExecutionStats.ToolCalls)
	}

	if result.ExecutionStats.Iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", result.ExecutionStats.Iterations)
	}

	if result.Output != "Final response after tool execution" {
		t.Errorf("Expected final output, got %q", result.Output)
	}
}

func TestBaseAgentExecuteWithModelError(t *testing.T) {
	testErr := context.DeadlineExceeded
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			return model.CompletionResponse{}, testErr
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	params := Arguments{
		"input": "Test input",
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, params)
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

func TestBaseAgentExecuteWithSystemPrompt(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Verify system prompt is included
			if len(request.Messages) < 2 {
				t.Error("Expected at least 2 messages (system + user)")
			}
			if request.Messages[0].Role != "system" {
				t.Errorf("Expected first message to be system, got %q", request.Messages[0].Role)
			}
			if request.Messages[0].Content[0].Text != "You are a helpful assistant" {
				t.Errorf("Expected system prompt 'You are a helpful assistant', got %q", request.Messages[0].Content[0].Text)
			}
			return model.CompletionResponse{
				Text: "Response",
			}, nil
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a helpful assistant",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewAgent("test-id", "test-agent", "Test description", config)

	params := Arguments{
		"input": "Test input",
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestAgentAsToolCall(t *testing.T) {
	// Create a simple agent that will act as a tool
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			return model.CompletionResponse{
				Text: "Math result: 42",
				UsageStats: model.UsageStats{
					PromptTokens:     5,
					CompletionTokens: 3,
					TotalTokens:      8,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		Prompt:      "You are a math solver",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	mathAgent := NewAgent("math-agent", "Math Solver", "Solves math problems", config)

	// Call the agent as a tool using the Call method
	ctx := context.Background()
	params := Arguments{
		"input":     "What is 2+2?",
		"precision": "high",
	}

	result, err := mathAgent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.GetToolName() != "Math Solver" {
		t.Errorf("Expected tool name 'Math Solver', got %q", result.GetToolName())
	}

	if result.GetError() != "" {
		t.Errorf("Expected no error, got %q", result.GetError())
	}

	// The result should be an AgentResult
	agentResult, ok := result.GetResult().(AgentResult)
	if !ok {
		t.Fatalf("Expected AgentResult, got %T", result.GetResult())
	}

	if agentResult.Output != "Math result: 42" {
		t.Errorf("Expected output 'Math result: 42', got %q", agentResult.Output)
	}
}

func TestAgentAsToolInAnotherAgent(t *testing.T) {
	callCount := 0

	// Create a math agent
	mathModel := &mockModel{
		info: model.ModelInfo{
			Name:     "math-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			return model.CompletionResponse{
				Text: "The answer is 42",
				UsageStats: model.UsageStats{
					PromptTokens:     5,
					CompletionTokens: 3,
					TotalTokens:      8,
				},
			}, nil
		},
	}

	mathAgent := NewAgent("math-solver", "Math Solver", "Solves mathematical problems", AgentConfig{
		Model:       mathModel,
		Prompt:      "You are a math solver",
		MaxTokens:   100,
		Temperature: 0.7,
	})

	// Create a coordinator agent that uses the math agent as a tool
	coordinatorModel := &mockModel{
		info: model.ModelInfo{
			Name:     "coordinator-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			callCount++

			// Verify the math agent tool is available
			if len(request.Tools) != 1 {
				t.Errorf("Expected 1 tool, got %d", len(request.Tools))
			}

			if request.Tools[0].Name != "Math Solver" {
				t.Errorf("Expected tool name 'Math Solver', got %q", request.Tools[0].Name)
			}

			// First call: use the math tool
			if callCount == 1 {
				return model.CompletionResponse{
					Text: "I'll use the math tool",
					ToolCalls: []model.ToolCall{
						{
							Name: "Math Solver",
							Arguments: map[string]interface{}{
								"input": "What is 21 * 2?",
							},
						},
					},
					UsageStats: model.UsageStats{
						PromptTokens:     10,
						CompletionTokens: 5,
						TotalTokens:      15,
					},
				}, nil
			}

			// Second call: provide final answer after tool execution
			return model.CompletionResponse{
				Text: "Based on the math tool, the final answer is 42",
				UsageStats: model.UsageStats{
					PromptTokens:     20,
					CompletionTokens: 10,
					TotalTokens:      30,
				},
			}, nil
		},
	}

	// Create the coordinator with the math agent as a tool (agents are tools directly!)
	coordinatorAgent := NewAgent("coordinator", "Coordinator", "Coordinates tasks", AgentConfig{
		Model:       coordinatorModel,
		Prompt:      "You are a coordinator that can use other agents",
		MaxTokens:   200,
		Temperature: 0.7,
		Tools:       []GotonomyTool{mathAgent}, // Agents implement Tool directly!
	})

	// Execute the coordinator
	ctx := context.Background()
	result, err := coordinatorAgent.ExecuteAgent(ctx, Arguments{
		"input": "Help me solve a math problem",
	}, nil)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", callCount)
	}

	if result.ExecutionStats.ToolCalls != 1 {
		t.Errorf("Expected 1 tool call, got %d", result.ExecutionStats.ToolCalls)
	}

	if result.ExecutionStats.Iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", result.ExecutionStats.Iterations)
	}

	if result.Output != "Based on the math tool, the final answer is 42" {
		t.Errorf("Unexpected output: %q", result.Output)
	}
}

func TestAgentIsTool(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			return model.CompletionResponse{
				Text: "Response",
			}, nil
		},
	}

	agent := NewAgent("test-agent", "Test Agent", "A test agent", AgentConfig{
		Model:       mockModel,
		MaxTokens:   100,
		Temperature: 0.7,
	})

	// Verify agent implements Tool interface (agents are tools directly!)
	var tool GotonomyTool = agent

	// Verify tool properties
	if tool.Name() != "Test Agent" {
		t.Errorf("Expected tool name 'Test Agent', got %q", tool.Name())
	}

	if tool.Description() != "A test agent" {
		t.Errorf("Expected description 'A test agent', got %q", tool.Description())
	}

	// Verify parameters schema
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", params["type"])
	}
}

func TestAgentCallWithAdditionalInputs(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			return model.CompletionResponse{
				Text: "Processed with additional inputs",
			}, nil
		},
	}

	agent := NewAgent("test-agent", "Test Agent", "A test agent", AgentConfig{
		Model:       mockModel,
		MaxTokens:   100,
		Temperature: 0.7,
	})

	ctx := context.Background()
	params := Arguments{
		"input":     "Primary input",
		"format":    "json",
		"precision": 2,
		"verbose":   true,
	}

	result, err := agent.ExecuteAgent(ctx, params, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// Verify additional inputs were passed through
	if result.AdditionalOutputs == nil {
		t.Error("Expected AdditionalOutputs to be initialized")
	}
}
