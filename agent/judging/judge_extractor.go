package judging

import (
	"fmt"

	"github.com/hlfshell/gotonomy/agent"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

// newJudgeExtractor returns an extractor that:
// - waits for tool calls to resolve (though judge should not use tools)
// - parses strict JSON into JudgeResult
// - retries with feedback when parsing/validation fails
func newJudgeExtractor(parser agent.ResponseParser) agent.ExtractResult {
	base := agent.ExtractorFromParser(parser, true)
	return func(a *agent.Agent, ctx *tool.Context, sess *agent.Session) agent.ExtractDecision {
		decision := base(a, ctx, sess)
		if decision.Err != nil {
			return decision
		}
		if decision.Done {
			// Parsed successfully (or we hit done=true with a nil result), nothing to add.
			// Ensure we return a typed result if present.
			if decision.Result == nil {
				return agent.ExtractDecision{Err: fmt.Errorf("judge produced empty result")}
			}
			return decision
		}

		// Not done means parser failed; provide feedback to force strict JSON.
		feedback := model.Message{
			Role: model.RoleSystem,
			Content: `Your previous response was invalid.

Return ONLY valid JSON matching:
{"verdict":"pass|fail|replan","justification":"...","suggested_fix":"...optional..."}

No markdown, no surrounding text, no trailing commas.`,
		}
		decision.Feedback = append(decision.Feedback, feedback)
		return decision
	}
}



