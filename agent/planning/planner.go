package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hlfshell/gogentic/pkg/agent"
	"github.com/hlfshell/gogentic/pkg/agent/plan"
	"github.com/hlfshell/gogentic/pkg/assets"
	"github.com/hlfshell/gogentic/pkg/model"
	"github.com/hlfshell/gogentic/pkg/prompt"
)

// PlannerAgent is an agent that creates structured plans from high-level objectives.
type PlannerAgent struct {
	*agent.Agent
	// promptTemplate is the cached prompt template for planning
	promptTemplate *prompt.Template
}

// ToolInfo represents information about a tool that can be used in plan steps.
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

// NewPlannerAgent creates a new planner agent with the default embedded prompt template.
// The prompt template is loaded from the embedded assets, so no external files are required.
func NewPlannerAgent(id, name, description string, config agent.AgentConfig) (*PlannerAgent, error) {
	if id == "" {
		id = uuid.New().String()
	}
	if name == "" {
		name = "Planner"
	}
	if description == "" {
		description = "An agent that creates structured plans from high-level objectives"
	}

	// Create the base agent
	baseAgent := agent.NewAgent(id, name, description, config)

	// Create the planner agent
	plannerAgent := &PlannerAgent{
		Agent: baseAgent,
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

// Plan creates a plan from the given objective and optional tools.
func (a *PlannerAgent) Plan(ctx context.Context, input PlannerInput) (*PlannerResult, error) {
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

	// Get the agent config
	config := a.Config()

	// Build the model messages
	messages := []model.Message{
		{
			Role: "user",
			Content: []model.Content{
				{
					Type: model.TextContent,
					Text: renderedPrompt,
				},
			},
		},
	}

	// Create the completion request
	request := model.CompletionRequest{
		Messages:    messages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
	}

	// Call the model
	response, err := config.Model.Complete(ctx, request)
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

	// Build a map to track steps as we create them
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

		stepMap[stepResp.ID] = &newStep
		newPlan.AddStep(newStep)
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
func (a *PlannerAgent) Replan(ctx context.Context, currentPlan *plan.Plan, feedback string, input PlannerInput) (*PlannerResult, error) {
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

// Execute implements the Agent interface for PlannerAgent.
func (a *PlannerAgent) ExecuteAgent(ctx context.Context, args agent.Arguments, options *agent.AgentOptions) (agent.AgentResult, error) {
	// Extract planning input from arguments
	objective := ""
	if objVal, ok := args["input"]; ok {
		if objStr, ok := objVal.(string); ok {
			objective = objStr
		}
	}
	input := PlannerInput{
		Objective: objective,
	}

	// Check for additional inputs
	if tools, ok := args["tools"].([]ToolInfo); ok {
		input.Tools = tools
	}
	if context, ok := args["context"].(string); ok {
		input.Context = context
	}

	// Record start time
	startTime := time.Now()

	// Create the plan
	result, err := a.Plan(ctx, input)
	if err != nil {
		return agent.AgentResult{}, fmt.Errorf("planning failed: %w", err)
	}

	// Record end time
	endTime := time.Now()

	// Convert the plan to text for the output
	planText := result.Plan.ToText()

	// Build the agent result
	agentResult := agent.AgentResult{
		Output: planText,
		AdditionalOutputs: map[string]interface{}{
			"plan":         result.Plan,
			"raw_response": result.RawResponse,
		},
		Conversation: nil, // Conversation management handled separately
		UsageStats:   result.UsageStats,
		ExecutionStats: agent.ExecutionStats{
			StartTime:  startTime,
			EndTime:    endTime,
			ToolCalls:  0, // Planner doesn't use tools
			Iterations: 1,
		},
		Message: agent.Message{
			Role:      "assistant",
			Content:   planText,
			Timestamp: endTime,
		},
	}

	return agentResult, nil
}
