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
	"github.com/hlfshell/gogentic/pkg/agent/tool"
)

// executeStep executes a single step using the execution agent.
func (a *PlanningAgent) executeStep(ctx context.Context, plan string, step string, stepNum int, options agent.AgentOptions) (string, agent.Message, error) {
	// Get ExecutionContext if available
	execCtx, hasExecCtx := agent.AsExecutionContext(ctx)

	// Create step node if ExecutionContext is available
	if hasExecCtx {
		stepNode, err := execCtx.CreateChildNode(nil, "step", fmt.Sprintf("step_%d", stepNum), map[string]interface{}{
			"step_number": stepNum,
			"step":        step,
			"plan":        plan,
		})
		if err == nil {
			_ = stepNode
		}
	}

	// Create a conversation for the execution agent
	executionConversation := &agent.Conversation{
		ID:        uuid.New().String(),
		Messages:  []agent.Message{},
		Metadata:  map[string]interface{}{"plan": plan, "step": step, "step_num": stepNum},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create the execution input
	executionInput := fmt.Sprintf("Execute the following step in the plan:\n\nStep %d: %s", stepNum, step)

	// Create a user message for the execution agent
	executionUserMessage := agent.Message{
		Role:      "user",
		Content:   executionInput,
		Timestamp: time.Now(),
	}

	// Add the message to the conversation
	executionConversation.Messages = append(executionConversation.Messages, executionUserMessage)
	executionConversation.UpdatedAt = time.Now()

	// Execute the execution agent
	executionResult, err := a.config.StepExecutionAgent.Execute(ctx, agent.AgentParameters{
		Input:        executionInput,
		Conversation: executionConversation,
		Options:      options,
	})
	if err != nil {
		if hasExecCtx {
			execCtx.SetError(err)
		}
		return "", agent.Message{}, fmt.Errorf("failed to execute step: %w", err)
	}

	// Set step output in execution context
	if hasExecCtx {
		agent.SetOutput(execCtx, executionResult.Output)
		agent.SetData(execCtx, "step_result", executionResult.Output)
	}

	return executionResult.Output, executionResult.Message, nil
}

// executeSteps executes each step in the plan.
func (a *PlanningAgent) executeSteps(ctx context.Context, plan string, steps []interface{}, conversation *agent.Conversation, params agent.AgentParameters) (string, error) {
	// Initialize the final result
	finalResult := ""

	// Execute each step
	for i, stepInterface := range steps {
		// Check for timeout
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("execution timed out during step execution: %w", ctx.Err())
		default:
		}

		step := stepInterface.(string)
		stepNum := i + 1

		// Execute the step
		stepResult := ""
		stepAttempts := 0
		stepSuccess := false

		for stepAttempts < a.config.MaxStepAttempts && !stepSuccess {
			// Execute step using execution agent
			var stepResultMessage agent.Message
			var err error
			stepResult, stepResultMessage, err = a.executeStep(ctx, plan, step, stepNum, params.Options)
			if err != nil {
				return "", err
			}

			// Add step result to conversation
			conversation.Messages = append(conversation.Messages, stepResultMessage)
			conversation.UpdatedAt = time.Now()

			// Evaluate step execution
			judgement, evaluationMessage, err := a.evaluateStep(ctx, plan, step, stepNum, stepResult, params.Options)
			if err != nil {
				return "", err
			}

			// Add evaluation to conversation
			conversation.Messages = append(conversation.Messages, evaluationMessage)
			conversation.UpdatedAt = time.Now()

			// Check judgement result
			switch judgement.Status {
			case JudgementStatusContinue, JudgementStatusComplete:
				// Step was successful, continue
				stepSuccess = true
			case JudgementStatusReplan:
				// Need to recalculate plan
				recalculateMessage := agent.Message{
					Role:      "user",
					Content:   fmt.Sprintf("The current plan is not working. Please create a new plan. Consider this feedback: %s", judgement.Recommendation),
					Timestamp: time.Now(),
				}
				conversation.Messages = append(conversation.Messages, recalculateMessage)
				conversation.UpdatedAt = time.Now()

				// Return error to trigger plan recalculation
				return "", errors.New("plan needs to be recalculated")
			case JudgementStatusAdjust:
				// Retry step with feedback
				retryMessage := agent.Message{
					Role:      "user",
					Content:   fmt.Sprintf("Please try again with this step. Consider this feedback: %s", judgement.Recommendation),
					Timestamp: time.Now(),
				}
				conversation.Messages = append(conversation.Messages, retryMessage)
				conversation.UpdatedAt = time.Now()
			case JudgementStatusError:
				// Error occurred, return error
				return "", fmt.Errorf("judgement error: %s", judgement.Reason)
			default:
				// Unknown status, treat as error
				return "", fmt.Errorf("unknown judgement status: %s", judgement.Status)
			}

			stepAttempts++
		}

		// Check if step was successful
		if !stepSuccess {
			return "", fmt.Errorf("failed to execute step %d after maximum attempts", stepNum)
		}

		// Add step result to final result
		finalResult += fmt.Sprintf("Step %d: %s\n\nResult: %s\n\n", stepNum, step, stepResult)
	}

	return finalResult, nil
}

// createExecutorAgent creates an executor agent with the given configuration.
func createExecutorAgent(config PlanningAgentConfig) agent.Agent {
	if config.StepExecutionAgent != nil {
		return config.StepExecutionAgent
	}

	executionLabels := []arkaineparser.Label{
		{Name: "Reasoning", IsBlockStart: true},
		{Name: "Action", Required: true},
		{Name: "ActionInput", IsJSON: true},
		{Name: "Result"},
	}

	agentConfig := agent.AgentConfig{
		Model:        config.AgentConfig.Model,
		SystemPrompt: "You are an agent that executes specific steps in a plan.",
		MaxTokens:    config.AgentConfig.MaxTokens,
		Temperature:  config.AgentConfig.Temperature,
		Timeout:      config.AgentConfig.Timeout,
		ParserLabels: executionLabels,
	}

	if len(config.AgentConfig.Tools) > 0 {
		agentConfig.Tools = config.AgentConfig.Tools
		agentConfig.MaxIterations = config.AgentConfig.MaxIterations
		return tool.NewToolAgent(
			uuid.New().String(),
			"Step Execution Agent",
			"Agent for executing plan steps",
			agentConfig,
		)
	}

	return agent.NewBaseAgent(
		uuid.New().String(),
		"Step Execution Agent",
		"Agent for executing plan steps",
		agentConfig,
	)
}
