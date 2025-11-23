package plan

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Step represents a single step in a plan.
type Step struct {
	// ID is a unique identifier for the step.
	ID string
	// Name is the name of the step.
	Name string
	// Instruction is the instruction for executing this step.
	Instruction string
	// Expectation describes what is expected from this step.
	Expectation string
	// Dependencies is a list of pointers to steps that must be completed before this step can execute.
	// If empty, the step has no dependencies and can execute immediately.
	Dependencies []*Step
	// Plan is an optional nested plan that this step contains
	Plan *Plan
}

// NewStep creates a new step with the given parameters. If id is empty, a new UUID is generated.
func NewStep(id, name, instruction, expectation string, dependencies []*Step, plan *Plan) Step {
	if id == "" {
		id = uuid.New().String()
	}
	return Step{
		ID:           id,
		Name:         name,
		Instruction:  instruction,
		Expectation:  expectation,
		Dependencies: dependencies,
		Plan:         plan,
	}
}

// AllDependenciesSatisfied checks if all dependencies for this step are satisfied.
func (s *Step) AllDependenciesSatisfied(completedSteps map[string]bool) bool {
	// If there are no dependencies, the step is ready to execute
	if len(s.Dependencies) == 0 {
		return true
	}
	// Check that all dependencies are completed
	for _, dep := range s.Dependencies {
		if dep == nil {
			return false
		}
		if !completedSteps[dep.ID] {
			return false
		}
	}
	return true
}

// ToText returns a human-friendly text representation of the step.
func (s *Step) ToText() string {
	if s == nil {
		return ""
	}

	var parts []string
	parts = append(parts, s.Name)
	parts = append(parts, fmt.Sprintf("  ID: %s", s.ID))
	parts = append(parts, fmt.Sprintf("  Instruction: %s", s.Instruction))
	parts = append(parts, fmt.Sprintf("  Expected: %s", s.Expectation))

	if len(s.Dependencies) > 0 {
		depNames := make([]string, 0, len(s.Dependencies))
		for _, dep := range s.Dependencies {
			if dep != nil {
				depNames = append(depNames, fmt.Sprintf("%s (%s)", dep.Name, dep.ID))
			}
		}
		parts = append(parts, fmt.Sprintf("  Depends on: %s", strings.Join(depNames, ", ")))
	}

	if s.Plan != nil {
		parts = append(parts, "  Contains sub-plan:")
		subParts := strings.Split(s.Plan.ToText(), "\n")
		for _, line := range subParts {
			if line != "" {
				parts = append(parts, "    "+line)
			}
		}
	}

	return strings.Join(parts, "\n")
}

// stepSerializable is a serializable version of Step that uses dependency IDs instead of pointers.
type stepSerializable struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Instruction  string            `json:"instruction"`
	Expectation  string            `json:"expectation"`
	Dependencies []string          `json:"dependencies"`
	SubPlan      *planSerializable `json:"sub_plan,omitempty"`
}
