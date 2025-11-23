package plan

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestChange(t *testing.T) {
	// Test the generic Change type
	change := Change[string]{From: "old", To: "new"}
	if change.From != "old" || change.To != "new" {
		t.Error("Change should store from/to values")
	}

	// Test with struct
	type Config struct {
		Value int
	}
	configChange := Change[Config]{From: Config{Value: 1}, To: Config{Value: 2}}
	if configChange.From.Value != 1 || configChange.To.Value != 2 {
		t.Error("Change should work with structs")
	}
}

func TestDelta(t *testing.T) {
	// Test empty delta
	delta := Delta[string, int]{
		Added:   make(map[string]int),
		Removed: make(map[string]int),
		Changed: make(map[string]Change[int]),
	}

	if len(delta.Added) != 0 || len(delta.Removed) != 0 || len(delta.Changed) != 0 {
		t.Error("Empty delta should have empty maps")
	}

	// Test with data
	delta.Added["new"] = 42
	delta.Removed["old"] = 10
	delta.Changed["modified"] = Change[int]{From: 1, To: 2}

	if delta.Added["new"] != 42 {
		t.Error("Delta.Added should store added items")
	}
	if delta.Removed["old"] != 10 {
		t.Error("Delta.Removed should store removed items")
	}
	if delta.Changed["modified"].From != 1 || delta.Changed["modified"].To != 2 {
		t.Error("Delta.Changed should store changes")
	}
}

func TestComputeStepDelta_Empty(t *testing.T) {
	// Both plans nil
	delta := ComputeStepDelta(nil, nil)
	if len(delta.Added) != 0 || len(delta.Removed) != 0 || len(delta.Changed) != 0 {
		t.Error("Delta from nil plans should be empty")
	}

	// Old plan nil
	newPlan := NewPlan("new")
	step := NewStep("s1", "Step", "Do", "Expect", nil, nil)
	newPlan.AddStep(step)

	delta = ComputeStepDelta(nil, newPlan)
	if len(delta.Added) != 1 || delta.Added["s1"].ID != "s1" {
		t.Error("Delta should show step as added when old plan is nil")
	}

	// New plan nil
	oldPlan := NewPlan("old")
	oldPlan.AddStep(step)

	delta = ComputeStepDelta(oldPlan, nil)
	if len(delta.Removed) != 1 || delta.Removed["s1"].ID != "s1" {
		t.Error("Delta should show step as removed when new plan is nil")
	}
}

func TestComputeStepDelta_Added(t *testing.T) {
	oldPlan := NewPlan("old")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	oldPlan.AddStep(step1)

	newPlan := NewPlan("new")
	newPlan.AddStep(step1)
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	newPlan.AddStep(step2)

	delta := ComputeStepDelta(oldPlan, newPlan)

	if len(delta.Added) != 1 {
		t.Errorf("Expected 1 added step, got %d", len(delta.Added))
	}
	if delta.Added["s2"].ID != "s2" {
		t.Error("Added step should be s2")
	}
	if len(delta.Removed) != 0 {
		t.Error("Should have no removed steps")
	}
	if len(delta.Changed) != 0 {
		t.Error("Should have no changed steps")
	}
}

func TestComputeStepDelta_Removed(t *testing.T) {
	oldPlan := NewPlan("old")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	oldPlan.AddStep(step1)
	oldPlan.AddStep(step2)

	newPlan := NewPlan("new")
	newPlan.AddStep(step1)

	delta := ComputeStepDelta(oldPlan, newPlan)

	if len(delta.Removed) != 1 {
		t.Errorf("Expected 1 removed step, got %d", len(delta.Removed))
	}
	if delta.Removed["s2"].ID != "s2" {
		t.Error("Removed step should be s2")
	}
	if len(delta.Added) != 0 {
		t.Error("Should have no added steps")
	}
	if len(delta.Changed) != 0 {
		t.Error("Should have no changed steps")
	}
}

func TestComputeStepDelta_Changed(t *testing.T) {
	oldPlan := NewPlan("old")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	oldPlan.AddStep(step1)

	newPlan := NewPlan("new")
	step1Modified := NewStep("s1", "Step 1 Modified", "Do 1 Updated", "Expect 1", nil, nil)
	newPlan.AddStep(step1Modified)

	delta := ComputeStepDelta(oldPlan, newPlan)

	if len(delta.Changed) != 1 {
		t.Errorf("Expected 1 changed step, got %d", len(delta.Changed))
	}
	change, ok := delta.Changed["s1"]
	if !ok {
		t.Fatal("Changed step s1 not found")
	}
	if change.From.Name != "Step 1" || change.To.Name != "Step 1 Modified" {
		t.Error("Change should capture old and new step versions")
	}
	if len(delta.Added) != 0 || len(delta.Removed) != 0 {
		t.Error("Should only have changed steps")
	}
}

func TestComputeStepDelta_Mixed(t *testing.T) {
	oldPlan := NewPlan("old")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	step3 := NewStep("s3", "Step 3", "Do 3", "Expect 3", nil, nil)
	oldPlan.AddStep(step1)
	oldPlan.AddStep(step2)
	oldPlan.AddStep(step3)

	newPlan := NewPlan("new")
	step1Modified := NewStep("s1", "Step 1 Updated", "Do 1", "Expect 1", nil, nil)
	step4 := NewStep("s4", "Step 4", "Do 4", "Expect 4", nil, nil)
	newPlan.AddStep(step1Modified) // s1 changed
	newPlan.AddStep(step2)          // s2 unchanged
	// s3 removed
	newPlan.AddStep(step4) // s4 added

	delta := ComputeStepDelta(oldPlan, newPlan)

	if len(delta.Added) != 1 || delta.Added["s4"].ID != "s4" {
		t.Error("s4 should be added")
	}
	if len(delta.Removed) != 1 || delta.Removed["s3"].ID != "s3" {
		t.Error("s3 should be removed")
	}
	if len(delta.Changed) != 1 {
		t.Error("s1 should be changed")
	}
}

func TestStepsEqual(t *testing.T) {
	// Identical steps
	step1 := NewStep("s1", "Step", "Do", "Expect", nil, nil)
	step2 := NewStep("s1", "Step", "Do", "Expect", nil, nil)

	if !stepsEqual(step1, step2) {
		t.Error("Identical steps should be equal")
	}

	// Different names
	step3 := NewStep("s1", "Different", "Do", "Expect", nil, nil)
	if stepsEqual(step1, step3) {
		t.Error("Steps with different names should not be equal")
	}

	// Different instructions
	step4 := NewStep("s1", "Step", "Different", "Expect", nil, nil)
	if stepsEqual(step1, step4) {
		t.Error("Steps with different instructions should not be equal")
	}

	// Different expectations
	step5 := NewStep("s1", "Step", "Do", "Different", nil, nil)
	if stepsEqual(step1, step5) {
		t.Error("Steps with different expectations should not be equal")
	}
}

func TestStepsEqual_Dependencies(t *testing.T) {
	dep1 := NewStep("d1", "Dep 1", "Do", "Expect", nil, nil)
	dep2 := NewStep("d2", "Dep 2", "Do", "Expect", nil, nil)

	step1 := NewStep("s1", "Step", "Do", "Expect", []*Step{&dep1}, nil)
	step2 := NewStep("s1", "Step", "Do", "Expect", []*Step{&dep1}, nil)
	step3 := NewStep("s1", "Step", "Do", "Expect", []*Step{&dep2}, nil)
	step4 := NewStep("s1", "Step", "Do", "Expect", []*Step{&dep1, &dep2}, nil)

	if !stepsEqual(step1, step2) {
		t.Error("Steps with same dependencies should be equal")
	}
	if stepsEqual(step1, step3) {
		t.Error("Steps with different dependencies should not be equal")
	}
	if stepsEqual(step1, step4) {
		t.Error("Steps with different number of dependencies should not be equal")
	}
}

func TestStepsEqual_NestedPlan(t *testing.T) {
	subPlan1 := NewPlan("sub1")
	subStep1 := NewStep("ss1", "Sub Step", "Do", "Expect", nil, nil)
	subPlan1.AddStep(subStep1)

	subPlan2 := NewPlan("sub2")
	subStep2 := NewStep("ss1", "Sub Step Modified", "Do", "Expect", nil, nil)
	subPlan2.AddStep(subStep2)

	step1 := NewStep("s1", "Step", "Do", "Expect", nil, subPlan1)
	step2 := NewStep("s1", "Step", "Do", "Expect", nil, subPlan1)
	step3 := NewStep("s1", "Step", "Do", "Expect", nil, subPlan2)
	step4 := NewStep("s1", "Step", "Do", "Expect", nil, nil)

	if !stepsEqual(step1, step2) {
		t.Error("Steps with same nested plans should be equal")
	}
	if stepsEqual(step1, step3) {
		t.Error("Steps with different nested plans should not be equal")
	}
	if stepsEqual(step1, step4) {
		t.Error("Steps with and without nested plans should not be equal")
	}
}

func TestNewPlanDiff(t *testing.T) {
	oldPlan := NewPlan("plan-v1")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	oldPlan.AddStep(step1)

	newPlan := NewPlan("plan-v2")
	step1Modified := NewStep("s1", "Step 1 Updated", "Do 1", "Expect 1", nil, nil)
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	newPlan.AddStep(step1Modified)
	newPlan.AddStep(step2)

	diffID := uuid.New().String()
	diff := NewPlanDiff(diffID, oldPlan, newPlan, "Added new step and updated existing step")

	if diff.ID != diffID {
		t.Errorf("Expected diff ID %s, got %s", diffID, diff.ID)
	}
	if diff.FromPlanID != "plan-v1" {
		t.Errorf("Expected FromPlanID 'plan-v1', got %s", diff.FromPlanID)
	}
	if diff.ToPlanID != "plan-v2" {
		t.Errorf("Expected ToPlanID 'plan-v2', got %s", diff.ToPlanID)
	}
	if diff.Reason != "Added new step and updated existing step" {
		t.Errorf("Expected reason 'Added new step and updated existing step', got %s", diff.Reason)
	}

	// Check the delta
	if len(diff.Steps.Added) != 1 || diff.Steps.Added["s2"].ID != "s2" {
		t.Error("Diff should show s2 as added")
	}
	if len(diff.Steps.Changed) != 1 {
		t.Error("Diff should show s1 as changed")
	}
	if len(diff.Steps.Removed) != 0 {
		t.Error("Diff should have no removed steps")
	}
}

func TestNewPlanDiff_NilPlans(t *testing.T) {
	// Both nil
	diff := NewPlanDiff("diff-1", nil, nil, "Test")
	if diff.FromPlanID != "" || diff.ToPlanID != "" {
		t.Error("Diff from nil plans should have empty plan IDs")
	}
	if len(diff.Steps.Added) != 0 || len(diff.Steps.Removed) != 0 || len(diff.Steps.Changed) != 0 {
		t.Error("Diff from nil plans should have empty delta")
	}

	// Old nil, new has steps
	newPlan := NewPlan("plan-1")
	step := NewStep("s1", "Step", "Do", "Expect", nil, nil)
	newPlan.AddStep(step)

	diff = NewPlanDiff("diff-2", nil, newPlan, "Initial plan")
	if diff.FromPlanID != "" || diff.ToPlanID != "plan-1" {
		t.Error("Diff should have empty from and plan-1 to")
	}
	if len(diff.Steps.Added) != 1 {
		t.Error("All steps should be added when old plan is nil")
	}

	// New nil, old has steps
	oldPlan := NewPlan("plan-2")
	oldPlan.AddStep(step)

	diff = NewPlanDiff("diff-3", oldPlan, nil, "Removed all")
	if diff.FromPlanID != "plan-2" || diff.ToPlanID != "" {
		t.Error("Diff should have plan-2 from and empty to")
	}
	if len(diff.Steps.Removed) != 1 {
		t.Error("All steps should be removed when new plan is nil")
	}
}

func TestPlanDiff_Serialization(t *testing.T) {
	oldPlan := NewPlan("v1")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	oldPlan.AddStep(step1)

	newPlan := NewPlan("v2")
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	newPlan.AddStep(step2)

	diff := NewPlanDiff("diff-123", oldPlan, newPlan, "Complete replacement")

	// Serialize
	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Failed to marshal PlanDiff: %v", err)
	}

	// Deserialize
	var unmarshaled PlanDiff
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal PlanDiff: %v", err)
	}

	// Verify
	if unmarshaled.ID != diff.ID {
		t.Errorf("Expected ID %s, got %s", diff.ID, unmarshaled.ID)
	}
	if unmarshaled.FromPlanID != diff.FromPlanID {
		t.Errorf("Expected FromPlanID %s, got %s", diff.FromPlanID, unmarshaled.FromPlanID)
	}
	if unmarshaled.ToPlanID != diff.ToPlanID {
		t.Errorf("Expected ToPlanID %s, got %s", diff.ToPlanID, unmarshaled.ToPlanID)
	}
	if unmarshaled.Reason != diff.Reason {
		t.Errorf("Expected Reason %s, got %s", diff.Reason, unmarshaled.Reason)
	}
	if len(unmarshaled.Steps.Added) != 1 || len(unmarshaled.Steps.Removed) != 1 {
		t.Error("Delta should be preserved through serialization")
	}
}

func TestPlanWithRevisionDiff(t *testing.T) {
	// Create original plan
	v1 := NewPlan("plan-v1")
	step1 := NewStep("s1", "Step 1", "Do 1", "Expect 1", nil, nil)
	v1.AddStep(step1)

	// Create revised plan
	v2 := NewPlan("plan-v2")
	step1Modified := NewStep("s1", "Step 1 Updated", "Do 1 Modified", "Expect 1", nil, nil)
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	v2.AddStep(step1Modified)
	v2.AddStep(step2)

	// Create diff and attach to plan
	diff := NewPlanDiff(uuid.New().String(), v1, v2, "Updated step 1 and added step 2")
	v2.RevisionDiff = &diff

	// Verify
	if v2.RevisionDiff == nil {
		t.Fatal("Plan should have revision diff")
	}
	if v2.RevisionDiff.FromPlanID != "plan-v1" {
		t.Error("Revision diff should reference previous plan")
	}
	if len(v2.RevisionDiff.Steps.Added) != 1 {
		t.Error("Revision diff should show added step")
	}
	if len(v2.RevisionDiff.Steps.Changed) != 1 {
		t.Error("Revision diff should show changed step")
	}
}

func TestPlanSerialization_WithRevisionDiff(t *testing.T) {
	v1 := NewPlan("v1")
	step1 := NewStep("s1", "Step", "Do", "Expect", nil, nil)
	v1.AddStep(step1)

	v2 := NewPlan("v2")
	step2 := NewStep("s2", "Step 2", "Do 2", "Expect 2", nil, nil)
	v2.AddStep(step2)

	diff := NewPlanDiff("diff-1", v1, v2, "Changed everything")
	v2.RevisionDiff = &diff

	// Serialize
	data, err := json.Marshal(v2)
	if err != nil {
		t.Fatalf("Failed to marshal plan: %v", err)
	}

	// Deserialize
	var unmarshaled Plan
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal plan: %v", err)
	}

	// Verify
	if unmarshaled.ID != "v2" {
		t.Errorf("Expected plan ID 'v2', got %s", unmarshaled.ID)
	}
	if unmarshaled.RevisionDiff == nil {
		t.Fatal("Plan should have revision diff after deserialization")
	}
	if unmarshaled.RevisionDiff.FromPlanID != "v1" {
		t.Error("Revision diff should be preserved")
	}
}
