package plan

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewPlan(t *testing.T) {
	plan := NewPlan("")
	if plan == nil {
		t.Fatal("NewPlan returned nil")
	}
	if plan.ID == "" {
		t.Error("Plan ID should not be empty")
	}
	if len(plan.Steps) != 0 {
		t.Error("New plan should have no steps")
	}
	if plan.RevisionDiff != nil {
		t.Error("New plan should not have a revision diff")
	}

	customID := "test-plan-123"
	plan2 := NewPlan(customID)
	if plan2.ID != customID {
		t.Errorf("Expected ID %s, got %s", customID, plan2.ID)
	}
}

func TestAddStep(t *testing.T) {
	plan := NewPlan("")
	step := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)

	plan.AddStep(step)
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].ID != "step1" {
		t.Errorf("Expected step ID 'step1', got %s", plan.Steps[0].ID)
	}
}

func TestFindStep(t *testing.T) {
	plan := NewPlan("")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", nil, nil)

	plan.AddStep(step1)
	plan.AddStep(step2)

	found, ok := plan.FindStep("step1")
	if !ok || found == nil {
		t.Fatal("FindStep should find step1")
	}
	if found.ID != "step1" {
		t.Errorf("Expected step ID 'step1', got %s", found.ID)
	}

	if _, ok := plan.FindStep("missing"); ok {
		t.Error("FindStep should not find nonexistent step")
	}
}

func TestGetReadySteps(t *testing.T) {
	plan := NewPlan("")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", []*Step{&step1}, nil)

	plan.AddStep(step1)
	plan.AddStep(step2)

	completed := map[string]bool{}
	ready := plan.NextSteps(completed)
	if len(ready) != 1 || ready[0].ID != "step1" {
		t.Fatalf("Expected only step1 to be ready")
	}

	completed["step1"] = true
	ready = plan.NextSteps(completed)
	if len(ready) != 2 {
		t.Errorf("Expected both steps ready, got %d", len(ready))
	}
}

func TestValidate(t *testing.T) {
	plan := NewPlan("")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	plan.AddStep(step1)

	if err := plan.Validate(); err != nil {
		t.Errorf("Valid plan should not error: %v", err)
	}

	plan2 := NewPlan("")
	plan2.AddStep(step1)
	plan2.AddStep(NewStep("step1", "Step 2", "Instruction", "Expectation", nil, nil))
	if err := plan2.Validate(); err == nil {
		t.Error("Plan with duplicate step IDs should error")
	}

	plan3 := NewPlan("")
	step3 := NewStep("step3", "Step 3", "Instruction", "Expectation", nil, nil)
	step4 := NewStep("step4", "Step 4", "Instruction", "Expectation", []*Step{&step3}, nil)
	plan3.AddStep(step4)
	if err := plan3.Validate(); err == nil {
		t.Error("Plan with invalid dependency should error")
	}
}

func TestGetExecutionOrder(t *testing.T) {
	plan := NewPlan("")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", []*Step{&step1}, nil)

	plan.AddStep(step2)
	plan.AddStep(step1)

	order, err := plan.GetExecutionOrder()
	if err != nil {
		t.Fatalf("GetExecutionOrder failed: %v", err)
	}
	if len(order) != 2 || order[0].ID != "step1" || order[1].ID != "step2" {
		t.Error("Execution order should respect dependencies")
	}
}

func TestRevisionDiff(t *testing.T) {
	// Test that we can track plan revisions using RevisionDiff
	v1 := NewPlan("plan-v1")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	v1.AddStep(step1)

	v2 := NewPlan("plan-v2")
	step1Modified := NewStep("s1", "Step 1 Modified", "Do 1 Modified", "Expect 1", nil, nil)
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	v2.AddStep(step1Modified)
	v2.AddStep(step2)

	// Create a diff tracking the change from v1 to v2
	diff := NewPlanDiff("diff-id", v1, v2, "Updated step 1 and added step 2")

	// Attach the diff to the revision
	v2.RevisionDiff = &diff

	if v2.RevisionDiff == nil {
		t.Fatal("Revision should have a diff")
	}
	if v2.RevisionDiff.FromPlanID != "plan-v1" {
		t.Errorf("Expected FromPlanID='plan-v1', got %s", v2.RevisionDiff.FromPlanID)
	}
	if v2.RevisionDiff.ToPlanID != "plan-v2" {
		t.Errorf("Expected ToPlanID='plan-v2', got %s", v2.RevisionDiff.ToPlanID)
	}
	if len(v2.RevisionDiff.Steps.Added) != 1 {
		t.Error("Revision diff should show 1 added step")
	}
	if len(v2.RevisionDiff.Steps.Changed) != 1 {
		t.Error("Revision diff should show 1 changed step")
	}

	// Test that a plan without RevisionDiff has no previous plan
	originalPlan := NewPlan("original")
	if originalPlan.RevisionDiff != nil {
		t.Error("New plan should not have a RevisionDiff")
	}
}

func TestGetAllStepsRecursive(t *testing.T) {
	plan := NewPlan("")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	subPlan := NewPlan("sub")
	subStep := NewStep("substep1", "Sub Step", "Instruction", "Expectation", nil, nil)
	subPlan.AddStep(subStep)
	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", nil, subPlan)

	plan.AddStep(step1)
	plan.AddStep(step2)

	allSteps := plan.GetAllSteps()
	if len(allSteps) != 3 {
		t.Errorf("Expected 3 steps total, got %d", len(allSteps))
	}
}

func TestGetMaxDepth(t *testing.T) {
	plan := NewPlan("")
	if plan.GetMaxDepth() != 1 {
		t.Errorf("Plan without sub-plans should have depth 1, got %d", plan.GetMaxDepth())
	}

	subPlan := NewPlan("sub")
	step := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, subPlan)
	plan.AddStep(step)
	if plan.GetMaxDepth() != 2 {
		t.Errorf("Plan with nesting should have depth 2, got %d", plan.GetMaxDepth())
	}
}

func TestPlanSerialization(t *testing.T) {
	plan := NewPlan("test-plan")
	step1 := NewStep("step1", "Step 1", "Instruction 1", "Expectation 1", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction 2", "Expectation 2", []*Step{&step1}, nil)
	plan.AddStep(step1)
	plan.AddStep(step2)

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal plan: %v", err)
	}

	var unmarshaled Plan
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal plan: %v", err)
	}

	if unmarshaled.ID != plan.ID || len(unmarshaled.Steps) != 2 {
		t.Error("Plan serialization should preserve fields")
	}
	if len(unmarshaled.Steps[1].Dependencies) != 1 || unmarshaled.Steps[1].Dependencies[0].ID != "step1" {
		t.Error("Dependencies should be restored after serialization")
	}
}

func TestPlanToText(t *testing.T) {
	plan := NewPlan("test-plan")
	step := NewStep("step1", "Test Step", "Do something", "Expected result", nil, nil)
	plan.AddStep(step)

	text := plan.ToText()
	if text == "" {
		t.Fatal("ToText should not return empty string")
	}
	if !strings.Contains(text, "test-plan") {
		t.Error("ToText should contain plan ID")
	}
	if !strings.Contains(text, "Test Step") {
		t.Error("ToText should include step details")
	}
}

func TestAddStepByID(t *testing.T) {
	plan := NewPlan("")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	plan.AddStep(step1)

	err := plan.AddStepByID("step2", "Step 2", "Instruction", "Expectation", []string{"step1"}, nil)
	if err != nil {
		t.Fatalf("AddStepByID failed: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}
	if len(plan.Steps[1].Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(plan.Steps[1].Dependencies))
	}
}

func TestNestedPlanSerialization(t *testing.T) {
	plan := NewPlan("parent")
	subPlan := NewPlan("sub")
	subStep := NewStep("substep1", "Sub Step", "Instruction", "Expectation", nil, nil)
	subPlan.AddStep(subStep)
	step := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, subPlan)
	plan.AddStep(step)

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("Failed to marshal nested plan: %v", err)
	}

	var unmarshaled Plan
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal nested plan: %v", err)
	}

	if unmarshaled.Steps[0].Plan == nil || len(unmarshaled.Steps[0].Plan.Steps) != 1 {
		t.Error("Sub-plan should be restored with its steps")
	}
}

func TestNestedPlanSerialization_DependencyPointers(t *testing.T) {
	// Ensure that dependency pointers inside a nested sub-plan are restored correctly.
	parent := NewPlan("parent")

	sub := NewPlan("sub")
	sub1 := NewStep("sub1", "Sub 1", "Do sub1", "sub1 done", nil, nil)
	sub2 := NewStep("sub2", "Sub 2", "Do sub2", "sub2 done", []*Step{&sub1}, nil)
	sub.AddStep(sub1)
	sub.AddStep(sub2)

	parentStep := NewStep("step1", "Parent Step", "Do parent", "parent done", nil, sub)
	parent.AddStep(parentStep)

	// Serialize
	data, err := json.Marshal(parent)
	if err != nil {
		t.Fatalf("Failed to marshal plan: %v", err)
	}

	// Deserialize
	var unmarshaled Plan
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal plan: %v", err)
	}

	if len(unmarshaled.Steps) != 1 {
		t.Fatalf("Expected 1 parent step, got %d", len(unmarshaled.Steps))
	}
	if unmarshaled.Steps[0].Plan == nil {
		t.Fatal("Expected parent step to have a sub-plan")
	}
	if len(unmarshaled.Steps[0].Plan.Steps) != 2 {
		t.Fatalf("Expected 2 sub-steps, got %d", len(unmarshaled.Steps[0].Plan.Steps))
	}

	usub := unmarshaled.Steps[0].Plan
	if len(usub.Steps[1].Dependencies) != 1 {
		t.Fatalf("Expected sub2 to have 1 dependency, got %d", len(usub.Steps[1].Dependencies))
	}

	depPtr := usub.Steps[1].Dependencies[0]
	if depPtr == nil {
		t.Fatal("Expected dependency pointer to be non-nil")
	}
	if depPtr.ID != "sub1" {
		t.Fatalf("Expected dependency id 'sub1', got %q", depPtr.ID)
	}
	// The pointer should refer to the actual step struct inside the same sub-plan slice.
	if depPtr != &usub.Steps[0] {
		t.Fatal("Expected dependency pointer to point at sub-plan step instance")
	}
}
