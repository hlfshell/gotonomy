// Package planning provides planning agent implementations.
package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	arkaineparser "github.com/hlfshell/go-arkaine-parser"

	"github.com/hlfshell/gogentic/pkg/agent"
)

// JudgementStatus represents the status of a judgement.
type JudgementStatus string

const (
	JudgementStatusContinue JudgementStatus = "continue"
	JudgementStatusComplete JudgementStatus = "complete"
	JudgementStatusError    JudgementStatus = "error"
	JudgementStatusAdjust   JudgementStatus = "adjust"
	JudgementStatusReplan   JudgementStatus = "replan"
)

// Evidence represents evidence of a deviation from expected behavior.
type Evidence struct {
	Observed  string `json:"observed"`  // What was observed happening
	Expected  string `json:"expected"`  // What was expected as output
	Deviation string `json:"deviation"` // Explanation of the deviation
}

// Judgement represents the result of evaluating a step execution.
type Judgement struct {
	Reason         string          `json:"reason"`         // Why we are making the decision we made
	Evidence       []Evidence      `json:"evidence"`       // Evidence of mistakes or deviations
	Status         JudgementStatus `json:"status"`         // Status: continue, complete, error, adjust, replan
	Recommendation string          `json:"recommendation"` // Recommendation to the agent (only used in replan or adjust)
}

// Judge is any agent.Agent that returns a response parseable into a Judgement.
// Any agent can be used as a judge - it just needs to return a Judgement structure.

// evaluateStep evaluates the execution of a step using the judge agent.
func (a *PlanningAgent) evaluateStep(ctx context.Context, plan string, step string, stepNum int, stepResult string, options agent.AgentOptions) (*Judgement, agent.Message, error) {
	// Create a conversation for the judge agent
	judgeConversation := &agent.Conversation{
		ID:        uuid.New().String(),
		Messages:  []agent.Message{},
		Metadata:  map[string]interface{}{"plan": plan, "step": step, "step_num": stepNum, "step_result": stepResult},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create the evaluation input
	evaluationInput := fmt.Sprintf("Evaluate the execution of the following step in the plan:\n\nStep %d: %s\n\nExecution Result:\n%s", stepNum, step, stepResult)

	// Create a user message for the judge agent
	judgeUserMessage := agent.Message{
		Role:      "user",
		Content:   evaluationInput,
		Timestamp: time.Now(),
	}

	// Add the message to the conversation
	judgeConversation.Messages = append(judgeConversation.Messages, judgeUserMessage)
	judgeConversation.UpdatedAt = time.Now()

	// Execute the judge agent
	judgeResult, err := a.config.Judge.Execute(ctx, agent.AgentParameters{
		Input:        evaluationInput,
		Conversation: judgeConversation,
		Options:      options,
	})
	if err != nil {
		return nil, agent.Message{}, fmt.Errorf("failed to evaluate step: %w", err)
	}

	// Parse the judgement from the result
	judgement, err := parseJudgement(judgeResult)
	if err != nil {
		return nil, judgeResult.Message, fmt.Errorf("failed to parse judgement: %w", err)
	}

	return judgement, judgeResult.Message, nil
}

// parseJudgement parses a Judgement from an AgentResult.
// It first tries to parse from ParsedOutput, then falls back to parsing the Output as JSON.
func parseJudgement(result agent.AgentResult) (*Judgement, error) {
	var judgement Judgement

	// Try to parse from ParsedOutput first
	if len(result.ParsedOutput) > 0 {
		// Convert ParsedOutput to JSON and then unmarshal
		jsonBytes, err := json.Marshal(result.ParsedOutput)
		if err == nil {
			if err := json.Unmarshal(jsonBytes, &judgement); err == nil {
				// Validate status
				if err := validateJudgementStatus(judgement.Status); err != nil {
					return nil, err
				}
				return &judgement, nil
			}
		}
	}

	// Fall back to parsing Output as JSON
	if result.Output != "" {
		if err := json.Unmarshal([]byte(result.Output), &judgement); err == nil {
			// Validate status
			if err := validateJudgementStatus(judgement.Status); err != nil {
				return nil, err
			}
			return &judgement, nil
		}
	}

	return nil, errors.New("unable to parse judgement from agent result")
}

// validateJudgementStatus validates that the status is one of the allowed values.
func validateJudgementStatus(status JudgementStatus) error {
	switch status {
	case JudgementStatusContinue, JudgementStatusComplete, JudgementStatusError, JudgementStatusAdjust, JudgementStatusReplan:
		return nil
	default:
		return fmt.Errorf("invalid judgement status: %s", status)
	}
}

// createJudgeAgent creates a judge agent with the given configuration.
func createJudgeAgent(config PlanningAgentConfig) agent.Agent {
	if config.Judge != nil {
		return config.Judge
	}

	// Parser labels for the judgement structure
	judgeLabels := []arkaineparser.Label{
		{Name: "reason", Required: true},
		{Name: "evidence", IsJSON: true, Required: true},
		{Name: "status", Required: true},
		{Name: "recommendation"},
	}

	judgeAgent := agent.NewBaseAgent(
		uuid.New().String(),
		"Judge Agent",
		"Agent for evaluating plan execution",
		agent.AgentConfig{
			Model:        config.AgentConfig.Model,
			SystemPrompt: "You are a judge agent that evaluates the execution of plan steps. You must return a JSON object with the following structure:\n{\n  \"reason\": \"why we are making the decision we made\",\n  \"evidence\": [\n    {\n      \"observed\": \"what we saw happening we are presenting as evidence of a mistake\",\n      \"expected\": \"What we would have expected as the output\",\n      \"deviation\": \"explanation of the deviation\"\n    }\n  ],\n  \"status\": \"continue\" | \"complete\" | \"error\" | \"adjust\" | \"replan\",\n  \"recommendation\": \"a string recommending to the agent what to fix or try. Only used in replan or adjust\"\n}",
			MaxTokens:    config.AgentConfig.MaxTokens,
			Temperature:  config.AgentConfig.Temperature,
			Timeout:      config.AgentConfig.Timeout,
			ParserLabels: judgeLabels,
		},
	)

	return judgeAgent
}
