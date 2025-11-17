package plan

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Plan represents a plan made up of steps.
type Plan struct {
	// ID is a unique identifier for the plan.
	ID string
	// Steps is the list of steps in the plan.
	Steps []Step
	// CreatedAt is when the plan was created.
	CreatedAt time.Time
	// RevisionDiff contains the diff from the previous plan if this is a revision/replan.
	// The diff's FromPlanID field contains the ID of the previous plan.
	// If nil, this is the original plan.
	RevisionDiff *PlanDiff
}

// NewPlan creates a new plan with the given ID. If id is empty, a new UUID is generated.
func NewPlan(id string) *Plan {
	if id == "" {
		id = uuid.New().String()
	}
	return &Plan{
		ID:           id,
		Steps:        []Step{},
		CreatedAt:    time.Now(),
		RevisionDiff: nil,
	}
}

// AddStep adds a step to the plan.
func (p *Plan) AddStep(step Step) {
	p.Steps = append(p.Steps, step)
}

// FindStep finds a step by ID. Returns a pointer to the step and true if found, or nil and false if not found.
func (p *Plan) FindStep(id string) (*Step, bool) {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i], true
		}
	}
	return nil, false
}

// NextSteps returns all steps that are ready to execute (have no dependencies or all dependencies are satisfied).
func (p *Plan) NextSteps(completedSteps map[string]bool) []Step {
	var ready []Step
	for _, step := range p.Steps {
		if step.AllDependenciesSatisfied(completedSteps) {
			ready = append(ready, step)
		}
	}
	return ready
}

// Validate validates the plan structure:
// - Checks that all dependency IDs reference existing steps
// - Checks for circular dependencies (basic check)
// - Recursively validates nested sub-plans
func (p *Plan) Validate() error {
	return p.validateWithContext(make(map[string]bool))
}

// validateWithContext validates the plan with a context to track visited plans and prevent infinite recursion.
func (p *Plan) validateWithContext(visitedPlans map[string]bool) error {
	if p == nil {
		return nil
	}

	// Check for circular plan references (plan containing itself through nested steps)
	if visitedPlans[p.ID] {
		return fmt.Errorf("circular plan reference detected: plan %s contains itself", p.ID)
	}
	visitedPlans[p.ID] = true
	defer delete(visitedPlans, p.ID)

	// Build a map of step IDs for quick lookup within this plan
	stepMap := make(map[string]bool)
	for _, step := range p.Steps {
		if step.ID == "" {
			return errors.New("step has empty ID")
		}
		if stepMap[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepMap[step.ID] = true

		// Recursively validate nested sub-plans
		if step.Plan != nil {
			if err := step.Plan.validateWithContext(visitedPlans); err != nil {
				return fmt.Errorf("step %s has invalid sub-plan: %w", step.ID, err)
			}
		}
	}

	// Validate all dependencies reference existing steps in this plan
	for _, step := range p.Steps {
		for _, dep := range step.Dependencies {
			if dep == nil {
				return fmt.Errorf("step %s has nil dependency", step.ID)
			}
			if !stepMap[dep.ID] {
				return fmt.Errorf("step %s has dependency on non-existent step: %s", step.ID, dep.ID)
			}
			// Check for self-dependency
			if dep.ID == step.ID {
				return fmt.Errorf("step %s depends on itself", step.ID)
			}
		}
	}

	// Check for circular dependencies using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(stepID string) bool
	hasCycle = func(stepID string) bool {
		visited[stepID] = true
		recStack[stepID] = true

		step, _ := p.FindStep(stepID)
		if step == nil {
			recStack[stepID] = false
			return false
		}
		for _, dep := range step.Dependencies {
			if dep == nil {
				continue
			}
			if !visited[dep.ID] {
				if hasCycle(dep.ID) {
					return true
				}
			} else if recStack[dep.ID] {
				return true
			}
		}

		recStack[stepID] = false
		return false
	}

	for _, step := range p.Steps {
		if !visited[step.ID] {
			if hasCycle(step.ID) {
				return fmt.Errorf("circular dependency detected involving step: %s", step.ID)
			}
		}
	}

	return nil
}

// GetExecutionOrder returns steps in a valid execution order (topological sort).
// Returns an error if the plan has circular dependencies or is invalid.
func (p *Plan) GetExecutionOrder() ([]Step, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// Initialize in-degree for all steps
	for _, step := range p.Steps {
		inDegree[step.ID] = 0
		graph[step.ID] = []string{}
	}

	// Build graph and calculate in-degrees
	for _, step := range p.Steps {
		for _, dep := range step.Dependencies {
			if dep == nil {
				continue
			}
			graph[dep.ID] = append(graph[dep.ID], step.ID)
			inDegree[step.ID]++
		}
	}

	// Topological sort using Kahn's algorithm
	var queue []string
	var result []Step

	// Find all steps with no dependencies
	for _, step := range p.Steps {
		if inDegree[step.ID] == 0 {
			queue = append(queue, step.ID)
		}
	}

	// Process queue
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		step, _ := p.FindStep(currentID)
		if step != nil {
			result = append(result, *step)
		}

		// Reduce in-degree of dependent steps
		for _, dependentID := range graph[currentID] {
			inDegree[dependentID]--
			if inDegree[dependentID] == 0 {
				queue = append(queue, dependentID)
			}
		}
	}

	// Check if all steps were processed (should be if no cycles)
	if len(result) != len(p.Steps) {
		return nil, errors.New("circular dependency detected: cannot determine execution order")
	}

	return result, nil
}

// GetAllStepsRecursive returns all steps in the plan and all nested sub-plans, flattened.
// The result includes steps from the current plan and all nested sub-plans at any depth.
func (p *Plan) GetAllStepsRecursive() []*Step {
	var allSteps []*Step
	p.collectStepsRecursive(&allSteps)
	return allSteps
}

// collectStepsRecursive is a helper that recursively collects all steps from the plan and nested sub-plans.
func (p *Plan) collectStepsRecursive(collector *[]*Step) {
	if p == nil {
		return
	}

	for i := range p.Steps {
		*collector = append(*collector, &p.Steps[i])
		// Recursively collect steps from sub-plans
		if p.Steps[i].Plan != nil {
			p.Steps[i].Plan.collectStepsRecursive(collector)
		}
	}
}

// GetMaxDepth returns the maximum nesting depth of the plan.
// A plan with no nested sub-plans has depth 1.
func (p *Plan) GetMaxDepth() int {
	if p == nil {
		return 0
	}

	maxDepth := 1
	for _, step := range p.Steps {
		if step.Plan != nil {
			subDepth := step.Plan.GetMaxDepth() + 1
			if subDepth > maxDepth {
				maxDepth = subDepth
			}
		}
	}
	return maxDepth
}

// planSerializable is a serializable version of Plan that uses step IDs instead of pointers.
// This is used for JSON marshaling/unmarshaling.
type planSerializable struct {
	ID           string             `json:"id"`
	Steps        []stepSerializable `json:"steps"`
	CreatedAt    time.Time          `json:"created_at"`
	RevisionDiff *PlanDiff          `json:"revision_diff,omitempty"`
}

// ToSerializable converts a Plan to its serializable representation.
func (p *Plan) ToSerializable() *planSerializable {
	if p == nil {
		return nil
	}

	steps := make([]stepSerializable, len(p.Steps))
	for i, step := range p.Steps {
		deps := make([]string, len(step.Dependencies))
		for j, dep := range step.Dependencies {
			if dep != nil {
				deps[j] = dep.ID
			}
		}
		var subPlan *planSerializable
		if step.Plan != nil {
			subPlan = step.Plan.ToSerializable()
		}
		steps[i] = stepSerializable{
			ID:           step.ID,
			Name:         step.Name,
			Instruction:  step.Instruction,
			Expectation:  step.Expectation,
			Dependencies: deps,
			SubPlan:      subPlan,
		}
	}

	return &planSerializable{
		ID:           p.ID,
		Steps:        steps,
		CreatedAt:    p.CreatedAt,
		RevisionDiff: p.RevisionDiff,
	}
}

// FromSerializable creates a Plan from its serializable representation.
func (ps *planSerializable) FromSerializable() (*Plan, error) {
	if ps == nil {
		return nil, nil
	}

	plan := &Plan{
		ID:           ps.ID,
		Steps:        make([]Step, len(ps.Steps)),
		CreatedAt:    ps.CreatedAt,
		RevisionDiff: nil,
	}

	// First, create all steps without dependencies and sub-plans
	for i, ss := range ps.Steps {
		plan.Steps[i] = Step{
			ID:           ss.ID,
			Name:         ss.Name,
			Instruction:  ss.Instruction,
			Expectation:  ss.Expectation,
			Dependencies: nil, // Will be set in next pass
			Plan:         nil, // Will be set in next pass
		}
	}

	// Build a map of step IDs to step pointers
	stepMap := make(map[string]*Step)
	for i := range plan.Steps {
		stepMap[plan.Steps[i].ID] = &plan.Steps[i]
	}

	// Now set up dependencies and sub-plans using pointers
	for i, ss := range ps.Steps {
		// Set up dependencies
		deps := make([]*Step, len(ss.Dependencies))
		for j, depID := range ss.Dependencies {
			dep, ok := stepMap[depID]
			if !ok {
				return nil, fmt.Errorf("dependency step ID not found: %s", depID)
			}
			deps[j] = dep
		}
		plan.Steps[i].Dependencies = deps

		// Set up sub-plan recursively
		if ss.SubPlan != nil {
			subPlan, err := ss.SubPlan.FromSerializable()
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize sub-plan for step %s: %w", ss.ID, err)
			}
			plan.Steps[i].Plan = subPlan
		}
	}

	// Handle revision diff
	plan.RevisionDiff = ps.RevisionDiff

	return plan, nil
}

// MarshalJSON implements json.Marshaler for Plan.
func (p *Plan) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.ToSerializable())
}

// UnmarshalJSON implements json.Unmarshaler for Plan.
func (p *Plan) UnmarshalJSON(data []byte) error {
	var ps planSerializable
	if err := json.Unmarshal(data, &ps); err != nil {
		return err
	}

	deserialized, err := ps.FromSerializable()
	if err != nil {
		return err
	}

	*p = *deserialized
	return nil
}

// AddStepByID adds a step to the plan and sets up dependencies by step IDs.
// This is a convenience method for building plans from serialized data.
// If subPlan is provided, it will be attached to the step.
func (p *Plan) AddStepByID(id, name, instruction, expectation string, dependencyIDs []string, subPlan *Plan) error {
	// Create the step
	step := Step{
		ID:           id,
		Name:         name,
		Instruction:  instruction,
		Expectation:  expectation,
		Dependencies: make([]*Step, len(dependencyIDs)),
		Plan:         subPlan,
	}

	// Set up dependencies by finding steps by ID
	for i, depID := range dependencyIDs {
		dep, found := p.FindStep(depID)
		if !found {
			return fmt.Errorf("dependency step not found: %s", depID)
		}
		step.Dependencies[i] = dep
	}

	p.AddStep(step)
	return nil
}

// ToText returns a human-friendly text representation of the plan.
func (p *Plan) ToText() string {
	if p == nil {
		return ""
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Plan: %s", p.ID))
	parts = append(parts, fmt.Sprintf("Created: %s", p.CreatedAt.Format(time.RFC3339)))

	if len(p.Steps) == 0 {
		parts = append(parts, "\nNo steps defined.")
	} else {
		parts = append(parts, fmt.Sprintf("\nSteps (%d):", len(p.Steps)))
		for i, step := range p.Steps {
			parts = append(parts, fmt.Sprintf("\n%d. %s", i+1, step.ToText()))
		}
	}

	return strings.Join(parts, "\n")
}
