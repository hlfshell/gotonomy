// Package planning provides planning agent implementations.
package planning

import (
	"context"

	"github.com/hlfshell/gogentic/pkg/agent"
)

// PlanApprovalFunc is a function that approves or rejects a plan.
// It returns a boolean indicating whether the plan is approved,
// and an optional string with recommended changes.
type PlanApprovalFunc func(ctx context.Context, plan string, history []agent.Message) (bool, string, error)

// PlanningAgentConfig extends AgentConfig with planning-specific settings.
type PlanningAgentConfig struct {
	// Base agent configuration
	AgentConfig agent.AgentConfig
	
	// Sub-agents for different stages of the planning process
	Planner          Planner       // Creates plans and final answers (if nil, creates GenericPlanner)
	Judge            agent.Agent   // Judges/evaluates step execution (any agent that returns a Judgement) (if nil, creates default judge)
	StepExecutionAgent agent.Agent // Executes individual steps
	
	// PlanApprovalFunc is an optional function to approve plans
	PlanApprovalFunc PlanApprovalFunc
	
	// Maximum attempts for planning and execution
	MaxPlanAttempts int
	MaxStepAttempts int
	
	// Optional prior context
	PriorPlan    string
	PriorHistory []agent.Message
}

