// Package planning provides planning agent implementations.
package planning

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	arkaineparser "github.com/hlfshell/go-arkaine-parser"
	"github.com/hlfshell/gogentic/pkg/model"

	"github.com/hlfshell/gogentic/pkg/agent"
)

// PlanningAgent is an implementation of the Agent interface that creates and
// executes plans to accomplish tasks.
type PlanningAgent struct {
	*agent.BaseAgent
	config PlanningAgentConfig
}

// initConversation initializes a conversation with the user's message.
func initConversation(params agent.AgentParameters) *agent.Conversation {
	// Use provided conversation or create a new one
	var conversation *agent.Conversation
	if params.Conversation != nil {
		conversation = params.Conversation
	} else {
		conversation = &agent.Conversation{
			ID:        uuid.New().String(),
			Messages:  []agent.Message{},
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Add user message to conversation
	userMessage := agent.Message{
		Role:      "user",
		Content:   params.Input,
		Timestamp: time.Now(),
	}
	conversation.Messages = append(conversation.Messages, userMessage)
	conversation.UpdatedAt = time.Now()

	return conversation
}

// defaultPlanApprovalFunc is the default implementation of PlanApprovalFunc.
func defaultPlanApprovalFunc(ctx context.Context, plan string, history []agent.Message) (bool, string, error) {
	// By default, all plans are approved
	return true, "", nil
}

// NewPlanningAgent creates a new planning agent with the given configuration.
func NewPlanningAgent(id, name, description string, config PlanningAgentConfig) *PlanningAgent {
	// Set default values
	if config.MaxPlanAttempts <= 0 {
		config.MaxPlanAttempts = 3
	}
	if config.MaxStepAttempts <= 0 {
		config.MaxStepAttempts = 2
	}
	if config.AgentConfig.Timeout <= 0 {
		config.AgentConfig.Timeout = 120 * time.Second
	}

	// Default parser labels
	planningLabels := []arkaineparser.Label{
		{Name: "Plan", IsBlockStart: true, Required: true},
		{Name: "Step", IsBlockStart: true, Required: true},
		{Name: "Reasoning", IsBlockStart: true},
	}

	executionLabels := []arkaineparser.Label{
		{Name: "Reasoning", IsBlockStart: true},
		{Name: "Action", Required: true},
		{Name: "ActionInput", IsJSON: true},
		{Name: "Result"},
	}

	evaluationLabels := []arkaineparser.Label{
		{Name: "Evaluation", IsBlockStart: true, Required: true},
		{Name: "Status", Required: true},
		{Name: "Feedback"},
		{Name: "RecalculatePlan", Required: true},
	}

	if len(config.AgentConfig.ParserLabels) == 0 {
		config.AgentConfig.ParserLabels = append(planningLabels, executionLabels...)
		config.AgentConfig.ParserLabels = append(config.AgentConfig.ParserLabels, evaluationLabels...)
		config.AgentConfig.ParserLabels = append(config.AgentConfig.ParserLabels, arkaineparser.Label{Name: "FinalAnswer", IsBlockStart: true})
	}

	// Create the base agent
	baseAgent := agent.NewBaseAgent(id, name, description, config.AgentConfig)

	// Create sub-agents if not provided
	if config.Planner == nil {
		config.Planner = createPlannerAgent(config)
	}
	if config.Judge == nil {
		config.Judge = createJudgeAgent(config)
	}
	config.StepExecutionAgent = createExecutorAgent(config)

	return &PlanningAgent{
		BaseAgent: baseAgent,
		config:    config,
	}
}

// Execute processes the given parameters and returns a result.
func (a *PlanningAgent) Execute(ctx context.Context, params agent.AgentParameters) (agent.AgentResult, error) {
	// Get or create ExecutionContext
	execCtx := agent.InitContext(ctx)
	ctx = execCtx // Use ExecutionContext as the context going forward

	// Create agent execution node and set as current
	agentNode, err := execCtx.CreateChildNode(nil, "agent", a.BaseAgent.Name(), map[string]interface{}{
		"input":      params.Input,
		"agent_id":   a.BaseAgent.ID(),
		"agent_type": "planning",
	})
	if err != nil {
		return agent.AgentResult{}, fmt.Errorf("failed to create agent node: %w", err)
	}
	if err := execCtx.SetCurrentNode(agentNode); err != nil {
		return agent.AgentResult{}, fmt.Errorf("failed to set current node: %w", err)
	}
	_ = agentNode

	// Set execution-level data
	agent.SetExecutionData(execCtx, "agent_id", a.BaseAgent.ID())
	agent.SetExecutionData(execCtx, "agent_name", a.BaseAgent.Name())
	agent.SetExecutionData(execCtx, "agent_type", "planning")

	// Setup execution environment
	startTime := time.Now()
	timeout := a.config.AgentConfig.Timeout
	if params.Options.Timeout != nil {
		timeout = *params.Options.Timeout
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Initialize conversation, stats, and approval function
	conversation := initConversation(params)
	usageStats := model.UsageStats{}
	executionStats := agent.ExecutionStats{StartTime: startTime}
	planApprovalFunc := a.config.PlanApprovalFunc
	if planApprovalFunc == nil {
		planApprovalFunc = defaultPlanApprovalFunc
	}

	// PHASE 1: PLAN GENERATION
	// Create plan generation node
	planNode, err := execCtx.CreateChildNode(nil, "phase", "plan_generation", map[string]interface{}{
		"input": params.Input,
	})
	if err == nil {
		_ = planNode
	}

	// Generate or use prior plan
	plan, steps, _, err := a.generateOrUsePriorPlan(timeoutCtx, params, conversation, planApprovalFunc)
	if err != nil {
		execCtx.SetError(err)
		return agent.AgentResult{}, err
	}

	// Store plan in execution context
	agent.SetData(execCtx, "plan", plan)
	agent.SetData(execCtx, "steps", steps)

	// PHASE 2: STEP EXECUTION
	// Create step execution node
	execNode, err := execCtx.CreateChildNode(nil, "phase", "step_execution", map[string]interface{}{
		"plan":  plan,
		"steps": steps,
	})
	if err == nil {
		_ = execNode
	}

	// Execute each step in the plan
	finalResult, err := a.executeSteps(timeoutCtx, plan, steps, conversation, params)
	if err != nil {
		execCtx.SetError(err)
		return agent.AgentResult{}, err
	}

	agent.SetData(execCtx, "final_result", finalResult)

	// PHASE 3: FINAL ANSWER
	// Create final answer node
	finalNode, err := execCtx.CreateChildNode(nil, "phase", "final_answer", map[string]interface{}{
		"final_result": finalResult,
	})
	if err == nil {
		_ = finalNode
	}

	// Generate final answer
	finalAnswer, err := a.generateFinalAnswer(timeoutCtx, params.Input, finalResult, params.Options)
	if err != nil {
		execCtx.SetError(err)
		return agent.AgentResult{}, fmt.Errorf("failed to generate final answer: %w", err)
	}

	// Update stats
	usageStats.PromptTokens += finalAnswer.UsageStats.PromptTokens
	usageStats.CompletionTokens += finalAnswer.UsageStats.CompletionTokens
	usageStats.TotalTokens += finalAnswer.UsageStats.TotalTokens
	executionStats.ToolCalls += finalAnswer.ExecutionStats.ToolCalls
	executionStats.Iterations++

	// Add final answer to conversation
	conversation.Messages = append(conversation.Messages, finalAnswer.Message)
	conversation.UpdatedAt = time.Now()

	// Record execution end time
	executionStats.EndTime = time.Now()

	// Set output in execution context
	agent.SetOutput(execCtx, finalAnswer.Output)
	agent.SetData(execCtx, "usage_stats", usageStats)
	agent.SetData(execCtx, "execution_stats", executionStats)

	// Return the result
	return agent.AgentResult{
		Output: finalAnswer.Output,
		AdditionalOutputs: map[string]interface{}{
			"plan":         plan,
			"steps":        steps,
			"step_results": finalResult,
		},
		Conversation:   conversation,
		UsageStats:     usageStats,
		ExecutionStats: executionStats,
		Message:        finalAnswer.Message,
		ParsedOutput:   finalAnswer.ParsedOutput,
		ParseErrors:    finalAnswer.ParseErrors,
	}, nil
}
