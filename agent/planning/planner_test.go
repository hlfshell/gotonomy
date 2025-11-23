package planning

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hlfshell/gogentic/pkg/agent"
	"github.com/hlfshell/gogentic/pkg/agent/plan"
	"github.com/hlfshell/gogentic/pkg/model"
	"github.com/hlfshell/gogentic/pkg/prompt"
)

// MockModel is a mock implementation of the Model interface for testing.
type MockModel struct {
	Response      string
	ResponseError error
	CompleteFunc  func(ctx context.Context, req model.CompletionRequest) (model.CompletionResponse, error)
}

func (m *MockModel) Complete(ctx context.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, req)
	}
	if m.ResponseError != nil {
		return model.CompletionResponse{}, m.ResponseError
	}
	return model.CompletionResponse{
		Text: m.Response,
		UsageStats: model.UsageStats{
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
		},
	}, nil
}

func (m *MockModel) CompleteStream(ctx context.Context, req model.CompletionRequest, handler model.StreamHandler) error {
	return nil
}

func (m *MockModel) GetInfo() model.ModelInfo {
	return model.ModelInfo{
		Name:         "mock-model",
		Provider:     "mock",
		Capabilities: []model.Capability{model.TextGeneration},
	}
}

func (m *MockModel) SupportsContentType(contentType model.ContentType) bool {
	return contentType == model.TextContent
}

func TestNewPlannerAgent(t *testing.T) {
	config := agent.AgentConfig{
		Model:       &MockModel{},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("", "", "", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	if planner == nil {
		t.Fatal("Planner agent is nil")
	}

	if planner.ID() == "" {
		t.Error("Planner agent should have an ID")
	}

	if planner.Name() == "" {
		t.Error("Planner agent should have a name")
	}

	// Verify that the prompt template was loaded automatically
	if planner.promptTemplate == nil {
		t.Error("Prompt template should be loaded automatically")
	}
}

func TestPlannerAgent_SetPromptTemplate(t *testing.T) {
	config := agent.AgentConfig{
		Model:       &MockModel{},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Verify default template is loaded
	if planner.promptTemplate == nil {
		t.Fatal("Default prompt template should be loaded")
	}

	originalTemplate := planner.promptTemplate

	// Create a custom template
	customTmpl, err := prompt.AddTemplate("test-planner-custom", "Objective: {{.objective}}")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// Set custom template
	planner.SetPromptTemplate(customTmpl)

	if planner.promptTemplate == nil {
		t.Error("Prompt template should be set")
	}

	if planner.promptTemplate == originalTemplate {
		t.Error("Prompt template should be different after SetPromptTemplate")
	}
}

func TestPlannerAgent_Plan_Success(t *testing.T) {
	// Create a mock response that represents a valid plan
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Research",
				"instruction": "Research the topic thoroughly",
				"expectation": "A comprehensive research report",
				"dependencies": []
			},
			{
				"id": "step2",
				"name": "Analysis",
				"instruction": "Analyze the research findings",
				"expectation": "A detailed analysis document",
				"dependencies": ["step1"]
			},
			{
				"id": "step3",
				"name": "Implementation",
				"instruction": "Implement the solution",
				"expectation": "A working implementation",
				"dependencies": ["step2"]
			}
		]
	}`

	mockModel := &MockModel{
		Response: mockPlanJSON,
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically, no need to set it

	// Create a plan
	input := PlannerInput{
		Objective: "Build a web application",
	}

	result, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.Plan == nil {
		t.Fatal("Plan is nil")
	}

	if len(result.Plan.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(result.Plan.Steps))
	}

	// Verify the steps
	if result.Plan.Steps[0].ID != "step1" {
		t.Errorf("Expected step1, got %s", result.Plan.Steps[0].ID)
	}

	// Verify dependencies
	if len(result.Plan.Steps[1].Dependencies) != 1 {
		t.Errorf("Expected 1 dependency for step2, got %d", len(result.Plan.Steps[1].Dependencies))
	}

	if result.Plan.Steps[1].Dependencies[0].ID != "step1" {
		t.Errorf("Expected step2 to depend on step1")
	}

	// Verify usage stats
	if result.UsageStats.TotalTokens == 0 {
		t.Error("Expected non-zero usage stats")
	}
}

func TestPlannerAgent_Plan_WithNestedSubPlan(t *testing.T) {
	// Create a mock response with nested sub-plans
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Setup Phase",
				"instruction": "Setup the project",
				"expectation": "Project is ready to start",
				"dependencies": [],
				"sub_plan": {
					"steps": [
						{
							"id": "sub1",
							"name": "Install Dependencies",
							"instruction": "Install all required dependencies",
							"expectation": "All dependencies installed",
							"dependencies": []
						},
						{
							"id": "sub2",
							"name": "Configure Environment",
							"instruction": "Configure the development environment",
							"expectation": "Environment configured",
							"dependencies": ["sub1"]
						}
					]
				}
			},
			{
				"id": "step2",
				"name": "Development",
				"instruction": "Develop the application",
				"expectation": "Application is developed",
				"dependencies": ["step1"]
			}
		]
	}`

	mockModel := &MockModel{
		Response: mockPlanJSON,
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	input := PlannerInput{
		Objective: "Build a web application with detailed setup",
	}

	result, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	if len(result.Plan.Steps) != 2 {
		t.Errorf("Expected 2 top-level steps, got %d", len(result.Plan.Steps))
	}

	// Check that the first step has a sub-plan
	if result.Plan.Steps[0].Plan == nil {
		t.Fatal("Expected step1 to have a sub-plan")
	}

	if len(result.Plan.Steps[0].Plan.Steps) != 2 {
		t.Errorf("Expected 2 sub-steps, got %d", len(result.Plan.Steps[0].Plan.Steps))
	}

	// Verify sub-plan dependencies
	if len(result.Plan.Steps[0].Plan.Steps[1].Dependencies) != 1 {
		t.Errorf("Expected sub2 to have 1 dependency")
	}
}

func TestPlannerAgent_Plan_WithTools(t *testing.T) {
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Search",
				"instruction": "Use the web_search tool to find information",
				"expectation": "Relevant information found",
				"dependencies": []
			}
		]
	}`

	mockModel := &MockModel{
		Response: mockPlanJSON,
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	input := PlannerInput{
		Objective: "Research a topic",
		Tools: []ToolInfo{
			{
				Name:        "web_search",
				Description: "Search the web for information",
				UsageNotes:  "Use for finding current information",
			},
		},
	}

	result, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	if result.Plan == nil {
		t.Fatal("Plan is nil")
	}

	// Verify the instruction mentions the tool
	if !containsSubstring(result.Plan.Steps[0].Instruction, "web_search") {
		t.Error("Expected instruction to mention web_search tool")
	}
}

func TestPlannerAgent_Plan_NoTemplate(t *testing.T) {
	mockModel := &MockModel{
		Response: "{}",
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Clear the template to test error handling
	planner.SetPromptTemplate(nil)

	input := PlannerInput{
		Objective: "Build something",
	}

	_, err = planner.Plan(context.Background(), input)
	if err == nil {
		t.Error("Expected error when template not loaded")
	}
}

func TestPlannerAgent_Replan(t *testing.T) {
	// Create an initial plan
	initialPlan := plan.NewPlan("initial-plan")
	step1 := plan.NewStep("step1", "Initial Step", "Do something", "Result", nil, nil)
	initialPlan.AddStep(step1)

	// Mock response for the revised plan
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Revised Step",
				"instruction": "Do something better",
				"expectation": "Better result",
				"dependencies": []
			},
			{
				"id": "step2",
				"name": "New Step",
				"instruction": "Additional work",
				"expectation": "Additional result",
				"dependencies": ["step1"]
			}
		]
	}`

	mockModel := &MockModel{
		Response: mockPlanJSON,
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	input := PlannerInput{
		Objective: "Build a better version",
	}

	feedback := "The initial plan is too simple, we need more steps"

	result, err := planner.Replan(context.Background(), initialPlan, feedback, input)
	if err != nil {
		t.Fatalf("Failed to replan: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.Plan == nil {
		t.Fatal("Plan is nil")
	}

	// Check that the plan has a revision diff
	if result.Plan.RevisionDiff == nil {
		t.Fatal("Expected revision diff to be present")
	}

	if result.Plan.RevisionDiff.FromPlanID != initialPlan.ID {
		t.Errorf("Expected FromPlanID to be %s, got %s", initialPlan.ID, result.Plan.RevisionDiff.FromPlanID)
	}

	// Check that the diff shows additions
	if len(result.Plan.RevisionDiff.Steps.Added) == 0 {
		t.Error("Expected some added steps in the diff")
	}
}

func TestPlannerAgent_Execute(t *testing.T) {
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Step 1",
				"instruction": "Do something",
				"expectation": "Something done",
				"dependencies": []
			}
		]
	}`

	mockModel := &MockModel{
		Response: mockPlanJSON,
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	params := agent.Arguments{
		"input": "Create a simple plan",
	}

	result, err := planner.ExecuteAgent(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}

	// Check that the plan is in additional outputs
	if result.AdditionalOutputs == nil {
		t.Fatal("Expected additional outputs")
	}

	planOutput, ok := result.AdditionalOutputs["plan"].(*plan.Plan)
	if !ok {
		t.Fatal("Expected plan in additional outputs")
	}

	if len(planOutput.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(planOutput.Steps))
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `{"steps": []}`,
			expected: `{"steps": []}`,
		},
		{
			name:     "JSON with markdown",
			input:    "```json\n{\"steps\": []}\n```",
			expected: `{"steps": []}`,
		},
		{
			name:     "JSON with plain markdown",
			input:    "```\n{\"steps\": []}\n```",
			expected: `{"steps": []}`,
		},
		{
			name:     "JSON with whitespace",
			input:    "  \n\n{\"steps\": []}  \n\n",
			expected: `{"steps": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanJSONResponse(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParsePlanFromResponse_InvalidJSON(t *testing.T) {
	config := agent.AgentConfig{
		Model:       &MockModel{},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	invalidJSON := "This is not valid JSON"
	_, err = planner.parsePlanFromResponse(invalidJSON)
	if err == nil {
		t.Error("Expected error when parsing invalid JSON")
	}
}

func TestParsePlanFromResponse_InvalidPlan(t *testing.T) {
	config := agent.AgentConfig{
		Model:       &MockModel{},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Plan with circular dependency
	invalidPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Step 1",
				"instruction": "Do something",
				"expectation": "Something done",
				"dependencies": ["step2"]
			},
			{
				"id": "step2",
				"name": "Step 2",
				"instruction": "Do something else",
				"expectation": "Something else done",
				"dependencies": ["step1"]
			}
		]
	}`

	_, err = planner.parsePlanFromResponse(invalidPlanJSON)
	if err == nil {
		t.Error("Expected error when parsing plan with circular dependencies")
	}
}

func TestPlannerAgent_Plan_VerifiesValidPlan(t *testing.T) {
	// This test ensures the planner validates the generated plan
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Step 1",
				"instruction": "First step",
				"expectation": "First done",
				"dependencies": []
			}
		]
	}`

	callCount := 0
	mockModel := &MockModel{
		CompleteFunc: func(ctx context.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
			callCount++
			return model.CompletionResponse{
				Text: mockPlanJSON,
				UsageStats: model.UsageStats{
					TotalTokens: 100,
				},
			}, nil
		},
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	input := PlannerInput{
		Objective: "Test objective",
	}

	result, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Verify the model was called
	if callCount != 1 {
		t.Errorf("Expected model to be called once, got %d", callCount)
	}

	// Verify the plan is valid
	if err := result.Plan.Validate(); err != nil {
		t.Errorf("Plan should be valid: %v", err)
	}
}

func TestBuildPlanFromResponse_MissingDependency(t *testing.T) {
	config := agent.AgentConfig{
		Model:       &MockModel{},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Plan with a dependency that doesn't exist
	resp := planResponse{
		Steps: []stepResponse{
			{
				ID:           "step1",
				Name:         "Step 1",
				Instruction:  "Do something",
				Expectation:  "Something done",
				Dependencies: []string{"nonexistent"},
			},
		},
	}

	_, err = planner.buildPlanFromResponse(resp)
	if err == nil {
		t.Error("Expected error when building plan with missing dependency")
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			contains(s, substr)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPlanSerialization_RoundTrip(t *testing.T) {
	// Test that a plan can be serialized and deserialized correctly
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Step 1",
				"instruction": "Do something",
				"expectation": "Something done",
				"dependencies": []
			},
			{
				"id": "step2",
				"name": "Step 2",
				"instruction": "Do something else",
				"expectation": "Something else done",
				"dependencies": ["step1"]
			}
		]
	}`

	config := agent.AgentConfig{
		Model:       &MockModel{Response: mockPlanJSON},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	input := PlannerInput{
		Objective: "Test objective",
	}

	result, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Serialize the plan
	data, err := json.Marshal(result.Plan)
	if err != nil {
		t.Fatalf("Failed to marshal plan: %v", err)
	}

	// Deserialize the plan
	var deserializedPlan plan.Plan
	if err := json.Unmarshal(data, &deserializedPlan); err != nil {
		t.Fatalf("Failed to unmarshal plan: %v", err)
	}

	// Verify the deserialized plan
	if len(deserializedPlan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(deserializedPlan.Steps))
	}

	if len(deserializedPlan.Steps[1].Dependencies) != 1 {
		t.Errorf("Expected 1 dependency for step2, got %d", len(deserializedPlan.Steps[1].Dependencies))
	}

	if deserializedPlan.Steps[1].Dependencies[0].ID != "step1" {
		t.Errorf("Expected step2 to depend on step1")
	}
}

func TestPlannerAgent_Execute_WithAdditionalInputs(t *testing.T) {
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Step 1",
				"instruction": "Do something",
				"expectation": "Something done",
				"dependencies": []
			}
		]
	}`

	mockModel := &MockModel{
		Response: mockPlanJSON,
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	params := agent.Arguments{
		"input": "Create a plan",
		"context": "This is additional context",
		"tools": []ToolInfo{
			{
				Name:        "tool1",
				Description: "A test tool",
			},
		},
	}

	result, err := planner.ExecuteAgent(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}

	// Verify execution stats
	if result.ExecutionStats.StartTime.IsZero() {
		t.Error("Expected non-zero start time")
	}

	if result.ExecutionStats.EndTime.IsZero() {
		t.Error("Expected non-zero end time")
	}

	if result.ExecutionStats.Iterations != 1 {
		t.Errorf("Expected 1 iteration, got %d", result.ExecutionStats.Iterations)
	}
}

func TestPlannerAgent_TimestampsAndStats(t *testing.T) {
	mockPlanJSON := `{
		"steps": [
			{
				"id": "step1",
				"name": "Step 1",
				"instruction": "Do something",
				"expectation": "Something done",
				"dependencies": []
			}
		]
	}`

	mockModel := &MockModel{
		CompleteFunc: func(ctx context.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			return model.CompletionResponse{
				Text: mockPlanJSON,
				UsageStats: model.UsageStats{
					PromptTokens:     150,
					CompletionTokens: 250,
					TotalTokens:      400,
				},
			}, nil
		},
	}

	config := agent.AgentConfig{
		Model:       mockModel,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	planner, err := NewPlannerAgent("test-planner", "Test Planner", "A test planner", config)
	if err != nil {
		t.Fatalf("Failed to create planner agent: %v", err)
	}

	// Planner now has embedded template loaded automatically

	params := agent.Arguments{
		"input": "Create a plan",
	}

	result, err := planner.ExecuteAgent(context.Background(), params, nil)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	// Verify usage stats
	if result.UsageStats.PromptTokens != 150 {
		t.Errorf("Expected 150 prompt tokens, got %d", result.UsageStats.PromptTokens)
	}

	if result.UsageStats.CompletionTokens != 250 {
		t.Errorf("Expected 250 completion tokens, got %d", result.UsageStats.CompletionTokens)
	}

	if result.UsageStats.TotalTokens != 400 {
		t.Errorf("Expected 400 total tokens, got %d", result.UsageStats.TotalTokens)
	}

	// Verify timestamps
	if result.ExecutionStats.EndTime.Before(result.ExecutionStats.StartTime) {
		t.Error("End time should be after start time")
	}

	// Verify the plan was created at the right time
	planOutput := result.AdditionalOutputs["plan"].(*plan.Plan)
	if planOutput.CreatedAt.IsZero() {
		t.Error("Plan creation time should not be zero")
	}
}
