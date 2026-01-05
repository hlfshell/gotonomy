package agent

import "github.com/hlfshell/gotonomy/tool"

// AgentOption is a functional option for configuring an Agent.
type AgentOption func(*Agent)

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
			// Keep the extractor in sync with the parser unless a custom
			// extractor has been explicitly provided.
			if a.extractResult == nil {
				a.extractResult = ExtractorFromParser(parser, false)
			}
		}
	}
}

// WithExtractor sets a custom extractor for the agent. This allows advanced
// workflows (e.g. judge agents or quality gates) that need full control over
// when the agent stops and what feedback is fed back into the model.
func WithExtractor(extractor ExtractResult) AgentOption {
	return func(a *Agent) {
		if extractor != nil {
			a.extractResult = extractor
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
			a.config.MaxSteps = n
		}
	}
}

// WithID sets a custom globally unique identifier for the agent.
// If not provided, a default ID will be generated from the name.
// The ID should be globally unique (e.g., "hlfshell/my_agent").
func WithID(id string) AgentOption {
	return func(a *Agent) {
		if id != "" {
			a.id = id
		}
	}
}
