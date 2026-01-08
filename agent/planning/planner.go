package planning

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hlfshell/gotonomy/assets"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/plan"
	"github.com/hlfshell/gotonomy/prompt"
	"github.com/hlfshell/gotonomy/tool"
)

// PlannerAgent creates structured plans from high-level objectives.
//
// NOTE: This is intentionally a lightweight “agent”:
// - It does not run the iterative tool-calling loop from `agent.Agent`.
// - It renders a planning prompt, calls `model.Model.Complete`, and parses a `plan.Plan`.
type PlannerAgent struct {
	id          string
	name        string
	description string

	model       model.Model
	temperature float32

	// promptTemplate is the cached prompt template for planning.
	promptTemplate *prompt.Template
}

// ToolInfo represents information about a tool that can be used in plan steps.
// TODO - this seems unneeded given tools already have this info...
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UsageNotes  string `json:"usage_notes,omitempty"`
}

// PlannerInput represents the input to the planner agent.
type PlannerInput struct {
	// Objective is the high-level goal to plan for
	Objective string
	// Tools is an optional list of tools available for use in the plan
	Tools []ToolInfo
	// Context provides additional context for planning
	Context string
}

// PlannerResult represents the result of planning.
type PlannerResult struct {
	// Plan is the generated plan
	Plan *plan.Plan
	// RawResponse is the raw LLM response
	RawResponse string
	// UsageStats contains token usage information
	UsageStats model.UsageStats
}

// Config configures a PlannerAgent.
type Config struct {
	Model       model.Model
	Temperature float32
}

// NewPlannerAgent creates a new planner agent with the default embedded prompt template.
// The prompt template is loaded from embedded assets, so no external files are required.
func NewPlannerAgent(id, name, description string, config Config) (*PlannerAgent, error) {
	if id == "" {
		id = uuid.New().String()
	}
	if name == "" {
		name = "Planner"
	}
	if description == "" {
		description = "An agent that creates structured plans from high-level objectives"
	}
	if config.Model == nil {
		return nil, fmt.Errorf("planner config: Model is required")
	}
	if config.Temperature < 0 || config.Temperature > 1 {
		return nil, fmt.Errorf("planner config: Temperature must be between 0 and 1")
	}

	// Create the planner agent
	plannerAgent := &PlannerAgent{
		id:          id,
		name:        name,
		description: description,
		model:       config.Model,
		temperature: config.Temperature,
	}

	// Load the default embedded prompt template
	if err := plannerAgent.LoadEmbeddedPrompt(); err != nil {
		return nil, fmt.Errorf("failed to load default prompt template: %w", err)
	}

	return plannerAgent, nil
}

// LoadEmbeddedPrompt loads the default planner prompt template from embedded assets.
// This is the recommended way to load the prompt and is called automatically by NewPlannerAgent.
func (a *PlannerAgent) LoadEmbeddedPrompt() error {
	tmpl, err := assets.LoadPrompt("planner.prompt")
	if err != nil {
		return fmt.Errorf("failed to load embedded planner prompt: %w", err)
	}
	a.promptTemplate = tmpl
	return nil
}

// LoadPromptTemplate loads a planner prompt template from a file path.
// This method is provided for custom prompt templates. For the default prompt,
// use LoadEmbeddedPrompt() or just call NewPlannerAgent() which loads it automatically.
func (a *PlannerAgent) LoadPromptTemplate(templatePath string) error {
	tmpl, err := prompt.LoadTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to load planner prompt template: %w", err)
	}
	a.promptTemplate = tmpl
	return nil
}

// SetPromptTemplate sets the prompt template directly.
func (a *PlannerAgent) SetPromptTemplate(tmpl *prompt.Template) {
	a.promptTemplate = tmpl
}

func (a *PlannerAgent) ID() string { return a.id }
func (a *PlannerAgent) Name() string {
	return a.name
}
func (a *PlannerAgent) Description() string {
	return a.description
}

// Plan creates a plan from the given objective and optional tools.
func (a *PlannerAgent) Plan(ctx *tool.Context, input PlannerInput) (*PlannerResult, error) {
	if a.promptTemplate == nil {
		return nil, fmt.Errorf("prompt template not loaded - call LoadPromptTemplate first")
	}

	// Prepare the template data
	templateData := map[string]interface{}{
		"objective": input.Objective,
		"tools":     input.Tools,
		"context":   input.Context,
	}

	// Render the prompt
	renderedPrompt, err := a.promptTemplate.Render(templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	// Build the model messages
	messages := []model.Message{
		{
			Role:    model.RoleUser,
			Content: renderedPrompt,
		},
	}

	// Create the completion request
	request := model.CompletionRequest{
		Messages: messages,
		Config: model.ModelConfig{
			Temperature: a.temperature,
		},
	}

	// Call the model
	response, err := a.model.Complete(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion from model: %w", err)
	}

	// Parse the response into a plan
	generatedPlan, err := a.parsePlanFromResponse(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan from response: %w", err)
	}

	return &PlannerResult{
		Plan:        generatedPlan,
		RawResponse: response.Text,
		UsageStats:  response.UsageStats,
	}, nil
}

// parsePlanFromResponse parses the LLM response into a Plan structure.
func (a *PlannerAgent) parsePlanFromResponse(responseText string) (*plan.Plan, error) {
	// Clean the response - remove markdown code blocks if present
	cleaned := cleanJSONResponse(responseText)

	// Parse the JSON into a planResponse structure
	var planResp planResponse
	if err := json.Unmarshal([]byte(cleaned), &planResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w\nResponse: %s", err, cleaned)
	}

	// Convert the planResponse to a Plan
	generatedPlan, err := a.buildPlanFromResponse(planResp)
	if err != nil {
		return nil, fmt.Errorf("failed to build plan from response: %w", err)
	}

	// Validate the plan
	if err := generatedPlan.Validate(); err != nil {
		return nil, fmt.Errorf("generated plan failed validation: %w", err)
	}

	return generatedPlan, nil
}

// planResponse represents the JSON structure returned by the LLM.
type planResponse struct {
	Steps []stepResponse `json:"steps"`
}

// stepResponse represents a step in the JSON response.
type stepResponse struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Instruction  string        `json:"instruction"`
	Expectation  string        `json:"expectation"`
	Dependencies []string      `json:"dependencies"`
	SubPlan      *planResponse `json:"sub_plan,omitempty"`
}

// buildPlanFromResponse converts a planResponse to a Plan.
func (a *PlannerAgent) buildPlanFromResponse(resp planResponse) (*plan.Plan, error) {
	// Create a new plan
	newPlan := plan.NewPlan("")

	// Build a map to track step pointers within this plan.
	stepMap := make(map[string]*plan.Step)

	// First pass: create all steps without dependencies
	for _, stepResp := range resp.Steps {
		// Handle sub-plan if present
		var subPlan *plan.Plan
		if stepResp.SubPlan != nil {
			var err error
			subPlan, err = a.buildPlanFromResponse(*stepResp.SubPlan)
			if err != nil {
				return nil, fmt.Errorf("failed to build sub-plan for step %s: %w", stepResp.ID, err)
			}
		}

		// Create the step
		newStep := plan.NewStep(
			stepResp.ID,
			stepResp.Name,
			stepResp.Instruction,
			stepResp.Expectation,
			nil, // Dependencies will be set in second pass
			subPlan,
		)

		newPlan.AddStep(newStep)
	}

	// Now that steps are in the slice, map IDs to the slice-backed pointers.
	for i := range newPlan.Steps {
		stepMap[newPlan.Steps[i].ID] = &newPlan.Steps[i]
	}

	// Second pass: set up dependencies using the map
	for i, stepResp := range resp.Steps {
		if len(stepResp.Dependencies) > 0 {
			deps := make([]*plan.Step, len(stepResp.Dependencies))
			for j, depID := range stepResp.Dependencies {
				dep, exists := stepMap[depID]
				if !exists {
					return nil, fmt.Errorf("step %s has dependency on non-existent step: %s", stepResp.ID, depID)
				}
				deps[j] = dep
			}
			newPlan.Steps[i].Dependencies = deps
		}
	}

	return newPlan, nil
}

// cleanJSONResponse removes markdown code blocks and other formatting from the response.
func cleanJSONResponse(response string) string {
	// Trim whitespace
	cleaned := strings.TrimSpace(response)

	// Remove markdown code blocks
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}

	// Trim again after removing code blocks
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// Replan creates a revised plan based on feedback or new information.
func (a *PlannerAgent) Replan(ctx *tool.Context, currentPlan *plan.Plan, feedback string, input PlannerInput) (*PlannerResult, error) {
	if a.promptTemplate == nil {
		return nil, fmt.Errorf("prompt template not loaded - call LoadPromptTemplate first")
	}

	// Build context that includes the current plan and feedback
	enhancedContext := fmt.Sprintf("Current Plan:\n%s\n\nFeedback/Changes Needed:\n%s", currentPlan.ToText(), feedback)
	if input.Context != "" {
		enhancedContext = fmt.Sprintf("%s\n\nAdditional Context:\n%s", enhancedContext, input.Context)
	}

	// Update the input context
	input.Context = enhancedContext

	// Create the new plan
	result, err := a.Plan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create revised plan: %w", err)
	}

	// Create a diff between the old and new plan
	diffID := uuid.New().String()
	diff := plan.NewPlanDiff(diffID, currentPlan, result.Plan, feedback)
	result.Plan.RevisionDiff = &diff

	return result, nil
}
