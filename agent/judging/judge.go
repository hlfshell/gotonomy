package judging

import (
	"encoding/json"
	"fmt"

	"github.com/hlfshell/gotonomy/agent"
	"github.com/hlfshell/gotonomy/assets"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

// Verdict is the judge's decision for a step.
type Verdict string

const (
	VerdictPass   Verdict = "pass"
	VerdictFail   Verdict = "fail"
	VerdictReplan Verdict = "replan"
)

func (v Verdict) Validate() error {
	switch v {
	case VerdictPass, VerdictFail, VerdictReplan:
		return nil
	default:
		return fmt.Errorf("invalid verdict %q", string(v))
	}
}

// JudgeResult is the structured output of the judge agent.
type JudgeResult struct {
	Verdict       Verdict `json:"verdict"`
	Justification string  `json:"justification"`
	SuggestedFix  string  `json:"suggested_fix,omitempty"`
}

func (r JudgeResult) Validate() error {
	if err := r.Verdict.Validate(); err != nil {
		return err
	}
	if r.Justification == "" {
		return fmt.Errorf("justification is required")
	}
	return nil
}

// NewJudgeAgent constructs an LLM-based judge as a standard gotonomy agent.
//
// The judge does NOT use tools. It outputs strict JSON only:
// {"verdict":"pass|fail|replan","justification":"...","suggested_fix":"...optional..."}
func NewJudgeAgent(m model.Model) *agent.Agent {
	parser := func(output string) (any, error) {
		var res JudgeResult
		if err := json.Unmarshal([]byte(output), &res); err != nil {
			return nil, err
		}
		if err := res.Validate(); err != nil {
			return nil, err
		}
		return res, nil
	}

	extractor := newJudgeExtractor(parser)

	return agent.NewAgent(
		"judge",
		"Judges whether a step output satisfies its expectation; returns pass|fail|replan with justification.",
		m,
		agent.WithParameters([]tool.Parameter{
			tool.NewParameter[string]("objective", "Overall objective of the plan.", true, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[string]("step_name", "Human-friendly step name.", false, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[string]("instruction", "Instruction that was executed.", true, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[string]("expectation", "What success looks like for this step.", true, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[string]("output", "Actual output produced for the step.", true, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[string]("context", "Optional additional context (prior outputs, constraints, etc.).", false, "", func(v string) (string, error) { return v, nil }),
		}),
		agent.WithArgumentsToMessages(judgeArgumentsToMessages),
		agent.WithParser(parser),
		agent.WithExtractor(extractor),
		agent.WithMaxIterations(3),
	)
}

func judgeArgumentsToMessages(args tool.Arguments, sess *agent.Session) ([]model.Message, error) {
	// Replay conversation if we already have iterations.
	if sess != nil && len(sess.Steps()) > 0 {
		return sess.Conversation(), nil
	}

	// Load the judge prompt template
	tmpl, err := assets.LoadPrompt("judge.prompt")
	if err != nil {
		return nil, fmt.Errorf("failed to load judge prompt: %w", err)
	}

	objective := args["objective"].(string)
	stepName := ""
	if v, ok := args["step_name"].(string); ok {
		stepName = v
	}
	instruction := args["instruction"].(string)
	expectation := args["expectation"].(string)
	output := args["output"].(string)
	context := ""
	if v, ok := args["context"].(string); ok {
		context = v
	}

	// Render the prompt template
	templateData := map[string]interface{}{
		"objective":  objective,
		"step_name":  stepName,
		"instruction": instruction,
		"expectation": expectation,
		"output":      output,
		"context":     context,
	}

	rendered, err := tmpl.Render(templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render judge prompt: %w", err)
	}

	// The rendered prompt contains the full instructions and data
	return []model.Message{
		{Role: model.RoleSystem, Content: rendered},
	}, nil
}


