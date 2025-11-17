package plan

import (
	"github.com/google/uuid"
)

// Diffable is an interface for types that can be diffed.
// It provides a way to get the ID of the object being diffed.
type Diffable interface {
	GetID() string
}

// Diff is a generic type that represents changes between two versions of a diffable object.
// T should be *Plan or *Step.
// It tracks metadata (unique diff ID, reason) and pointers to the old and new versions.
// The actual differences can be computed on-demand by comparing From and To.
type Diff[T Diffable] struct {
	// ID is a unique identifier for this diff itself.
	ID string `json:"id"`
	// Reason describes why this change was made.
	Reason string `json:"reason,omitempty"`
	// From is a pointer to the old version of the object.
	// This is not serialized to avoid circular references and large payloads.
	From T `json:"-"`
	// To is a pointer to the new version of the object.
	// This is not serialized to avoid circular references and large payloads.
	To T `json:"-"`
	// FromID is the ID of the old version (for serialization).
	FromID string `json:"from_id,omitempty"`
	// ToID is the ID of the new version (for serialization).
	ToID string `json:"to_id,omitempty"`
}

// CreateStepDiff creates a diff between two steps.
// Returns nil if the steps are identical (same pointer) or if either is nil.
// Can diff steps with different IDs (e.g., tracking replacement or evolution).
func CreateStepDiff(oldStep, newStep *Step, reason string) *Diff[*Step] {
	if oldStep == nil || newStep == nil {
		return nil
	}

	// Prevent creating a diff from an object to itself
	if oldStep == newStep {
		return nil
	}

	// Check if steps are identical in content (only for same-ID steps)
	if oldStep.ID == newStep.ID &&
		oldStep.Name == newStep.Name &&
		oldStep.Instruction == newStep.Instruction &&
		oldStep.Expectation == newStep.Expectation {
		return nil
	}

	return &Diff[*Step]{
		ID:     uuid.New().String(), // Unique ID for this diff
		Reason: reason,
		From:   oldStep,
		To:     newStep,
		FromID: oldStep.ID,
		ToID:   newStep.ID,
	}
}

// CreatePlanDiff creates a diff between two plans.
// Returns nil if the plans are identical (same pointer) or if either is nil.
// Can diff plans with different IDs (e.g., tracking replacement or evolution).
func CreatePlanDiff(oldPlan, newPlan *Plan, reason string) *Diff[*Plan] {
	if oldPlan == nil || newPlan == nil {
		return nil
	}

	// Prevent creating a diff from an object to itself
	if oldPlan == newPlan {
		return nil
	}

	return &Diff[*Plan]{
		ID:     uuid.New().String(), // Unique ID for this diff
		Reason: reason,
		From:   oldPlan,
		To:     newPlan,
		FromID: oldPlan.ID,
		ToID:   newPlan.ID,
	}
}

// planDiffSerializable is a serializable version of Diff[*Plan].
// It stores IDs instead of pointers to avoid circular references.
type planDiffSerializable struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
	FromID string `json:"from_id,omitempty"`
	ToID   string `json:"to_id,omitempty"`
}
