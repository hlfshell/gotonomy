package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

var DefaultAgentConfig = AgentConfig{
	MaxSteps:            0,
	Timeout:             5 * time.Minute,
	MaxToolWorkers:      5,
	ToolErrorHandling:   PassErrorsToModel,
	OnToolErrorFunction: nil,
}

// DefaultArgumentsToPrompt marshals the entire arguments map to a single JSON string
// under the "input" key. This produces nested JSON when later embedded in prompts.
// If you want per-field templating, provide a custom ArgumentsToMessagesFunc.
func DefaultArgumentsToPrompt(args tool.Arguments) (map[string]string, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}
	return map[string]string{
		"input": string(data),
	}, nil
}

// DefaultArgumentsToMessages builds a simple single-turn conversation:
//   - If the session has prior steps, it replays the full conversation history.
//   - For the first iteration, it converts args into a single user message whose
//     content is the JSON-encoded "input" field from DefaultArgumentsToPrompt.
func DefaultArgumentsToMessages(args tool.Arguments, sess *Session) ([]model.Message, error) {
	if sess != nil && len(sess.Steps()) > 0 {
		return sess.Conversation(), nil
	}

	// No prior steps - start a new conversation from arguments.
	inputMap, err := DefaultArgumentsToPrompt(args)
	if err != nil {
		return nil, fmt.Errorf("building prompt from args: %w", err)
	}
	input, ok := inputMap["input"]
	if !ok {
		return nil, fmt.Errorf("default prompt missing input field")
	}

	return []model.Message{
		{
			Role:    model.RoleUser,
			Content: input,
		},
	}, nil
}

// DefaultResponseParser returns the raw text output unchanged.
func DefaultResponseParser(output string) (any, error) {
	return output, nil
}
