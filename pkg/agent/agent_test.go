package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hlfshell/gogentic/pkg/model"
)

// mockModel is a mock implementation of model.Model for testing
type mockModel struct {
	info            model.ModelInfo
	completeFunc    func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error)
	completeStreamFunc func(ctx context.Context, request model.CompletionRequest, handler model.StreamHandler) error
}

func (m *mockModel) GetInfo() model.ModelInfo {
	return m.info
}

func (m *mockModel) Complete(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, request)
	}
	return model.CompletionResponse{
		Text: "Mock response",
		UsageStats: model.UsageStats{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:       15,
		},
	}, nil
}

func (m *mockModel) CompleteStream(ctx context.Context, request model.CompletionRequest, handler model.StreamHandler) error {
	if m.completeStreamFunc != nil {
		return m.completeStreamFunc(ctx, request, handler)
	}
	return handler(model.StreamedCompletionChunk{
		Text:      "Mock streamed response",
		IsFinal:   true,
		FinishReason: "stop",
	})
}

func (m *mockModel) SupportsContentType(contentType model.ContentType) bool {
	return contentType == model.TextContent
}

func TestNewToolResult(t *testing.T) {
	// Test with string
	result := NewToolResult("test_tool", "result_value")
	if result == nil {
		t.Fatal("NewToolResult should not return nil")
	}

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetError() != "" {
		t.Errorf("Expected empty error, got %q", result.GetError())
	}

	if result.GetResult() != "result_value" {
		t.Errorf("Expected result 'result_value', got %v", result.GetResult())
	}

	// Test with int
	resultInt := NewToolResult("test_tool", 42)
	if resultInt.GetResult() != 42 {
		t.Errorf("Expected result 42, got %v", resultInt.GetResult())
	}
}

func TestNewToolResultError(t *testing.T) {
	err := context.DeadlineExceeded
	result := NewToolResultError("test_tool", err)

	if result == nil {
		t.Fatal("NewToolResultError should not return nil")
	}

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetError() != err.Error() {
		t.Errorf("Expected error %q, got %q", err.Error(), result.GetError())
	}

	// Test with nil error
	resultNil := NewToolResultError("test_tool", nil)
	if resultNil.GetError() != "" {
		t.Errorf("Expected empty error for nil error, got %q", resultNil.GetError())
	}
}

func TestToolResultString(t *testing.T) {
	// Test with string
	result := ToolResult[string]{
		ToolName: "test_tool",
		Result:   "test_value",
	}
	str := result.String()
	if str != "test_value" {
		t.Errorf("Expected string 'test_value', got %q", str)
	}

	// Test with int
	resultInt := ToolResult[int]{
		ToolName: "test_tool",
		Result:   42,
	}
	strInt := resultInt.String()
	if strInt != "42" {
		t.Errorf("Expected string '42', got %q", strInt)
	}

	// Test with error
	resultErr := ToolResult[string]{
		ToolName: "test_tool",
		Error:    "test error",
	}
	strErr := resultErr.String()
	if strErr != "test error" {
		t.Errorf("Expected string 'test error', got %q", strErr)
	}

	// Test with struct
	type TestStruct struct {
		Name  string
		Value int
	}
	resultStruct := ToolResult[TestStruct]{
		ToolName: "test_tool",
		Result:   TestStruct{Name: "test", Value: 42},
	}
	strStruct := resultStruct.String()
	if strStruct == "" {
		t.Fatal("String should not be empty for struct")
	}
	// Should be JSON
	var parsed TestStruct
	if err := json.Unmarshal([]byte(strStruct), &parsed); err != nil {
		t.Errorf("String should be valid JSON: %v", err)
	}
}

func TestToolResultMarshalJSON(t *testing.T) {
	result := ToolResult[string]{
		ToolName: "test_tool",
		Result:   "test_value",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var aux map[string]interface{}
	if err := json.Unmarshal(data, &aux); err != nil {
		t.Fatalf("Unmarshaled data should be valid JSON: %v", err)
	}

	if aux["tool_name"] != "test_tool" {
		t.Errorf("Expected tool_name 'test_tool', got %v", aux["tool_name"])
	}

	if aux["result"] != "test_value" {
		t.Errorf("Expected result 'test_value', got %v", aux["result"])
	}

	// Test with error
	resultErr := ToolResult[string]{
		ToolName: "test_tool",
		Error:    "test error",
	}

	dataErr, err := json.Marshal(resultErr)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var auxErr map[string]interface{}
	if err := json.Unmarshal(dataErr, &auxErr); err != nil {
		t.Fatalf("Unmarshaled data should be valid JSON: %v", err)
	}

	if auxErr["error"] != "test error" {
		t.Errorf("Expected error 'test error', got %v", auxErr["error"])
	}
}

func TestToolResultUnmarshalJSON(t *testing.T) {
	data := []byte(`{"tool_name":"test_tool","result":"test_value"}`)

	var result ToolResult[string]
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if result.ToolName != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.ToolName)
	}

	if result.Result != "test_value" {
		t.Errorf("Expected result 'test_value', got %q", result.Result)
	}

	// Test with error
	dataErr := []byte(`{"tool_name":"test_tool","error":"test error"}`)

	var resultErr ToolResult[string]
	if err := json.Unmarshal(dataErr, &resultErr); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if resultErr.Error != "test error" {
		t.Errorf("Expected error 'test error', got %q", resultErr.Error)
	}
}

func TestToolResultToJSON(t *testing.T) {
	result := ToolResult[string]{
		ToolName: "test_tool",
		Result:   "test_value",
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("ToJSON should return valid JSON: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got %q", value)
	}

	// Test with error
	resultErr := ToolResult[string]{
		ToolName: "test_tool",
		Error:    "test error",
	}

	dataErr, err := resultErr.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var errMap map[string]string
	if err := json.Unmarshal(dataErr, &errMap); err != nil {
		t.Fatalf("ToJSON should return valid JSON: %v", err)
	}

	if errMap["error"] != "test error" {
		t.Errorf("Expected error 'test error', got %q", errMap["error"])
	}
}

func TestNewGenericToolHandler(t *testing.T) {
	handler := NewGenericToolHandler("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	})

	if handler == nil {
		t.Fatal("NewGenericToolHandler should not return nil")
	}

	// Test calling the handler
	result, err := handler.Call(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler.Call failed: %v", err)
	}

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetResult() != "result" {
		t.Errorf("Expected result 'result', got %v", result.GetResult())
	}
}

func TestGenericToolHandlerWithError(t *testing.T) {
	testErr := context.DeadlineExceeded
	handler := NewGenericToolHandler("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "", testErr
	})

	result, err := handler.Call(context.Background(), map[string]interface{}{})
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	if result.GetError() != testErr.Error() {
		t.Errorf("Expected error string %q, got %q", testErr.Error(), result.GetError())
	}
}

func TestNewStringToolHandler(t *testing.T) {
	handler := NewStringToolHandler("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	})

	if handler == nil {
		t.Fatal("NewStringToolHandler should not return nil")
	}

	// Test calling the handler
	result, err := handler.Call(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler.Call failed: %v", err)
	}

	if result.GetToolName() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", result.GetToolName())
	}

	if result.GetResult() != "result" {
		t.Errorf("Expected result 'result', got %v", result.GetResult())
	}
}

func TestStringToolHandlerWithError(t *testing.T) {
	testErr := context.DeadlineExceeded
	handler := NewStringToolHandler("test_tool", func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "", testErr
	})

	result, err := handler.Call(context.Background(), map[string]interface{}{})
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	if result.GetError() != testErr.Error() {
		t.Errorf("Expected error string %q, got %q", testErr.Error(), result.GetError())
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
		SystemPrompt: "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test agent description", config)

	if agent == nil {
		t.Fatal("NewBaseAgent should not return nil")
	}

	if agent.ID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got %q", agent.ID())
	}

	if agent.Name() != "test-agent" {
		t.Errorf("Expected name 'test-agent', got %q", agent.Name())
	}

	if agent.Description() != "Test agent description" {
		t.Errorf("Expected description 'Test agent description', got %q", agent.Description())
	}

	if agent.Config().Model != mockModel {
		t.Error("Config should contain the provided model")
	}

	if agent.GetParser() == nil {
		t.Fatal("Parser should not be nil")
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

	// Test with empty ID (should generate UUID)
	agent := NewBaseAgent("", "test-agent", "Test description", config)
	if agent.ID() == "" {
		t.Fatal("Agent should have an ID even if empty string provided")
	}

	// Test with zero values (should use defaults)
	agent2 := NewBaseAgent("test-id", "test-agent", "Test description", AgentConfig{
		Model: mockModel,
	})

	if agent2.Config().MaxTokens <= 0 {
		t.Error("MaxTokens should have a default value")
	}

	if agent2.Config().Temperature <= 0 {
		t.Error("Temperature should have a default value")
	}

	if agent2.Config().MaxIterations <= 0 {
		t.Error("MaxIterations should have a default value")
	}

	if agent2.Config().Timeout <= 0 {
		t.Error("Timeout should have a default value")
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
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature:  0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	params := AgentParameters{
		Input: "Test input",
	}

	ctx := context.Background()
	result, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Output != "Test response" {
		t.Errorf("Expected output 'Test response', got %q", result.Output)
	}

	if result.Conversation == nil {
		t.Fatal("Conversation should not be nil")
	}

	if len(result.Conversation.Messages) != 2 {
		t.Errorf("Expected 2 messages (user + assistant), got %d", len(result.Conversation.Messages))
	}

	if result.Conversation.Messages[0].Role != "user" {
		t.Errorf("Expected first message role 'user', got %q", result.Conversation.Messages[0].Role)
	}

	if result.Conversation.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message role 'assistant', got %q", result.Conversation.Messages[1].Role)
	}

	if result.UsageStats.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", result.UsageStats.TotalTokens)
	}

	if result.ExecutionStats.Iterations != 1 {
		t.Errorf("Expected iterations 1, got %d", result.ExecutionStats.Iterations)
	}
}

func TestBaseAgentExecuteWithExistingConversation(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Verify conversation history is included
			if len(request.Messages) < 2 {
				t.Errorf("Expected at least 2 messages in request, got %d", len(request.Messages))
			}
			return model.CompletionResponse{
				Text: "Test response",
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:       mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	existingConv := &Conversation{
		ID:        "existing-conv-id",
		Messages:  []Message{
			{
				Role:      "user",
				Content:   "Previous message",
				Timestamp: time.Now(),
			},
		},
		Metadata:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	params := AgentParameters{
		Input:       "New input",
		Conversation: existingConv,
	}

	ctx := context.Background()
	result, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Conversation.ID != "existing-conv-id" {
		t.Errorf("Expected conversation ID 'existing-conv-id', got %q", result.Conversation.ID)
	}

	if len(result.Conversation.Messages) != 3 {
		t.Errorf("Expected 3 messages (previous + user + assistant), got %d", len(result.Conversation.Messages))
	}
}

func TestBaseAgentExecuteWithOptions(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Verify options are applied
			if request.Temperature != 0.5 {
				t.Errorf("Expected temperature 0.5, got %f", request.Temperature)
			}
			if request.MaxTokens != 500 {
				t.Errorf("Expected max tokens 500, got %d", request.MaxTokens)
			}
			return model.CompletionResponse{
				Text: "Test response",
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature: 0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	temp := float32(0.5)
	maxTokens := 500
	params := AgentParameters{
		Input: "Test input",
		Options: AgentOptions{
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		},
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestBaseAgentExecuteWithTools(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Verify tools are included
			if len(request.Tools) != 1 {
				t.Errorf("Expected 1 tool in request, got %d", len(request.Tools))
			}
			if request.Tools[0].Name != "test_tool" {
				t.Errorf("Expected tool name 'test_tool', got %q", request.Tools[0].Name)
			}
			return model.CompletionResponse{
				Text: "Test response",
				ToolCalls: []model.ToolCall{
					{
						Name:      "test_tool",
						Arguments: map[string]interface{}{"arg1": "value1"},
					},
				},
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature:  0.7,
		Tools: []Tool{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  map[string]interface{}{"arg1": map[string]interface{}{"type": "string"}},
			},
		},
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	params := AgentParameters{
		Input: "Test input",
	}

	ctx := context.Background()
	result, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.Message.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(result.Message.ToolCalls))
	}

	if result.ExecutionStats.ToolCalls != 1 {
		t.Errorf("Expected tool calls count 1, got %d", result.ExecutionStats.ToolCalls)
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
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature:  0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	params := AgentParameters{
		Input: "Test input",
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, params)
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

func TestBaseAgentExecuteCreatesExecutionContext(t *testing.T) {
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
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature:  0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	params := AgentParameters{
		Input: "Test input",
	}

	ctx := context.Background()
	result, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify execution context was created and used
	// The agent should have created an execution node
	// We can verify this by checking that the result has proper structure
	if result.Output == "" {
		t.Fatal("Result should have output")
	}
}

func TestBaseAgentExecuteWithTimeout(t *testing.T) {
	mockModel := &mockModel{
		info: model.ModelInfo{
			Name:     "test-model",
			Provider: "test",
		},
		completeFunc: func(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
			// Check if context has timeout
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("Context should have a deadline when timeout is set")
			}
			if deadline.IsZero() {
				t.Error("Deadline should not be zero")
			}
			return model.CompletionResponse{
				Text: "Test response",
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature:  0.7,
		Timeout:      30 * time.Second,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	params := AgentParameters{
		Input: "Test input",
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
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
			if len(request.Messages) == 0 {
				t.Fatal("Request should have at least one message")
			}
			if request.Messages[0].Role != "system" {
				t.Errorf("Expected first message role 'system', got %q", request.Messages[0].Role)
			}
			if len(request.Messages[0].Content) == 0 {
				t.Fatal("System message should have content")
			}
			if request.Messages[0].Content[0].Text != "You are a test agent" {
				t.Errorf("Expected system prompt 'You are a test agent', got %q", request.Messages[0].Content[0].Text)
			}
			return model.CompletionResponse{
				Text: "Test response",
				UsageStats: model.UsageStats{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:       15,
				},
			}, nil
		},
	}

	config := AgentConfig{
		Model:        mockModel,
		SystemPrompt: "You are a test agent",
		MaxTokens:    1000,
		Temperature:  0.7,
	}

	agent := NewBaseAgent("test-id", "test-agent", "Test description", config)

	params := AgentParameters{
		Input: "Test input",
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

