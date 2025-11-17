package plan

import (
	"strings"
	"testing"
)

func TestNewStep(t *testing.T) {
	step := NewStep("", "Test Step", "Do something", "Expected result", nil, nil)
	if step.ID == "" {
		t.Error("Step ID should not be empty")
	}
	if step.Name != "Test Step" {
		t.Errorf("Expected name 'Test Step', got %s", step.Name)
	}

	customID := "step-123"
	step2 := NewStep(customID, "Step 2", "Instruction", "Expectation", nil, nil)
	if step2.ID != customID {
		t.Errorf("Expected ID %s, got %s", customID, step2.ID)
	}
}

func TestHasDependencies(t *testing.T) {
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	if len(step1.Dependencies) > 0 {
		t.Error("Step without dependencies should return false")
	}

	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", []*Step{&step1}, nil)
	if len(step2.Dependencies) == 0 {
		t.Error("Step with dependencies should return true")
	}
}

func TestAllDependenciesSatisfied(t *testing.T) {
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", []*Step{&step1}, nil)

	completed := map[string]bool{}
	if step2.AllDependenciesSatisfied(completed) {
		t.Error("Dependencies should not be satisfied when step1 is not completed")
	}

	completed["step1"] = true
	if !step2.AllDependenciesSatisfied(completed) {
		t.Error("Dependencies should be satisfied when step1 is completed")
	}
}

func TestHasSubPlan(t *testing.T) {
	step := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	if step.Plan != nil {
		t.Error("Step without sub-plan should return false")
	}

	subPlan := NewPlan("sub")
	stepWithSub := NewStep("step2", "Step 2", "Instruction", "Expectation", nil, subPlan)
	if stepWithSub.Plan == nil {
		t.Error("Step with sub-plan should return true")
	}
}

func TestStepToText(t *testing.T) {
	step := NewStep("step1", "Test Step", "Instruction", "Expectation", nil, nil)
	text := step.ToText()
	if text == "" {
		t.Fatal("ToText should not return empty string")
	}
	// Basic substring check using strings package
	if !strings.Contains(text, "Test Step") {
		t.Error("ToText should contain step name")
	}
	if !strings.Contains(text, "Instruction") {
		t.Error("ToText should contain instruction")
	}
}
