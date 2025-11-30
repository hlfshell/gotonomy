package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

const SessionKey = "session"

// PrepareInput converts tool arguments and the current Session into
// a set of model messages for the next LLM call.
type PrepareInput func(args tool.Arguments, sess *Session) ([]model.Message, error)

// ResponseParser parses the final LLM text output into a typed value and
// optional warnings.
type ResponseParser func(output string) (any, []string)

// Agent is a simple expandable type of tool that utilizes an LLM
// to accomplish a task. It implements tool.Tool directly so it can be
// called as a child tool from other agents.
type Agent struct {
	// name is the name of the agent. When used as a tool, this must be globally unique.
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

	// maxIterations is the maximum number of LLM/tool iterations before failing.
	maxIterations int
}

// AgentOption is a functional option for configuring an Agent.
type AgentOption func(*Agent)

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
func DefaultResponseParser(output string) (any, []string) {
	return output, nil
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

	a := &Agent{
		name:          name,
		description:   description,
		parameters:    defaultParams,
		model:         model,
		tools:         make(map[string]tool.Tool),
		prepareInput:  DefaultArgumentsToMessages,
		parseResponse: DefaultResponseParser,
		maxIterations: 16,
	}

	// Apply all options
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// WithPrompt is currently a no-op placeholder for future prompt templating.
// It exists to avoid breaking API, but does not alter behavior yet.
// Future versions will wire this into a prompt template system.
func WithPrompt(prompt string) AgentOption {
	return func(a *Agent) {
		// intentionally no-op
		_ = prompt
	}
}

// WithArgumentsToMessages sets a custom arguments-to-messages function for the agent.
func WithArgumentsToMessages(fn PrepareInput) AgentOption {
	return func(a *Agent) {
		if fn != nil {
			a.prepareInput = fn
		}
	}
}

// WithParser sets a custom output parser for the agent.
// The parser extracts structured data from the agent's final text output.
func WithParser(parser ResponseParser) AgentOption {
	return func(a *Agent) {
		if parser != nil {
			a.parseResponse = parser
		}
	}
}

// WithParameters sets the parameters for the agent.
func WithParameters(parameters []tool.Parameter) AgentOption {
	return func(a *Agent) {
		a.parameters = parameters
	}
}

// WithTool adds a tool to the agent.
func WithTool(t tool.Tool) AgentOption {
	return func(a *Agent) {
		if a.tools == nil {
			a.tools = make(map[string]tool.Tool)
		}
		a.tools[t.Name()] = t
	}
}

// WithTools adds multiple tools to the agent.
func WithTools(tools []tool.Tool) AgentOption {
	return func(a *Agent) {
		if a.tools == nil {
			a.tools = make(map[string]tool.Tool)
		}
		for _, t := range tools {
			a.tools[t.Name()] = t
		}
	}
}

// WithMaxIterations overrides the default maximum number of reasoning iterations.
func WithMaxIterations(n int) AgentOption {
	return func(a *Agent) {
		if n > 0 {
			a.maxIterations = n
		}
	}
}

// Name returns the name of the agent. When used as a tool, this serves as the unique identifier.
func (a *Agent) Name() string {
	return a.name
}

// Description returns a description of the agent.
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
//  2. Load or create a Session from the global ledger.
//  3. Repeatedly build messages, call the model, handle tool calls, and
//     either continue or return the final parsed result.
func (a *Agent) Execute(ctx *tool.Context, args tool.Arguments) tool.ResultInterface {
	// 1) Ensure we have a proper context for this agent call.
	ctx = tool.PrepareContext(ctx, a, args)
	ctx.Stats().MarkStarted()
	defer ctx.Stats().MarkFinished()

	// 2) Start our session for internal agent looping
	session := NewSession()
	// When this function leaves, we save the current
	// state of the session to our context's scoped
	// data ledger.
	defer ctx.Data().SetData(SessionKey, session)

	//todo - if no max iterations, go infinite
	// todo - add timeout
	maxIterations := a.maxIterations
	if maxIterations <= 0 {
		maxIterations = 16
	}

	for iter := 0; iter < maxIterations; iter++ {
		//todo - no magic strings, consts for names
		ctx.Stats().Incr("iterations")

		// 1) Build messages from args + session.
		messages, err := a.prepareInput(args, sess)
		if err != nil {
			return tool.NewError(fmt.Errorf("building messages: %w", err))
		}

		// 2) Create Step.
		step := NewStep(messages)

		// 3) Call model.
		//TODO - use tool contexts, not golang contexts!
		resp, err := a.model.Complete(context.Background(), model.CompletionRequest{
			Messages: messages,
			Tools:    a.toolsSlice(),
			Config:   model.ModelConfig{},
		})

		if err != nil {
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

		// 4) Attach response to step & session.
		step.SetResponse(ResponseFromModel(resp))
		session.AddStep(step)

		// Track tool calls at the agent level.
		if len(resp.ToolCalls) > 0 {
			//TODO - no magic strings
			ctx.Stats().Add("tool_calls", int64(len(resp.ToolCalls)))
		}

		// 5) If response has tool calls, handle them then continue loop.
		if len(resp.ToolCalls) > 0 {
			//TODO - some tool clals should be able to fail
			// and the LLM should be able to handle it - BUT
			// also the agent should have a setting for if this
			// is the case
			if err := a.handleToolCalls(ctx, session, step); err != nil {
				return tool.NewError(err)
			}
			// After tool calls, continue loop to let model reason again.
			continue
		}

		// 6) No tool calls: finalize / parse and return.
		// TODO - agents should have a termination strategy
		// which is checked prior to aborting which passes
		// in the entire session
		finalText := resp.Text
		if a.parseResponse == nil {
			return tool.NewOK(finalText)
		}

		parsed, warnings := a.parseResponse(finalText)
		if len(warnings) > 0 {
			ctx.Stats().Set("agent_parse_warnings", warnings)
		}
		return tool.NewOK(parsed)
	}

	// If we exit the loop: too many iterations TODO or timeout
	return tool.NewError(fmt.Errorf("agent %s exceeded max iterations", a.name))
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

// handleToolCalls executes tool calls requested by the model using the provided
// parent context. Tool results are optionally appended to the Session as tool
// messages so they can be fed back into subsequent LLM calls.
// TODO - changeable "on error" behavior AND optional parallelization
func (a *Agent) handleToolCalls(parentCtx *tool.Context, sess *Session, step *Step) error {
	for _, call := range step.GetResponse().ToolCalls {
		toolName := call.Name
		t, ok := a.tools[toolName]
		if !ok {
			return fmt.Errorf("unknown tool: %s", toolName)
		}

		// Arguments are already in tool.Arguments form.
		args := call.Arguments

		// Delegate to the tool system; the child tool will get its own Context
		// via PrepareContext inside its Execute implementation.
		res := t.Execute(parentCtx, args)

		// Convert result into a tool message for the conversation.
		content, err := res.String()
		if err != nil {
			content = fmt.Sprintf("tool %s error: %v", toolName, err)
		}

		toolMsg := model.Message{
			Role:    model.RoleTool,
			Content: content,
		}

		// Attach tool message to the current session so it can be replayed on
		// the next LLM call.
		sess.AppendToolMessage(toolMsg)
	}
	return nil
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
