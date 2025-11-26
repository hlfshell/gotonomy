# Planning Agent

The Planning Agent is a specialized agent that creates structured, executable plans from high-level objectives. It uses LLMs to break down complex goals into concrete steps with dependencies, expectations, and optional nested sub-plans.

## Features

- **Embedded Prompts**: Prompt templates are embedded in the binary - no external files needed!
- **Structured Planning**: Converts high-level objectives into detailed, executable plans
- **Dependency Management**: Automatically manages step dependencies and validates DAG structure
- **Nested Sub-Plans**: Supports hierarchical planning with nested sub-plans for complex phases
- **Tool Integration**: Incorporates available tools into plan steps
- **Plan Revision**: Supports replanning based on feedback or new information
- **Validation**: Automatically validates plans for consistency and correctness
- **Serialization**: Full JSON serialization support for storing and loading plans

## Usage

### Basic Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hlfshell/gogentic/pkg/agent"
	"github.com/hlfshell/gogentic/pkg/agent/planning"
	"github.com/hlfshell/gogentic/pkg/provider/openai"
)

func main() {
	// Create an OpenAI provider
	provider := openai.NewOpenAIProvider("your-api-key")
	
	// Get a model
	model, err := provider.GetModel(context.Background(), "gpt-4")
	if err != nil {
		log.Fatal(err)
	}

	// Configure the planner agent
	config := agent.AgentConfig{
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	// Create the planner agent (automatically loads embedded prompt template)
	planner, err := planning.NewPlannerAgent("planner-1", "Project Planner", "Plans software projects", config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a plan
	input := planning.PlannerInput{
		Objective: "Build a web application for task management",
		Context:   "The application should support user authentication, task CRUD operations, and real-time updates",
	}

	result, err := planner.Plan(context.Background(), input)
	if err != nil {
		log.Fatal(err)
	}

	// Print the plan
	fmt.Println(result.Plan.ToText())
	
	// Access individual steps
	for _, step := range result.Plan.Steps {
		fmt.Printf("\nStep: %s\n", step.Name)
		fmt.Printf("Instruction: %s\n", step.Instruction)
		fmt.Printf("Expectation: %s\n", step.Expectation)
	}
}
```

### With Tool Integration

```go
// Define available tools
tools := []planning.ToolInfo{
	{
		Name:        "web_search",
		Description: "Search the web for information",
		UsageNotes:  "Use for finding current information and best practices",
	},
	{
		Name:        "code_analysis",
		Description: "Analyze existing code structure",
		UsageNotes:  "Use to understand existing codebase before making changes",
	},
}

input := planning.PlannerInput{
	Objective: "Refactor the authentication system to use JWT tokens",
	Tools:     tools,
	Context:   "Current system uses session-based authentication",
}

result, err := planner.Plan(context.Background(), input)
if err != nil {
	log.Fatal(err)
}

// The plan will now include steps that reference these tools
```

### Replanning with Feedback

```go
// Create an initial plan
initialResult, err := planner.Plan(context.Background(), planning.PlannerInput{
	Objective: "Implement user notification system",
})
if err != nil {
	log.Fatal(err)
}

// Execute some steps and gather feedback
feedback := "The initial plan doesn't account for email notifications and push notifications should be prioritized"

// Create a revised plan
revisedResult, err := planner.Replan(
	context.Background(),
	initialResult.Plan,
	feedback,
	planning.PlannerInput{
		Objective: "Implement user notification system with email and prioritized push notifications",
	},
)
if err != nil {
	log.Fatal(err)
}

// The revised plan includes a diff showing what changed
if revisedResult.Plan.RevisionDiff != nil {
	fmt.Println("Plan Changes:")
	fmt.Printf("Added Steps: %d\n", len(revisedResult.Plan.RevisionDiff.Steps.Added))
	fmt.Printf("Modified Steps: %d\n", len(revisedResult.Plan.RevisionDiff.Steps.Changed))
	fmt.Printf("Removed Steps: %d\n", len(revisedResult.Plan.RevisionDiff.Steps.Removed))
}
```

### Using with Agent Interface

```go
// The planner implements the standard Agent interface
var myAgent agent.Agent = planner

// Execute using standard agent parameters
params := agent.AgentParameters{
	Input: "Create a deployment plan for our microservices",
	AdditionalInputs: map[string]interface{}{
		"context": "We have 5 services that need to be deployed in order",
		"tools": []planning.ToolInfo{
			{
				Name:        "kubernetes_deploy",
				Description: "Deploy to Kubernetes cluster",
			},
		},
	},
}

result, err := myAgent.Execute(context.Background(), params)
if err != nil {
	log.Fatal(err)
}

// Extract the plan from additional outputs
generatedPlan := result.AdditionalOutputs["plan"].(*plan.Plan)
```

### Working with Plans

```go
// Validate a plan
if err := generatedPlan.Validate(); err != nil {
	log.Printf("Plan validation failed: %v", err)
}

// Get execution order (topologically sorted)
executionOrder, err := generatedPlan.GetExecutionOrder()
if err != nil {
	log.Fatal(err)
}

for i, step := range executionOrder {
	fmt.Printf("%d. %s\n", i+1, step.Name)
}

// Find specific steps
step, found := generatedPlan.FindStep("step_id")
if found {
	fmt.Printf("Found step: %s\n", step.Name)
}

// Get all steps including nested ones
allSteps := generatedPlan.GetAllStepsRecursive()
fmt.Printf("Total steps (including nested): %d\n", len(allSteps))

// Check plan depth
depth := generatedPlan.GetMaxDepth()
fmt.Printf("Plan nesting depth: %d\n", depth)

// Serialize to JSON
jsonData, err := json.Marshal(generatedPlan)
if err != nil {
	log.Fatal(err)
}

// Deserialize from JSON
var loadedPlan plan.Plan
if err := json.Unmarshal(jsonData, &loadedPlan); err != nil {
	log.Fatal(err)
}
```

## Plan Structure

A plan consists of:

- **ID**: Unique identifier for the plan
- **Steps**: Array of steps to execute
- **CreatedAt**: Timestamp of plan creation
- **RevisionDiff**: Optional diff from a previous plan version

Each step contains:

- **ID**: Unique identifier within the plan
- **Name**: Short, descriptive name
- **Instruction**: Detailed instructions for execution
- **Expectation**: What success looks like for this step
- **Dependencies**: Array of step IDs that must complete first
- **Plan**: Optional nested sub-plan for complex steps

## Prompt Template

The planner uses an embedded Go template located at `pkg/assets/prompts/planner.prompt`. The template is automatically loaded when you create a planner agent - no file paths needed!

The template supports:

- **objective**: The high-level goal to plan for
- **tools**: Optional array of available tools
- **context**: Additional context for planning

The template is embedded at compile time using Go's `embed` directive. You can:

1. Use the default embedded template (automatic with `NewPlannerAgent()`)
2. Load a custom template file: `planner.LoadPromptTemplate("/path/to/custom.prompt")`
3. Set a template programmatically: `planner.SetPromptTemplate(customTemplate)`

### Custom Prompt Templates

If you need to use a custom prompt template:

```go
// Load from a file
if err := planner.LoadPromptTemplate("/path/to/custom.prompt"); err != nil {
    log.Fatal(err)
}

// Or create and set programmatically
tmpl, _ := prompt.AddTemplate("custom", "{{.objective}}")
planner.SetPromptTemplate(tmpl)
```

## Plan Validation

Plans are automatically validated for:

- **Unique step IDs**: No duplicate IDs within a plan
- **Valid dependencies**: All dependency IDs reference existing steps
- **No self-dependencies**: Steps cannot depend on themselves
- **Acyclic graph**: Dependencies form a DAG with no cycles
- **Nested plan validity**: Sub-plans are recursively validated

## Advanced Features

### Execution Tracking

```go
// Track which steps are completed
completedSteps := map[string]bool{
	"step1": true,
	"step2": true,
}

// Get next ready steps
readySteps := generatedPlan.NextSteps(completedSteps)
for _, step := range readySteps {
	fmt.Printf("Ready to execute: %s\n", step.Name)
}
```

### Plan Comparison

```go
// Compare two plan versions
diff := plan.NewPlanDiff("diff-id", oldPlan, newPlan, "Updated based on feedback")

// Examine what changed
for stepID, addedStep := range diff.Steps.Added {
	fmt.Printf("Added: %s - %s\n", stepID, addedStep.Name)
}

for stepID, change := range diff.Steps.Changed {
	fmt.Printf("Changed: %s\n", stepID)
	fmt.Printf("  From: %s\n", change.From.Name)
	fmt.Printf("  To: %s\n", change.To.Name)
}

for stepID, removedStep := range diff.Steps.Removed {
	fmt.Printf("Removed: %s - %s\n", stepID, removedStep.Name)
}
```

## Best Practices

1. **Clear Objectives**: Provide specific, well-defined objectives for better plans
2. **Contextual Information**: Include relevant context to guide the planner
3. **Tool Awareness**: List available tools so they can be incorporated into steps
4. **Iterative Refinement**: Use replanning to refine plans based on execution feedback
5. **Validation**: Always validate plans before execution
6. **Error Handling**: Handle validation errors and provide feedback for replanning

## Configuration

### Temperature
- **0.0-0.3**: More deterministic, structured plans
- **0.4-0.7**: Balanced creativity and structure (recommended)
- **0.8-1.0**: More creative, varied approaches

### Max Tokens
- Recommended: 2000-4000 tokens for detailed plans
- Minimum: 1000 tokens for simple plans
- Maximum: 8000+ tokens for complex, nested plans

## Error Handling

```go
result, err := planner.Plan(ctx, input)
if err != nil {
	// Check for specific error types
	if strings.Contains(err.Error(), "validation") {
		// Plan validation failed - likely cyclic dependencies
		log.Printf("Plan validation failed: %v", err)
	} else if strings.Contains(err.Error(), "parse") {
		// Failed to parse LLM response
		log.Printf("Response parsing failed: %v", err)
		log.Printf("Raw response: %s", result.RawResponse)
	} else {
		// Other errors (model failure, context timeout, etc.)
		log.Printf("Planning failed: %v", err)
	}
}
```

## Integration with Other Agents

The planner can be used with other agents in an agentic system:

```go
// 1. Use planner to create a plan
planResult, _ := planner.Plan(ctx, plannerInput)

// 2. Use executor agent to execute steps
for _, step := range planResult.Plan.Steps {
	executorParams := agent.AgentParameters{
		Input: step.Instruction,
		AdditionalInputs: map[string]interface{}{
			"expectation": step.Expectation,
		},
	}
	executorResult, _ := executorAgent.Execute(ctx, executorParams)
	
	// 3. Use judge agent to validate results
	judgeParams := agent.AgentParameters{
		Input: executorResult.Output,
		AdditionalInputs: map[string]interface{}{
			"expectation": step.Expectation,
		},
	}
	judgeResult, _ := judgeAgent.Execute(ctx, judgeParams)
	
	// Handle validation results...
}
```

## License

This package is part of the gogentic project.

