package agent

import (
	"fmt"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

// ExtractDecision describes the result of running an ExtractResultFunc.
// It unifies three concerns:
//   - Should the agent stop looping? (Done)
//   - What typed value should be returned to the caller? (Result)
//   - What non-fatal issues or refinement feedback should be recorded? (Warnings, FeedbackMessages)
//
// Any fatal error is returned via Err and will cause the agent to abort with an error ResultInterface.
type ExtractDecision struct {
	// Done indicates whether the agent should stop iterating.
	Done bool

	// Result is the final typed value to return when Done is true.
	// It may be nil, in which case the agent will fall back to the
	// last assistant message text as a string.
	Result any

	// Warnings are non-fatal issues associated with the extracted result.
	// These are used to guide the model on unsuccessful tool execution
	// but you don't want to terminate via an error.
	Warnings []string

	// Feedback are messages that should be appended to the Session
	// so the model can see additional guidance on the next iteration when
	// Done is false.
	Feedback []model.Message

	// Err is a fatal error. If non-nil, the agent will abort and return
	// an error ResultInterface to the caller.
	Err error
}

// ExtractResult inspects the current Session and decides whether the
// agent should stop, optionally returning a typed result and feedback
// messages for the next iteration. If another tool/agent is used within,
// utilize the context to make it a child node of the existing agent.
type ExtractResult func(
	agent *Agent,
	ctx *tool.Context,
	session *Session,
) ExtractDecision

// ExtractorFromParser adapts a simple ResponseParser into an ExtractResultFunc.
// It uses the parser to produce a structured result once the last step has
// no outstanding tool calls. This simplistic pattern covers most agent use
// cases and is provided as a batteries-included piece.
func ExtractorFromParser(
	parser ResponseParser,
	retryOnError bool,
) ExtractResult {
	return func(
		agent *Agent,
		ctx *tool.Context,
		session *Session,
	) ExtractDecision {
		last := session.LastStep()
		if last == nil {
			fmt.Printf("[AGENT DEBUG] Extractor: No last step\n")
			return ExtractDecision{}
		}

		resp := last.GetResponse()
		fmt.Printf("[AGENT DEBUG] Extractor: Last step has %d tool calls\n", len(resp.ToolCalls))
		if len(resp.ToolCalls) > 0 {
			// Still have tool calls to resolve; keep going.
			fmt.Printf("[AGENT DEBUG] Extractor: Tool calls pending, continuing...\n")
			return ExtractDecision{
				Done: false,
			}
		}

		// No tool calls - time to parse!
		fmt.Printf("[AGENT DEBUG] Extractor: No tool calls, parsing response: %q\n", truncateString(resp.Output.Content, 200))
		parsed, err := parser(resp.Output.Content)
		warnings := []string{}
		done := true
		if err != nil {
			fmt.Printf("[AGENT DEBUG] Extractor: Parser error: %v\n", err)
			warnings = append(warnings, fmt.Sprintf("failed to parse response: %v", err))
			// If retryOnError is true, we leave Done as false so that the
			// agent can iterate again, optionally using feedback messages
			// provided by a higher-level extractor wrapper to fix the
			// output from the error.
			if retryOnError {
				done = false
				fmt.Printf("[AGENT DEBUG] Extractor: Retry on error enabled, will continue\n")
			}
		} else {
			fmt.Printf("[AGENT DEBUG] Extractor: Parsed successfully: %v (type: %T)\n", parsed, parsed)
		}
		return ExtractDecision{
			Done:     done,
			Result:   parsed,
			Warnings: warnings,
		}
	}
}
