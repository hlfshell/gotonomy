package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
	"github.com/hlfshell/gotonomy/utils/semver"
)

const SessionKey = "session"

// PrepareInput converts tool arguments and the current Session into
// a set of model messages for the next LLM call.
type PrepareInput func(args tool.Arguments, sess *Session) ([]model.Message, error)

// ResponseParser parses the final LLM text output into a typed value or
// returns an error if parsing fails.
type ResponseParser func(output string) (any, error)

const (
	StopOnFirstToolError string = "stop_on_first_error"
	PassErrorsToModel    string = "pass_errors_to_model"
	FunctionOnError      string = "function_on_error"
)

type AgentConfig struct {
	MaxSteps          int           `json:"max_iterations"`   // if 0, no max iterations
	Timeout           time.Duration `json:"timeout"`          // if 0, no timeout
	MaxToolWorkers    int           `json:"max_tool_workers"` // if 0, no max tool workers
	ToolErrorHandling string        `json:"error_handling"`   // one of StopOnFirstToolError, PassErrorsToModel, FunctionOnError
	// If ToolErrorHandling is FunctionOnError, this function is called to handle the error
	// The function accepts the tool.ResultInterface and returns either a  new tool.ResultInterface
	// or an error to stringify and pass to the model for the tool's failure. Note that the ResultInterface
	// can optionally still contain the error, allowing default parsing to handle it.
	OnToolErrorFunction func(tool.ResultInterface) (tool.ResultInterface, error) `json:"on_error_function"`
}

// IterationChecker returns a function that is called to check if the agent should continue or
// halt based on its settings. The function returns an error if the agent should stop, nil if it should continue.
func (c *AgentConfig) IterationChecker() func(session *Session) error {
	return func(session *Session) error {
		if c.MaxSteps > 0 && len(session.Steps()) >= c.MaxSteps {
			return fmt.Errorf("exceeded max steps of %d", c.MaxSteps)
		}
		if c.Timeout > 0 && session.Duration() > c.Timeout {
			return fmt.Errorf("exceeded timeout of %s", c.Timeout)
		}
		return nil
	}
}

// Agent is a simple expandable type of tool that utilizes an LLM
// to accomplish a task. It implements tool.Tool directly so it can be
// called as a child tool from other agents.
type Agent struct {
	// id is the globally unique identifier of the agent (e.g., "hlfshell/my_agent")
	id string
	// name is the human-readable name of the agent (can have overlaps)
	name string
	// description is a human readable description of the agent.
	description string
	// parameters is the list of parameters the agent accepts.
	parameters []tool.Parameter
	// model is the language model to use. (Required)
	model model.Model
	// tools is the registry of tools the agent can call.
	tools map[string]tool.Tool

	// prepareInput converts arguments + session into model messages.
	prepareInput PrepareInput
	// parseResponse parses the final assistant text into a typed result.
	parseResponse ResponseParser

	// extractResult inspects the full Session and decides whether the agent
	// should stop iterating, optionally returning a typed result and feedback
	// messages to feed into the next iteration.
	extractResult ExtractResult

	config AgentConfig
}

// NewAgent creates a new agent with the given name, description, and model.
// Additional configuration can be provided via AgentOption functions.
//
// Example:
//
//	agent := agent.NewAgent(
//	    "my_agent",
//	    "Performs calculations",
//	    myModel,
//	    agent.WithPrompt("You are a helpful calculator"),
//	    agent.WithTools(tool1, tool2),
//	)
func NewAgent(
	name, description string,
	model model.Model,
	opts ...AgentOption,
) *Agent {
	// Create agent with sensible defaults - default parameter is "input"
	defaultParams := []tool.Parameter{
		tool.NewParameter[string](
			"input",
			"The input for the agent",
			true,
			"",
			func(v string) (string, error) { return v, nil },
		),
	}

	// Generate a globally unique ID from the name if not provided
	// Use a default prefix to ensure uniqueness
	agentID := fmt.Sprintf("agent/%s", name)

	a := &Agent{
		id:            agentID,
		name:          name,
		description:   description,
		parameters:    defaultParams,
		model:         model,
		tools:         make(map[string]tool.Tool),
		prepareInput:  DefaultArgumentsToMessages,
		parseResponse: DefaultResponseParser,
		extractResult: nil,
		config:        DefaultAgentConfig,
	}

	// Apply all options
	for _, opt := range opts {
		opt(a)
	}

	// If no custom extractor was provided, derive a default one from the
	// configured parser. This keeps the developer API simple: providing a
	// ResponseParser is enough to get structured results.
	if a.extractResult == nil {
		a.extractResult = ExtractorFromParser(a.parseResponse, false)
	}

	return a
}

func (a *Agent) ID() string {
	return a.id
}

func (a *Agent) Version() semver.SemVer {
	return semver.SemVer{} // Default version for agents
}

func (a *Agent) Name() string {
	return a.name
}

func (a *Agent) Description() string {
	return a.description
}

// Parameters returns the list of parameters for the agent.
func (a *Agent) Parameters() []tool.Parameter {
	// Preserve declaration order; return a shallow copy for encapsulation
	result := make([]tool.Parameter, len(a.parameters))
	copy(result, a.parameters)
	return result
}

// Execute executes the agent with the given arguments and returns a result.
// This method implements the tool.Tool interface, allowing agents to be used as tools.
// Errors are returned as part of the ResultInterface, not as a separate error.
//
// The execution loop:
//  1. Prepare a tool.Context and mark stats started/finished.
//  2. Load or create a Session from the context-scoped ledger.
//  3. Repeatedly build messages, call the model, handle tool calls, and
//     either continue or return the final parsed result.
func (a *Agent) Execute(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
	// 1) Ensure we have a proper context for this agent call.
	ctx = tool.PrepareContext(ctx, a, args)
	ctx.Stats().MarkStarted()
	defer ctx.Stats().MarkFinished()

	// 2) Start our session for internal agent looping
	// Try to load existing session from context, or create new one
	sessionLedger := ctx.Data()
	session := NewSession(sessionLedger)
	// When this function leaves, we save the current
	// state of the session to our context's scoped
	// data ledger.
	// TODO - this was done due to the ledger interface on
	// the context, but since that's changing, this odd
	// code smell can be reworked to just write it outright
	defer ctx.Data().SetData(SessionKey, session)

	// 3) Get the iteration checker from config
	shouldContinue := a.config.IterationChecker()

	// Main iteration loop - continues until checker returns false
	iteration := 0
	for {
		iteration++
		// Check if we should continue before starting this iteration
		if err := shouldContinue(session); err != nil {
			fmt.Printf("[AGENT DEBUG] Stopping due to iteration checker: %v\n", err)
			return tool.NewError(err)
		}

		//todo - no magic strings, consts for names
		ctx.Stats().Incr("iterations")
		fmt.Printf("\n[AGENT DEBUG] ===== Iteration %d =====\n", iteration)

		// 1) Build messages from args + session.
		messages, err := a.prepareInput(args, session)
		if err != nil {
			fmt.Printf("[AGENT DEBUG] Error building messages: %v\n", err)
			return tool.NewError(fmt.Errorf("building messages: %w", err))
		}
		fmt.Printf("[AGENT DEBUG] Prepared %d messages for model\n", len(messages))
		for i, msg := range messages {
			fmt.Printf("[AGENT DEBUG]   Message[%d]: %s: %q\n", i, msg.Role, truncateString(msg.Content, 150))
		}
		fmt.Printf("[AGENT DEBUG] Available tools: %d\n", len(a.toolsSlice()))
		for _, t := range a.toolsSlice() {
			fmt.Printf("[AGENT DEBUG]   - %s: %s\n", t.Name(), truncateString(t.Description(), 60))
		}

		// 2) Create Step.
		step := NewStep(messages)

		// 3) Call model.
		//TODO - use tool contexts, not golang contexts!
		fmt.Printf("[AGENT DEBUG] Calling model...\n")

		// Write outgoing messages to debug file
		debugFilename := fmt.Sprintf("debug_iteration_%d_%s.json", iteration, time.Now().Format("150405"))
		debugData := map[string]interface{}{
			"iteration":         iteration,
			"timestamp":         time.Now().Format(time.RFC3339),
			"outgoing_messages": messages,
			"tools_count":       len(a.toolsSlice()),
		}
		if debugJSON, err := json.MarshalIndent(debugData, "", "  "); err == nil {
			os.WriteFile(debugFilename, debugJSON, 0644)
			fmt.Printf("[AGENT DEBUG] Outgoing messages written to: %s\n", debugFilename)
		}

		resp, err := a.model.Complete(context.Background(), model.CompletionRequest{
			Messages: messages,
			Tools:    a.toolsSlice(),
			Config:   model.ModelConfig{},
		})

		if err != nil {
			fmt.Printf("[AGENT DEBUG] Model call error: %v\n", err)
			// Record error in session and return an error result.
			step.SetResponse(Response{
				Output: model.Message{
					Role:    model.RoleAssistant,
					Content: "",
				},
				ToolCalls: nil,
				Error:     err.Error(),
			})
			session.AddStep(step)
			return tool.NewError(err)
		}

		fmt.Printf("[AGENT DEBUG] Model response received:\n")
		fmt.Printf("[AGENT DEBUG]   Text: %q\n", truncateString(resp.Text, 200))
		fmt.Printf("[AGENT DEBUG]   Tool calls: %d\n", len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			fmt.Printf("[AGENT DEBUG]     [%d] %s(%v)\n", i, tc.Name, tc.Arguments)
		}
		fmt.Printf("[AGENT DEBUG]   Usage: %d input, %d output tokens\n", resp.UsageStats.InputTokens, resp.UsageStats.OutputTokens)

		// Write raw response to debug file
		responseDebugData := map[string]interface{}{
			"iteration": iteration,
			"timestamp": time.Now().Format(time.RFC3339),
			"response": map[string]interface{}{
				"text":       resp.Text,
				"tool_calls": resp.ToolCalls,
				"usage": map[string]interface{}{
					"input_tokens":  resp.UsageStats.InputTokens,
					"output_tokens": resp.UsageStats.OutputTokens,
				},
			},
		}
		if responseDebugJSON, err := json.MarshalIndent(responseDebugData, "", "  "); err == nil {
			responseDebugFilename := fmt.Sprintf("debug_response_%d_%s.json", iteration, time.Now().Format("150405"))
			os.WriteFile(responseDebugFilename, responseDebugJSON, 0644)
			fmt.Printf("[AGENT DEBUG] Raw response written to: %s\n", responseDebugFilename)
		}

		// 4) Attach response to step & session.
		step.SetResponse(ResponseFromModel(resp))
		session.AddStep(step)

		// Track tool calls via ctx metrics.
		if len(resp.ToolCalls) > 0 {
			//TODO - no magic strings
			ctx.Stats().Add("tool_calls", int64(len(resp.ToolCalls)))
			fmt.Printf("[AGENT DEBUG] Executing %d tool calls...\n", len(resp.ToolCalls))
			// Execute tools before making any extraction decisions.
			// Some extractors may depend on tool outputs being present
			// in the session's conversation.
			if err := a.handleToolCalls(ctx, session, step); err != nil {
				fmt.Printf("[AGENT DEBUG] Tool execution error: %v\n", err)
				return tool.NewError(err)
			}
			fmt.Printf("[AGENT DEBUG] Tool calls completed\n")
		}

		fmt.Printf("[AGENT DEBUG] Calling extractResult...\n")
		decision := a.extractResult(a, ctx, session)
		fmt.Printf("[AGENT DEBUG] ExtractDecision:\n")
		fmt.Printf("[AGENT DEBUG]   Done: %v\n", decision.Done)
		fmt.Printf("[AGENT DEBUG]   Result: %v (type: %T)\n", decision.Result, decision.Result)
		fmt.Printf("[AGENT DEBUG]   Warnings: %v\n", decision.Warnings)
		fmt.Printf("[AGENT DEBUG]   Feedback messages: %d\n", len(decision.Feedback))
		if decision.Err != nil {
			fmt.Printf("[AGENT DEBUG]   Error: %v\n", decision.Err)
			return tool.NewError(decision.Err)
		}

		// Append any feedback messages so they are included in the next
		// iteration's step.
		for _, msg := range decision.Feedback {
			session.AppendToolMessage(msg)
		}

		if !decision.Done {
			fmt.Printf("[AGENT DEBUG] Not done, continuing to next iteration...\n")
			// Continue iterating; extractor has not reached a terminal state.
			continue
		}

		fmt.Printf("[AGENT DEBUG] Terminal state reached!\n")
		// Terminal state reached - determine the final result to return.
		result := decision.Result
		if result == nil {
			fmt.Printf("[AGENT DEBUG] Result is nil, falling back to last assistant message\n")
			// Fall back to the last assistant message text.
			last := session.LastStep()
			if last != nil {
				result = last.GetResponse().Output.Content
				fmt.Printf("[AGENT DEBUG] Fallback result: %q\n", result)
			} else {
				fmt.Printf("[AGENT DEBUG] No last step available for fallback\n")
			}
		} else {
			fmt.Printf("[AGENT DEBUG] Using decision result: %v\n", result)
		}

		if len(decision.Warnings) > 0 {
			ctx.Stats().Set("agent_extract_warnings", decision.Warnings)
		}

		fmt.Printf("[AGENT DEBUG] Returning final result: %v (type: %T)\n", result, result)
		return tool.NewOK(result)
	}
}

// truncateString truncates a string for debug output
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// toolsSlice returns the agent's tools in a deterministic (sorted-by-name) slice.
// TODO - tools should be presented in the order the user specified
// them, preventing any decision making from being done based on either
// random chance or some unexplained behavior.
func (a *Agent) toolsSlice() []tool.Tool {
	if len(a.tools) == 0 {
		return nil
	}

	names := make([]string, 0, len(a.tools))
	for name := range a.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]tool.Tool, 0, len(names))
	for _, name := range names {
		result = append(result, a.tools[name])
	}
	return result
}

// ResponseFromModel converts a model.CompletionResponse into an agent Response.
func ResponseFromModel(resp model.CompletionResponse) Response {
	return Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: resp.Text,
		},
		ToolCalls: resp.ToolCalls,
	}
}
