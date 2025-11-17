package plan

import (
	"encoding/json"
	"testing"
)

func TestCreateStepDiff(t *testing.T) {
	oldStep := NewStep("step1", "Old Name", "Old Instruction", "Old Expectation", nil, nil)
	newStep := NewStep("step1", "New Name", "New Instruction", "Old Expectation", nil, nil)

	diff := CreateStepDiff(&oldStep, &newStep, "Testing changes")
	if diff == nil {
		t.Fatal("CreateStepDiff returned nil")
	}
	if diff.ID == "" {
		t.Error("Diff should have a unique ID")
	}
	if diff.FromID != "step1" || diff.ToID != "step1" {
		t.Errorf("Expected FromID and ToID to be 'step1', got FromID=%s, ToID=%s", diff.FromID, diff.ToID)
	}
	if diff.Reason != "Testing changes" {
		t.Errorf("Expected reason 'Testing changes', got %s", diff.Reason)
	}
	if diff.From == nil || diff.To == nil {
		t.Error("Diff should have From and To pointers")
	}
	if diff.From.Name != "Old Name" || diff.To.Name != "New Name" {
		t.Error("Diff should preserve old/new names")
	}

	if CreateStepDiff(&oldStep, &oldStep, "") != nil {
		t.Error("Diffing a step with itself (same pointer) should return nil diff")
	}
}

func TestCreatePlanDiff(t *testing.T) {
	oldPlan := NewPlan("plan1")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	oldPlan.AddStep(step1)

	newPlan := NewPlan("plan1")
	step1Modified := NewStep("step1", "Step 1 Modified", "New Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction", "Expectation", nil, nil)
	newPlan.AddStep(step1Modified)
	newPlan.AddStep(step2)

	diff := CreatePlanDiff(oldPlan, newPlan, "Adding step and modifying existing")
	if diff == nil {
		t.Fatal("CreatePlanDiff returned nil")
	}
	if diff.ID == "" {
		t.Error("Diff should have a unique ID")
	}
	if diff.FromID != "plan1" || diff.ToID != "plan1" {
		t.Errorf("Expected FromID and ToID to be 'plan1', got FromID=%s, ToID=%s", diff.FromID, diff.ToID)
	}
	if diff.Reason != "Adding step and modifying existing" {
		t.Errorf("Expected reason 'Adding step and modifying existing', got %s", diff.Reason)
	}
	if diff.From == nil || diff.To == nil {
		t.Error("Diff should have From and To pointers")
	}
}

func TestDiffAccessors(t *testing.T) {
	// Test that diffs provide access to old and new versions
	step := NewStep("step1", "Old Name", "Old Instruction", "Old Expectation", nil, nil)
	newStepObj := NewStep("step1", "New Name", "New Instruction", "Old Expectation", nil, nil)

	diff := CreateStepDiff(&step, &newStepObj, "Testing diff accessors")
	if diff == nil {
		t.Fatal("CreateStepDiff returned nil")
	}

	// Access the new version directly through diff.To
	if diff.To.Name != "New Name" || diff.To.Instruction != "New Instruction" {
		t.Error("Diff.To should contain the new step")
	}

	// Access the old version through diff.From
	if diff.From.Name != "Old Name" || diff.From.Instruction != "Old Instruction" {
		t.Error("Diff.From should contain the old step")
	}
}

func TestDiffUsagePattern(t *testing.T) {
	// Test typical usage pattern: create diff, use it to track revisions
	plan := NewPlan("plan1")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	plan.AddStep(step1)

	newPlanObj := NewPlan("plan1")
	step1Modified := NewStep("step1", "Step 1 Modified", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "New Instruction", "New Expectation", nil, nil)
	newPlanObj.AddStep(step1Modified)
	newPlanObj.AddStep(step2)

	diff := CreatePlanDiff(plan, newPlanObj, "Added step and modified existing")
	if diff == nil {
		t.Fatal("CreatePlanDiff returned nil")
	}

	// Use the diff to track revision history
	revision := diff.To
	revision.RevisionDiff = diff

	if revision.RevisionDiff == nil {
		t.Error("Revision should have a diff")
	}
	if revision.RevisionDiff.FromID != "plan1" {
		t.Error("Revision diff should reference previous plan")
	}
	if len(revision.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(revision.Steps))
	}
}

func TestDiffSerialization(t *testing.T) {
	oldPlan := NewPlan("plan1")
	newPlan := NewPlan("plan1")
	step1 := NewStep("step1", "New Step", "Instruction", "Expectation", nil, nil)
	newPlan.AddStep(step1)

	diff := &Diff[*Plan]{
		ID:     "plan1",
		Reason: "Test revision",
		From:   oldPlan,
		To:     newPlan,
		FromID: "plan1",
		ToID:   "plan1",
	}

	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Failed to marshal diff: %v", err)
	}

	var unmarshaled Diff[*Plan]
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal diff: %v", err)
	}

	if unmarshaled.ID != diff.ID {
		t.Errorf("Expected ID %s, got %s", diff.ID, unmarshaled.ID)
	}
	if unmarshaled.FromID != diff.FromID || unmarshaled.ToID != diff.ToID {
		t.Error("Serialized diff should preserve FromID/ToID")
	}
}

func TestDiffEdgeCases(t *testing.T) {
	if CreateStepDiff(nil, nil, "") != nil {
		t.Error("CreateStepDiff with nil inputs should return nil")
	}

	step1 := NewStep("step1", "Step", "Instruction", "Expectation", nil, nil)
	if CreateStepDiff(nil, &step1, "") != nil {
		t.Error("CreateStepDiff with nil oldStep should return nil")
	}
	if CreateStepDiff(&step1, nil, "") != nil {
		t.Error("CreateStepDiff with nil newStep should return nil")
	}

	step2 := NewStep("step2", "Step", "Instruction", "Expectation", nil, nil)
	diffCrossStep := CreateStepDiff(&step1, &step2, "")
	if diffCrossStep == nil {
		t.Error("CreateStepDiff with different IDs should create a diff (cross-step diff)")
	}
	if diffCrossStep != nil && (diffCrossStep.FromID != "step1" || diffCrossStep.ToID != "step2") {
		t.Errorf("Cross-step diff should have FromID=step1 and ToID=step2, got FromID=%s, ToID=%s", diffCrossStep.FromID, diffCrossStep.ToID)
	}

	if CreatePlanDiff(nil, nil, "") != nil {
		t.Error("CreatePlanDiff with nil inputs should return nil")
	}

	plan1 := NewPlan("plan1")
	if CreatePlanDiff(nil, plan1, "") != nil {
		t.Error("CreatePlanDiff with nil oldPlan should return nil")
	}
	if CreatePlanDiff(plan1, nil, "") != nil {
		t.Error("CreatePlanDiff with nil newPlan should return nil")
	}
}

func TestDiffWithEmptyFields(t *testing.T) {
	step1 := NewStep("step1", "Step", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step1", "Step", "Instruction", "Expectation", nil, nil)
	if CreateStepDiff(&step1, &step2, "") != nil {
		t.Error("CreateStepDiff with identical steps should return nil")
	}

	plan1 := NewPlan("plan1")
	plan1.AddStep(step1)

	step3 := NewStep("step1", "New Name", "Instruction", "Expectation", nil, nil)
	diff := CreateStepDiff(&step1, &step3, "")
	if diff == nil {
		t.Fatal("CreateStepDiff should return diff when name changes")
	}
	if diff.Reason != "" {
		t.Error("Diff reason should be empty when not provided")
	}
}

func TestDiffSerializationEdgeCases(t *testing.T) {
	var nilDiff *Diff[*Plan]
	data, err := json.Marshal(nilDiff)
	if err != nil {
		t.Fatalf("Failed to marshal nil diff: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("Expected 'null' for nil diff, got %s", string(data))
	}

	emptyDiff := &Diff[*Plan]{ID: "test"}
	data, err = json.Marshal(emptyDiff)
	if err != nil {
		t.Fatalf("Failed to marshal empty diff: %v", err)
	}

	var unmarshaled Diff[*Plan]
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal empty diff: %v", err)
	}

	if unmarshaled.ID != "test" {
		t.Errorf("Expected ID 'test', got %s", unmarshaled.ID)
	}
}

func TestDiffWithComplexNesting(t *testing.T) {
	subSubPlan := NewPlan("sub-sub-plan")
	subSubStep := NewStep("sub-sub-step", "Sub Sub Step", "Instruction", "Expectation", nil, nil)
	subSubPlan.AddStep(subSubStep)

	subPlan := NewPlan("sub-plan")
	subStep := NewStep("sub-step", "Sub Step", "Instruction", "Expectation", nil, subSubPlan)
	subPlan.AddStep(subStep)

	plan := NewPlan("parent-plan")
	step := NewStep("step", "Step", "Instruction", "Expectation", nil, subPlan)
	plan.AddStep(step)

	newSubSubPlan := NewPlan("sub-sub-plan")
	newSubSubStep := NewStep("sub-sub-step", "Sub Sub Step Updated", "New Instruction", "Expectation", nil, nil)
	newSubSubPlan.AddStep(newSubSubStep)

	newSubPlan := NewPlan("sub-plan")
	newSubStep := NewStep("sub-step", "Sub Step", "Instruction", "Expectation", nil, newSubSubPlan)
	newSubPlan.AddStep(newSubStep)

	newPlan := NewPlan("parent-plan")
	newStep := NewStep("step", "Step", "Instruction", "Expectation", nil, newSubPlan)
	newPlan.AddStep(newStep)

	diff := CreatePlanDiff(plan, newPlan, "Deep nesting test")
	if diff == nil {
		t.Fatal("CreatePlanDiff should not return nil")
	}
	if diff.From == nil || diff.To == nil {
		t.Fatal("Diff should have From and To pointers")
	}
	if len(diff.To.Steps) != 1 || diff.To.Steps[0].Plan == nil {
		t.Fatal("Nested structures should be preserved in diff")
	}

	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Failed to marshal deeply nested diff: %v", err)
	}

	var unmarshaled Diff[*Plan]
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal deeply nested diff: %v", err)
	}

	if unmarshaled.ID != diff.ID || unmarshaled.FromID != diff.FromID || unmarshaled.ToID != diff.ToID {
		t.Error("Nested diff serialization should preserve IDs")
	}
}

func TestGenericDiff(t *testing.T) {
	planDiff := Diff[*Plan]{ID: "plan-123", Reason: "Test plan revision"}
	if planDiff.ID != "plan-123" {
		t.Errorf("Expected plan diff ID 'plan-123', got %s", planDiff.ID)
	}
	if planDiff.Reason != "Test plan revision" {
		t.Errorf("Expected reason 'Test plan revision', got %s", planDiff.Reason)
	}

	stepDiff := Diff[*Step]{ID: "step-456", Reason: "Test step revision"}
	if stepDiff.ID != "step-456" {
		t.Errorf("Expected step diff ID 'step-456', got %s", stepDiff.ID)
	}
	if stepDiff.Reason != "Test step revision" {
		t.Errorf("Expected reason 'Test step revision', got %s", stepDiff.Reason)
	}
}

func TestPlanDiffEmbedding(t *testing.T) {
	oldPlan := NewPlan("plan-1")
	newPlan := NewPlan("plan-1")
	planDiff := &Diff[*Plan]{
		ID:     "plan-1",
		Reason: "Embedded diff test",
		From:   oldPlan,
		To:     newPlan,
		FromID: "plan-1",
		ToID:   "plan-1",
	}

	data, err := json.Marshal(planDiff)
	if err != nil {
		t.Fatalf("Failed to marshal Diff[*Plan]: %v", err)
	}

	var unmarshaled Diff[*Plan]
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Diff[*Plan]: %v", err)
	}

	if unmarshaled.ID != planDiff.ID || unmarshaled.Reason != planDiff.Reason {
		t.Error("Embedded fields should roundtrip correctly")
	}
}

func TestStepDiffEmbedding(t *testing.T) {
	oldStep := NewStep("step-1", "Old Name", "Instruction", "Expectation", nil, nil)
	newStep := NewStep("step-1", "New Name", "Instruction", "Expectation", nil, nil)
	stepDiff := &Diff[*Step]{
		ID:     "step-1",
		Reason: "Embedded diff test",
		From:   &oldStep,
		To:     &newStep,
		FromID: "step-1",
		ToID:   "step-1",
	}

	data, err := json.Marshal(stepDiff)
	if err != nil {
		t.Fatalf("Failed to marshal Diff[*Step]: %v", err)
	}

	var unmarshaled Diff[*Step]
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Diff[*Step]: %v", err)
	}

	if unmarshaled.ID != stepDiff.ID || unmarshaled.Reason != stepDiff.Reason {
		t.Error("Embedded fields should roundtrip correctly")
	}
}

func TestDiffableInterface(t *testing.T) {
	plan := NewPlan("test-plan")
	if plan.GetID() != "test-plan" {
		t.Errorf("Expected plan ID 'test-plan', got %s", plan.GetID())
	}

	step := NewStep("test-step", "Step", "Instruction", "Expectation", nil, nil)
	if step.GetID() != "test-step" {
		t.Errorf("Expected step ID 'test-step', got %s", step.GetID())
	}

	planDiff := Diff[*Plan]{ID: plan.GetID(), Reason: "Test"}
	if planDiff.ID != plan.GetID() {
		t.Errorf("Diff ID should match plan ID")
	}

	stepDiff := Diff[*Step]{ID: step.GetID(), Reason: "Test"}
	if stepDiff.ID != step.GetID() {
		t.Errorf("Diff ID should match step ID")
	}
}

func TestNestedDiffWithGeneric(t *testing.T) {
	oldSubPlan := NewPlan("sub-plan-1")
	newSubPlan := NewPlan("sub-plan-1")
	subStep := NewStep("sub-step-1", "Sub Step", "Instruction", "Expectation", nil, nil)
	newSubPlan.AddStep(subStep)

	oldStep := NewStep("step-1", "Old Step", "Instruction", "Expectation", nil, oldSubPlan)
	newStep := NewStep("step-1", "Updated Step", "Instruction", "Expectation", nil, newSubPlan)

	stepDiff := &Diff[*Step]{
		ID:     "step-1",
		Reason: "Step with nested plan diff",
		From:   &oldStep,
		To:     &newStep,
		FromID: "step-1",
		ToID:   "step-1",
	}

	data, err := json.Marshal(stepDiff)
	if err != nil {
		t.Fatalf("Failed to marshal nested diff: %v", err)
	}

	var unmarshaled Diff[*Step]
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal nested diff: %v", err)
	}

	if unmarshaled.ID != stepDiff.ID || unmarshaled.FromID != stepDiff.FromID || unmarshaled.ToID != stepDiff.ToID {
		t.Error("Nested diff serialization should preserve IDs")
	}
}

func TestDiffCreationWithGeneric(t *testing.T) {
	oldPlan := NewPlan("old-plan")
	step1 := NewStep("step1", "Step 1", "Instruction 1", "Expectation 1", nil, nil)
	oldPlan.AddStep(step1)

	newPlan := NewPlan("new-plan")
	step1Modified := NewStep("step1", "Step 1 Updated", "New Instruction", "Expectation 1", nil, nil)
	step2 := NewStep("step2", "Step 2", "Instruction 2", "Expectation 2", nil, nil)
	newPlan.AddStep(step1Modified)
	newPlan.AddStep(step2)

	diff := CreatePlanDiff(oldPlan, newPlan, "Testing generic diff creation")
	if diff == nil {
		t.Fatal("CreatePlanDiff should not return nil")
	}
	if diff.ID == "" {
		t.Error("Diff should have a unique ID")
	}
	if diff.FromID != "old-plan" || diff.ToID != "new-plan" {
		t.Errorf("Expected FromID='old-plan' and ToID='new-plan', got FromID=%s, ToID=%s", diff.FromID, diff.ToID)
	}
	if diff.Reason != "Testing generic diff creation" {
		t.Errorf("Expected reason 'Testing generic diff creation', got %s", diff.Reason)
	}
	if diff.From == nil || diff.To == nil {
		t.Error("Diff should have From and To pointers")
	}
}

func TestDiffApplicationWithGeneric(t *testing.T) {
	plan := NewPlan("plan-1")
	step1 := NewStep("step1", "Step 1", "Instruction", "Expectation", nil, nil)
	plan.AddStep(step1)

	newPlanObj := NewPlan("plan-1")
	step1Modified := NewStep("step1", "Step 1 Updated", "Instruction", "Expectation", nil, nil)
	step2 := NewStep("step2", "Step 2", "New Instruction", "New Expectation", nil, nil)
	newPlanObj.AddStep(step1Modified)
	newPlanObj.AddStep(step2)

	diff := &Diff[*Plan]{
		ID:     "plan-1",
		Reason: "Apply generic diff",
		From:   plan,
		To:     newPlanObj,
		FromID: "plan-1",
		ToID:   "plan-1",
	}

	// Use diff.To to get the new plan and set its RevisionDiff
	newPlan := diff.To
	newPlan.RevisionDiff = diff

	if newPlan.RevisionDiff.FromID != plan.ID {
		t.Errorf("Revision diff should reference old plan ID %s, got %s", plan.ID, newPlan.RevisionDiff.FromID)
	}
	if newPlan.RevisionDiff.Reason != "Apply generic diff" {
		t.Errorf("Expected reason 'Apply generic diff', got %s", newPlan.RevisionDiff.Reason)
	}
}
