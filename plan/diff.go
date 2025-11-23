package plan

// Change represents a change of a single value from one version to another.
type Change[T any] struct {
	From T `json:"from"`
	To   T `json:"to"`
}

// Delta represents added, removed, and changed items in a keyed collection.
type Delta[ID comparable, T any] struct {
	Added   map[ID]T         `json:"added"`
	Removed map[ID]T         `json:"removed"`
	Changed map[ID]Change[T] `json:"changed"`
}

// PlanDiff describes how a plan changed between two versions.
// It is the only diff type used for tracking plan evolution.
type PlanDiff struct {
	// ID is the identifier of this diff record (e.g., a UUID).
	ID string `json:"id"`
	// FromPlanID is the ID of the previous plan snapshot.
	// It may be empty if this is the first plan.
	FromPlanID string `json:"from_plan_id"`
	// ToPlanID is the ID of the new plan snapshot.
	ToPlanID string `json:"to_plan_id"`
	// Reason is the model's explanation for why it changed the plan.
	Reason string `json:"reason"`
	// Steps describes which steps were added, removed, or changed
	// between the previous plan and the new plan.
	Steps Delta[string, Step] `json:"steps_delta"`
}

// NewPlanDiff constructs a PlanDiff for the transition from oldPlan to newPlan.
// Either plan may be nil; a nil plan is treated as an empty plan.
func NewPlanDiff(id string, oldPlan, newPlan *Plan, reason string) PlanDiff {
	var fromID, toID string
	if oldPlan != nil {
		fromID = oldPlan.ID
	}
	if newPlan != nil {
		toID = newPlan.ID
	}

	stepsDelta := ComputeStepDelta(oldPlan, newPlan)

	return PlanDiff{
		ID:         id,
		FromPlanID: fromID,
		ToPlanID:   toID,
		Reason:     reason,
		Steps:      stepsDelta,
	}
}

// ComputeStepDelta computes the added, removed, and changed steps
// between two plans. A nil plan is treated as a plan with no steps.
func ComputeStepDelta(oldP, newP *Plan) Delta[string, Step] {
	out := Delta[string, Step]{
		Added:   make(map[string]Step),
		Removed: make(map[string]Step),
		Changed: make(map[string]Change[Step]),
	}

	oldMap := map[string]Step{}
	newMap := map[string]Step{}

	if oldP != nil {
		for _, s := range oldP.Steps {
			oldMap[s.ID] = s
		}
	}
	if newP != nil {
		for _, s := range newP.Steps {
			newMap[s.ID] = s
		}
	}

	// Removed or changed
	for id, oldS := range oldMap {
		newS, exists := newMap[id]
		if !exists {
			out.Removed[id] = oldS
			continue
		}
		if !stepsEqual(oldS, newS) {
			out.Changed[id] = Change[Step]{From: oldS, To: newS}
		}
	}

	// Added
	for id, newS := range newMap {
		if _, existed := oldMap[id]; !existed {
			out.Added[id] = newS
		}
	}

	return out
}

// stepsEqual defines when two steps are considered "the same"
// for the purposes of plan diffs. Adjust as needed if you add fields.
func stepsEqual(a, b Step) bool {
	if a.ID != b.ID ||
		a.Name != b.Name ||
		a.Instruction != b.Instruction ||
		a.Expectation != b.Expectation {
		return false
	}

	if len(a.Dependencies) != len(b.Dependencies) {
		return false
	}
	for i := range a.Dependencies {
		if a.Dependencies[i] != b.Dependencies[i] {
			return false
		}
	}

	// Nested plan: treat any change in sub-plan text as a step change.
	switch {
	case a.Plan == nil && b.Plan == nil:
		return true
	case a.Plan == nil || b.Plan == nil:
		return false
	default:
		return a.Plan.ToText() == b.Plan.ToText()
	}
}
