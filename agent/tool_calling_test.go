package agent

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
	"github.com/hlfshell/gotonomy/utils/semver"
)

// mockTool is a simple tool implementation for testing
type mockTool struct {
	id          string
	name        string
	description string
	params      []tool.Parameter
	executeFunc func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface
}

func (m *mockTool) ID() string {
	if m.id != "" {
		return m.id
	}
	return "mock/" + m.name
}

func (m *mockTool) Version() semver.SemVer {
	return semver.SemVer{}
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Parameters() []tool.Parameter {
	return m.params
}

func (m *mockTool) Execute(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return tool.NewOK("default result")
}

func newMockTool(name string, executeFunc func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface) tool.Tool {
	return &mockTool{
		id:          "mock/" + name,
		name:        name,
		description: "mock tool for testing",
		params:      []tool.Parameter{},
		executeFunc: executeFunc,
	}
}

func createTestAgent(tools map[string]tool.Tool, config AgentConfig) *Agent {
	return &Agent{
		name:        "test-agent",
		description: "test agent",
		tools:       tools,
		config:      config,
	}
}

func TestValidateToolsCalled(t *testing.T) {
	tests := []struct {
		name      string
		tools     map[string]tool.Tool
		calls     []model.ToolCall
		wantError bool
		errorMsg  string
	}{
		{
			name: "all tools exist",
			tools: map[string]tool.Tool{
				"tool1": newMockTool("tool1", nil),
				"tool2": newMockTool("tool2", nil),
			},
			calls: []model.ToolCall{
				{Name: "tool1", Arguments: tool.Arguments{}},
				{Name: "tool2", Arguments: tool.Arguments{}},
			},
			wantError: false,
		},
		{
			name: "unknown tool",
			tools: map[string]tool.Tool{
				"tool1": newMockTool("tool1", nil),
			},
			calls: []model.ToolCall{
				{Name: "tool1", Arguments: tool.Arguments{}},
				{Name: "unknown_tool", Arguments: tool.Arguments{}},
			},
			wantError: true,
			errorMsg:  "unknown tool: unknown_tool",
		},
		{
			name:      "empty calls",
			tools:     map[string]tool.Tool{},
			calls:     []model.ToolCall{},
			wantError: false,
		},
		{
			name:  "no tools registered",
			tools: map[string]tool.Tool{},
			calls: []model.ToolCall{
				{Name: "tool1", Arguments: tool.Arguments{}},
			},
			wantError: true,
			errorMsg:  "unknown tool: tool1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := createTestAgent(tt.tools, DefaultAgentConfig)
			err := agent.validateToolsCalled(tt.calls)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCalculateWorkerPoolSize(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
		numCalls   int
		want       int
	}{
		{
			name:       "unlimited workers (0)",
			maxWorkers: 0,
			numCalls:   10,
			want:       10,
		},
		{
			name:       "maxWorkers greater than numCalls",
			maxWorkers: 20,
			numCalls:   10,
			want:       10,
		},
		{
			name:       "maxWorkers less than numCalls",
			maxWorkers: 3,
			numCalls:   10,
			want:       3,
		},
		{
			name:       "maxWorkers equals numCalls",
			maxWorkers: 5,
			numCalls:   5,
			want:       5,
		},
		{
			name:       "single call",
			maxWorkers: 5,
			numCalls:   1,
			want:       1,
		},
		{
			name:       "unlimited with single call",
			maxWorkers: 0,
			numCalls:   1,
			want:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultAgentConfig
			config.MaxToolWorkers = tt.maxWorkers
			agent := createTestAgent(map[string]tool.Tool{}, config)
			got := agent.calculateWorkerPoolSize(tt.numCalls)
			if got != tt.want {
				t.Errorf("calculateWorkerPoolSize(%d) = %d, want %d", tt.numCalls, got, tt.want)
			}
		})
	}
}

func TestHandleToolError(t *testing.T) {
	tests := []struct {
		name           string
		errorHandling  string
		onErrorFunc    func(tool.ResultInterface) (tool.ResultInterface, error)
		toolName       string
		result         tool.ResultInterface
		wantFirstError bool
		wantContentErr bool
	}{
		{
			name:           "StopOnFirstToolError",
			errorHandling:  StopOnFirstToolError,
			toolName:       "test_tool",
			result:         tool.NewError(errors.New("test error")),
			wantFirstError: true,
			wantContentErr: false,
		},
		{
			name:           "PassErrorsToModel",
			errorHandling:  PassErrorsToModel,
			toolName:       "test_tool",
			result:         tool.NewError(errors.New("test error")),
			wantFirstError: false,
			wantContentErr: false,
		},
		{
			name:          "FunctionOnError with handler returning new result",
			errorHandling: FunctionOnError,
			onErrorFunc: func(res tool.ResultInterface) (tool.ResultInterface, error) {
				return tool.NewOK("recovered result"), nil
			},
			toolName:       "test_tool",
			result:         tool.NewError(errors.New("test error")),
			wantFirstError: false,
			wantContentErr: false,
		},
		{
			name:          "FunctionOnError with handler returning error",
			errorHandling: FunctionOnError,
			onErrorFunc: func(res tool.ResultInterface) (tool.ResultInterface, error) {
				return res, errors.New("handler error")
			},
			toolName:       "test_tool",
			result:         tool.NewError(errors.New("test error")),
			wantFirstError: false,
			wantContentErr: true,
		},
		{
			name:           "FunctionOnError without handler",
			errorHandling:  FunctionOnError,
			onErrorFunc:    nil,
			toolName:       "test_tool",
			result:         tool.NewError(errors.New("test error")),
			wantFirstError: false,
			wantContentErr: false,
		},
		{
			name:           "non-error result",
			errorHandling:  StopOnFirstToolError,
			toolName:       "test_tool",
			result:         tool.NewOK("success"),
			wantFirstError: false,
			wantContentErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultAgentConfig
			config.ToolErrorHandling = tt.errorHandling
			config.OnToolErrorFunction = tt.onErrorFunc
			agent := createTestAgent(map[string]tool.Tool{}, config)

			var firstError error
			var firstErrorMutex sync.Mutex

			finalResult, contentErr := agent.handleToolError(
				tt.toolName,
				tt.result,
				&firstError,
				&firstErrorMutex,
			)

			if tt.wantFirstError {
				if firstError == nil {
					t.Error("Expected firstError to be set, got nil")
				}
			} else {
				if firstError != nil {
					t.Errorf("Unexpected firstError: %v", firstError)
				}
			}

			if tt.wantContentErr {
				if contentErr == nil {
					t.Error("Expected contentErr to be set, got nil")
				}
			} else {
				if contentErr != nil {
					t.Errorf("Unexpected contentErr: %v", contentErr)
				}
			}

			if finalResult == nil {
				t.Error("finalResult should not be nil")
			}
		})
	}
}

func TestProcessToolResult(t *testing.T) {
	tests := []struct {
		name            string
		toolName        string
		finalResult     tool.ResultInterface
		contentErr      error
		originalError   error
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:          "successful result",
			toolName:      "test_tool",
			finalResult:   tool.NewOK("success result"),
			contentErr:    nil,
			originalError: nil,
			wantContains:  []string{`"success result"`},
		},
		{
			name:          "error result with String() error",
			toolName:      "test_tool",
			finalResult:   tool.NewError(errors.New("tool failed")),
			contentErr:    nil,
			originalError: errors.New("tool failed"),
			wantContains:  []string{"tool test_tool error:", "tool failed"},
		},
		{
			name:          "handler error",
			toolName:      "test_tool",
			finalResult:   tool.NewOK("result"),
			contentErr:    errors.New("handler error"),
			originalError: nil,
			wantContains:  []string{"tool test_tool error:", "handler error"},
		},
		{
			name:          "both handler error and original error",
			toolName:      "test_tool",
			finalResult:   tool.NewError(errors.New("original error")),
			contentErr:    errors.New("handler error"),
			originalError: errors.New("original error"),
			wantContains:  []string{"tool test_tool error:", "original error", "handler error"},
		},
		{
			name:          "complex result type",
			toolName:      "test_tool",
			finalResult:   tool.NewOK(map[string]interface{}{"key": "value", "num": 42}),
			contentErr:    nil,
			originalError: nil,
			wantContains:  []string{`"key"`, `"value"`, `"num"`, `42`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := processToolResult(tt.toolName, tt.finalResult, tt.contentErr, tt.originalError)

			for _, want := range tt.wantContains {
				if !contains(content, want) {
					t.Errorf("Expected content to contain %q, got %q", want, content)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if contains(content, notWant) {
					t.Errorf("Expected content to not contain %q, got %q", notWant, content)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAppendToolMessagesToSession(t *testing.T) {
	session := NewSession()
	results := []toolResult{
		{
			index:   0,
			call:    model.ToolCall{Name: "tool1"},
			content: "result 1",
		},
		{
			index:   1,
			call:    model.ToolCall{Name: "tool2"},
			content: "result 2",
		},
		{
			index:   2,
			call:    model.ToolCall{Name: "tool3"},
			content: "result 3",
		},
	}

	// The messages should be appended to the last step's input.
	// Create a step and then append tool messages.
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "test"},
	})
	session.AddStep(step)

	appendToolMessagesToSession(session, results)

	// Verify messages were appended to the step
	lastStep := session.Steps()[len(session.Steps())-1]
	messages := lastStep.GetAppended()

	// Should have 3 appended tool messages
	if len(messages) != 3 {
		t.Errorf("Expected 3 appended tool messages, got %d", len(messages))
	}

	// Check tool messages
	for i, result := range results {
		toolMsg := messages[i]
		if toolMsg.Role != model.RoleSystem {
			t.Errorf("Message %d: expected RoleSystem, got %v", i, toolMsg.Role)
		}
		expected := fmt.Sprintf("Tool %s returned: %s", result.call.Name, result.content)
		if toolMsg.Content != expected {
			t.Errorf("Message %d: expected content %q, got %q", i, expected, toolMsg.Content)
		}
	}
}

func TestHandleToolCalls_EmptyCalls(t *testing.T) {
	agent := createTestAgent(map[string]tool.Tool{}, DefaultAgentConfig)
	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Expected no error for empty calls, got %v", err)
	}
}

func TestHandleToolCalls_UnknownTool(t *testing.T) {
	agent := createTestAgent(map[string]tool.Tool{
		"tool1": newMockTool("tool1", nil),
	}, DefaultAgentConfig)
	session := NewSession()
	step := NewStep([]model.Message{})
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "unknown_tool", Arguments: tool.Arguments{}},
		},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err == nil {
		t.Error("Expected error for unknown tool, got nil")
	}
	if err.Error() != "unknown tool: unknown_tool" {
		t.Errorf("Expected error 'unknown tool: unknown_tool', got %q", err.Error())
	}
}

func TestHandleToolCalls_SingleTool_Success(t *testing.T) {
	executed := false
	tool1 := newMockTool("tool1", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		executed = true
		return tool.NewOK("tool1 result")
	})

	agent := createTestAgent(map[string]tool.Tool{
		"tool1": tool1,
	}, DefaultAgentConfig)
	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "tool1", Arguments: tool.Arguments{"arg": "value"}},
		},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !executed {
		t.Error("Tool was not executed")
	}

	// Verify message was added to session
	steps := session.Steps()
	if len(steps) == 0 {
		t.Fatal("Expected step to be added")
	}
	lastStep := steps[len(steps)-1]
	messages := lastStep.GetAppended()

	// Find tool message
	found := false
	for _, msg := range messages {
		if msg.Role == model.RoleSystem && contains(msg.Content, "tool1 result") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected tool message with 'tool1 result' to be added")
	}
}

func TestHandleToolCalls_MultipleTools_ParallelExecution(t *testing.T) {
	executionOrder := make([]string, 0)
	var mu sync.Mutex

	tool1 := newMockTool("tool1", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		time.Sleep(50 * time.Millisecond) // Simulate work
		mu.Lock()
		executionOrder = append(executionOrder, "tool1")
		mu.Unlock()
		return tool.NewOK("tool1 result")
	})

	tool2 := newMockTool("tool2", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		time.Sleep(30 * time.Millisecond) // Simulate work
		mu.Lock()
		executionOrder = append(executionOrder, "tool2")
		mu.Unlock()
		return tool.NewOK("tool2 result")
	})

	tool3 := newMockTool("tool3", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		time.Sleep(10 * time.Millisecond) // Simulate work
		mu.Lock()
		executionOrder = append(executionOrder, "tool3")
		mu.Unlock()
		return tool.NewOK("tool3 result")
	})

	agent := createTestAgent(map[string]tool.Tool{
		"tool1": tool1,
		"tool2": tool2,
		"tool3": tool3,
	}, DefaultAgentConfig)

	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "tool1", Arguments: tool.Arguments{}},
			{Name: "tool2", Arguments: tool.Arguments{}},
			{Name: "tool3", Arguments: tool.Arguments{}},
		},
	})

	start := time.Now()
	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// With parallel execution, total time should be less than sequential (50+30+10 = 90ms)
	// Should be close to max(50, 30, 10) = 50ms with some overhead
	if duration > 80*time.Millisecond {
		t.Errorf("Execution took too long (%v), expected parallel execution to be faster", duration)
	}

	// Verify all tools executed
	mu.Lock()
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 tools to execute, got %d", len(executionOrder))
	}
	mu.Unlock()
}

func TestHandleToolCalls_WorkerPoolLimit(t *testing.T) {
	activeWorkers := 0
	maxActive := 0
	var mu sync.Mutex

	// Create tools that track concurrent execution
	tools := make(map[string]tool.Tool)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("tool%d", i)
		tools[name] = newMockTool(name, func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
			mu.Lock()
			activeWorkers++
			if activeWorkers > maxActive {
				maxActive = activeWorkers
			}
			mu.Unlock()

			time.Sleep(20 * time.Millisecond) // Simulate work

			mu.Lock()
			activeWorkers--
			mu.Unlock()

			return tool.NewOK(fmt.Sprintf("%s result", name))
		})
	}

	config := DefaultAgentConfig
	config.MaxToolWorkers = 3 // Limit to 3 concurrent workers
	agent := createTestAgent(tools, config)

	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	calls := make([]model.ToolCall, 10)
	for i := 0; i < 10; i++ {
		calls[i] = model.ToolCall{
			Name:      fmt.Sprintf("tool%d", i),
			Arguments: tool.Arguments{},
		}
	}
	step.SetResponse(Response{ToolCalls: calls})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify max concurrent workers didn't exceed limit
	mu.Lock()
	if maxActive > 3 {
		t.Errorf("Max concurrent workers (%d) exceeded limit (3)", maxActive)
	}
	mu.Unlock()
}

func TestHandleToolCalls_OrderPreservation(t *testing.T) {
	tools := make(map[string]tool.Tool)
	results := []string{"result0", "result1", "result2", "result3", "result4"}

	for i := 0; i < 5; i++ {
		idx := i
		name := fmt.Sprintf("tool%d", i)
		// Make later tools execute faster to test ordering
		sleepTime := time.Duration(50-i*10) * time.Millisecond
		tools[name] = newMockTool(name, func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
			time.Sleep(sleepTime)
			return tool.NewOK(results[idx])
		})
	}

	agent := createTestAgent(tools, DefaultAgentConfig)
	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	calls := make([]model.ToolCall, 5)
	for i := 0; i < 5; i++ {
		calls[i] = model.ToolCall{
			Name:      fmt.Sprintf("tool%d", i),
			Arguments: tool.Arguments{},
		}
	}
	step.SetResponse(Response{ToolCalls: calls})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify messages are in correct order
	steps := session.Steps()
	if len(steps) == 0 {
		t.Fatal("Expected step to be added")
	}
	lastStep := steps[len(steps)-1]
	messages := lastStep.GetAppended()

	// Find tool messages and verify order
	toolMessages := make([]string, 0)
	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			toolMessages = append(toolMessages, msg.Content)
		}
	}

	if len(toolMessages) != 5 {
		t.Fatalf("Expected 5 tool messages, got %d", len(toolMessages))
	}

	for i, expected := range results {
		if !contains(toolMessages[i], expected) {
			t.Errorf("Message %d: expected to contain %q, got %q", i, expected, toolMessages[i])
		}
	}
}

func TestHandleToolCalls_StopOnFirstToolError(t *testing.T) {
	executed := make(map[string]bool)
	var mu sync.Mutex

	tool1 := newMockTool("tool1", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		mu.Lock()
		executed["tool1"] = true
		mu.Unlock()
		return tool.NewError(errors.New("tool1 failed"))
	})

	tool2 := newMockTool("tool2", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		mu.Lock()
		executed["tool2"] = true
		mu.Unlock()
		return tool.NewOK("tool2 result")
	})

	config := DefaultAgentConfig
	config.ToolErrorHandling = StopOnFirstToolError
	agent := createTestAgent(map[string]tool.Tool{
		"tool1": tool1,
		"tool2": tool2,
	}, config)

	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	session.AddStep(step)
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "tool1", Arguments: tool.Arguments{}},
			{Name: "tool2", Arguments: tool.Arguments{}},
		},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err == nil {
		t.Error("Expected error for StopOnFirstToolError, got nil")
	}
	if !contains(err.Error(), "tool1 failed") {
		t.Errorf("Expected error to mention 'tool1 failed', got %q", err.Error())
	}

	// Both tools may execute (parallel), but we should return error
	mu.Lock()
	// At least tool1 should have executed
	if !executed["tool1"] {
		t.Error("Expected tool1 to execute")
	}
	mu.Unlock()
}

func TestHandleToolCalls_PassErrorsToModel(t *testing.T) {
	tool1 := newMockTool("tool1", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		return tool.NewError(errors.New("tool1 failed"))
	})

	tool2 := newMockTool("tool2", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		return tool.NewOK("tool2 success")
	})

	config := DefaultAgentConfig
	config.ToolErrorHandling = PassErrorsToModel
	agent := createTestAgent(map[string]tool.Tool{
		"tool1": tool1,
		"tool2": tool2,
	}, config)

	session := NewSession()
	step := NewStep([]model.Message{})
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "tool1", Arguments: tool.Arguments{}},
			{Name: "tool2", Arguments: tool.Arguments{}},
		},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Expected no error for PassErrorsToModel, got %v", err)
	}
}

func TestHandleToolCalls_FunctionOnError(t *testing.T) {
	tool1 := newMockTool("tool1", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		return tool.NewError(errors.New("original error"))
	})

	config := DefaultAgentConfig
	config.ToolErrorHandling = FunctionOnError
	config.OnToolErrorFunction = func(res tool.ResultInterface) (tool.ResultInterface, error) {
		// Return a recovered result
		return tool.NewOK("recovered from error"), nil
	}
	agent := createTestAgent(map[string]tool.Tool{
		"tool1": tool1,
	}, config)

	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "tool1", Arguments: tool.Arguments{}},
		},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify recovered message was added
	steps := session.Steps()
	if len(steps) == 0 {
		t.Fatal("Expected step to be added")
	}
	lastStep := steps[len(steps)-1]
	messages := lastStep.GetAppended()

	found := false
	for _, msg := range messages {
		if msg.Role == model.RoleSystem && contains(msg.Content, "recovered from error") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected tool message with 'recovered from error' to be added")
	}
}

func TestHandleToolCalls_FunctionOnError_HandlerReturnsError(t *testing.T) {
	tool1 := newMockTool("tool1", func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
		return tool.NewError(errors.New("original error"))
	})

	config := DefaultAgentConfig
	config.ToolErrorHandling = FunctionOnError
	config.OnToolErrorFunction = func(res tool.ResultInterface) (tool.ResultInterface, error) {
		// Handler returns an error to pass to model
		return res, errors.New("handler error message")
	}
	agent := createTestAgent(map[string]tool.Tool{
		"tool1": tool1,
	}, config)

	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	step.SetResponse(Response{
		ToolCalls: []model.ToolCall{
			{Name: "tool1", Arguments: tool.Arguments{}},
		},
	})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify handler error message was added
	steps := session.Steps()
	if len(steps) == 0 {
		t.Fatal("Expected step to be added")
	}
	lastStep := steps[len(steps)-1]
	messages := lastStep.GetAppended()

	found := false
	for _, msg := range messages {
		if msg.Role == model.RoleSystem && contains(msg.Content, "handler error message") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected tool message with 'handler error message' to be added")
	}
}

func TestHandleToolCalls_ConcurrentErrorHandling(t *testing.T) {
	// Test that multiple errors are handled correctly in parallel
	errorCount := 0
	var mu sync.Mutex

	tools := make(map[string]tool.Tool)
	for i := 0; i < 5; i++ {
		idx := i
		name := fmt.Sprintf("tool%d", i)
		tools[name] = newMockTool(name, func(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
			mu.Lock()
			errorCount++
			mu.Unlock()

			if idx%2 == 0 {
				return tool.NewError(fmt.Errorf("error from %s", name))
			}
			return tool.NewOK(fmt.Sprintf("success from %s", name))
		})
	}

	config := DefaultAgentConfig
	config.ToolErrorHandling = PassErrorsToModel
	agent := createTestAgent(tools, config)

	session := NewSession()
	step := NewStep([]model.Message{})
	session.AddStep(step)
	calls := make([]model.ToolCall, 5)
	for i := 0; i < 5; i++ {
		calls[i] = model.ToolCall{
			Name:      fmt.Sprintf("tool%d", i),
			Arguments: tool.Arguments{},
		}
	}
	step.SetResponse(Response{ToolCalls: calls})

	ctx := tool.PrepareContext(nil, agent, tool.Arguments{})
	err := agent.handleToolCalls(ctx, session, step)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all messages were added (both errors and successes)
	steps := session.Steps()
	if len(steps) == 0 {
		t.Fatal("Expected step to be added")
	}
	lastStep := steps[len(steps)-1]
	messages := lastStep.GetAppended()

	toolMessages := make([]string, 0)
	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			toolMessages = append(toolMessages, msg.Content)
		}
	}

	if len(toolMessages) != 5 {
		t.Fatalf("Expected 5 tool messages, got %d", len(toolMessages))
	}
}
