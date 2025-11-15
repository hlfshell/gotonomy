// Package planning provides planning agent implementations.
package planning

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	arkaineparser "github.com/hlfshell/go-arkaine-parser"

	"github.com/hlfshell/gogentic/pkg/agent"
)

// Planner is an interface for generating plans and final answers.
type Planner interface {
	// GeneratePlan generates a plan from the given input.
	// Returns: plan (string), steps ([]interface{}), message (Message), error
	GeneratePlan(ctx context.Context, input string, conversation *agent.Conversation, options agent.AgentOptions) (string, []interface{}, agent.Message, error)

	// GenerateFinalAnswer generates a final answer based on execution results.
	GenerateFinalAnswer(ctx context.Context, input string, finalResult string, options agent.AgentOptions) (agent.AgentResult, error)
}

// GenericPlanner is a generic implementation of Planner that uses an agent.Agent.
type GenericPlanner struct {
	agent agent.Agent
}

// NewGenericPlanner creates a new generic planner using the provided agent.
func NewGenericPlanner(agent agent.Agent) *GenericPlanner {
	return &GenericPlanner{
		agent: agent,
	}
}

// GeneratePlan implements the Planner interface.
func (p *GenericPlanner) GeneratePlan(ctx context.Context, input string, conversation *agent.Conversation, options agent.AgentOptions) (string, []interface{}, agent.Message, error) {
	// Create a conversation for the planner agent
	planConversation := &agent.Conversation{
		ID:        uuid.New().String(),
		Messages:  []agent.Message{},
		Metadata:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a user message for the planner agent
	planUserMessage := agent.Message{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}

	// Add the message to the conversation
	planConversation.Messages = append(planConversation.Messages, planUserMessage)
	planConversation.UpdatedAt = time.Now()

	// Execute the planner agent
	planResult, err := p.agent.Execute(ctx, agent.AgentParameters{
		Input:        input,
		Conversation: planConversation,
		Options:      options,
	})
	if err != nil {
		return "", nil, agent.Message{}, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Get the plan from the result
	plan := planResult.Output

	// Parse the plan to extract steps
	parsedOutput := planResult.ParsedOutput
	stepsValue, ok := parsedOutput["Step"]
	if !ok {
		return "", nil, agent.Message{}, errors.New("steps not found in plan")
	}
	steps := stepsValue.([]interface{})

	return plan, steps, planResult.Message, nil
}

// GenerateFinalAnswer implements the Planner interface.
func (p *GenericPlanner) GenerateFinalAnswer(ctx context.Context, input string, finalResult string, options agent.AgentOptions) (agent.AgentResult, error) {
	// Create a conversation for the final answer generation
	finalConversation := &agent.Conversation{
		ID:        uuid.New().String(),
		Messages:  []agent.Message{},
		Metadata:  map[string]interface{}{"final_result": finalResult},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create the final answer input
	finalInput := fmt.Sprintf("You have completed all steps of your plan. Based on the results of each step, provide a final answer to the original question or task.\n\nOriginal Task: %s\n\nResults:\n%s", input, finalResult)

	// Create a user message for the final answer
	finalUserMessage := agent.Message{
		Role:      "user",
		Content:   finalInput,
		Timestamp: time.Now(),
	}

	// Add the message to the final conversation
	finalConversation.Messages = append(finalConversation.Messages, finalUserMessage)
	finalConversation.UpdatedAt = time.Now()

	// Execute the planner agent for the final answer
	return p.agent.Execute(ctx, agent.AgentParameters{
		Input:        finalInput,
		Conversation: finalConversation,
		Options:      options,
	})
}

// createPlannerAgent creates a planner agent with the given configuration.
func createPlannerAgent(config PlanningAgentConfig) Planner {
	if config.Planner != nil {
		return config.Planner
	}

	planningLabels := []arkaineparser.Label{
		{Name: "Plan", IsBlockStart: true, Required: true},
		{Name: "Step", IsBlockStart: true, Required: true},
		{Name: "Reasoning", IsBlockStart: true},
	}

	plannerAgent := agent.NewBaseAgent(
		uuid.New().String(),
		"Plan Generation Agent",
		"Agent for generating plans",
		agent.AgentConfig{
			Model:        config.AgentConfig.Model,
			SystemPrompt: "You are an agent that creates detailed, step-by-step plans to accomplish tasks.",
			MaxTokens:    config.AgentConfig.MaxTokens,
			Temperature:  config.AgentConfig.Temperature,
			Timeout:      config.AgentConfig.Timeout,
			ParserLabels: planningLabels,
		},
	)

	return NewGenericPlanner(plannerAgent)
}

// generatePlan uses the planner to create a plan.
func (a *PlanningAgent) generatePlan(ctx context.Context, input string, conversation *agent.Conversation, options agent.AgentOptions) (string, []interface{}, agent.Message, error) {
	return a.config.Planner.GeneratePlan(ctx, input, conversation, options)
}

// generateOrUsePriorPlan generates a new plan or uses a prior plan if provided.
func (a *PlanningAgent) generateOrUsePriorPlan(ctx context.Context, params agent.AgentParameters, conversation *agent.Conversation, planApprovalFunc PlanApprovalFunc) (string, []interface{}, agent.Message, error) {
	// Check if prior plan is provided
	if a.config.PriorPlan != "" {
		// Use the provided prior plan
		plan := a.config.PriorPlan
		planMessage := agent.Message{
			Role:      "assistant",
			Content:   a.config.PriorPlan,
			Timestamp: time.Now(),
		}

		// Add the plan message to the conversation
		conversation.Messages = append(conversation.Messages, planMessage)
		conversation.UpdatedAt = time.Now()

		// Parse the plan to extract steps
		parsedOutput, _ := a.GetParser().Parse(planMessage.Content)
		stepsValue, ok := parsedOutput["Step"]
		if !ok {
			return "", nil, agent.Message{}, errors.New("steps not found in prior plan")
		}
		steps := stepsValue.([]interface{})

		return plan, steps, planMessage, nil
	}

	// Generate a new plan
	var plan string
	var steps []interface{}
	var planMessage agent.Message
	planApproved := false
	planAttempts := 0

	for planAttempts < a.config.MaxPlanAttempts && !planApproved {
		// Check for timeout
		select {
		case <-ctx.Done():
			return "", nil, agent.Message{}, fmt.Errorf("execution timed out during planning: %w", ctx.Err())
		default:
		}

		// Create planning input
		planningInput := params.Input
		if planAttempts > 0 {
			// Add feedback from previous attempt
			planningInput = fmt.Sprintf("Your previous plan was not approved. Please revise it based on the feedback.\n\nOriginal task: %s", params.Input)
		}

		// Generate plan using the planner
		var err error
		plan, steps, planMessage, err = a.generatePlan(ctx, planningInput, conversation, params.Options)
		if err != nil {
			return "", nil, agent.Message{}, err
		}

		// Add plan to conversation
		conversation.Messages = append(conversation.Messages, planMessage)
		conversation.UpdatedAt = time.Now()

		// Get approval for the plan
		var feedback string
		planApproved, feedback, err = planApprovalFunc(ctx, planMessage.Content, conversation.Messages)
		if err != nil {
			return "", nil, agent.Message{}, fmt.Errorf("failed to get plan approval: %w", err)
		}

		// If not approved, add feedback to conversation
		if !planApproved {
			feedbackMessage := agent.Message{
				Role:      "user",
				Content:   feedback,
				Timestamp: time.Now(),
			}
			conversation.Messages = append(conversation.Messages, feedbackMessage)
			conversation.UpdatedAt = time.Now()
		}

		planAttempts++
	}

	// Check if plan was approved
	if !planApproved {
		return "", nil, agent.Message{}, errors.New("failed to create an approved plan after maximum attempts")
	}

	return plan, steps, planMessage, nil
}

// generateFinalAnswer creates a final answer based on the execution results.
func (a *PlanningAgent) generateFinalAnswer(ctx context.Context, input string, finalResult string, options agent.AgentOptions) (agent.AgentResult, error) {
	return a.config.Planner.GenerateFinalAnswer(ctx, input, finalResult, options)
}
