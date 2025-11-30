package tool

import (
	"fmt"
	"reflect"
)

// Type is a helper to grab the type
// without instantiating a zero value
func Type[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// Arguments is a type alias for tool/agent arguments
type Arguments = map[string]any

// Tool represents a tool that an agent can use.
// Agents implement this interface directly, allowing them to be used as tools.
// Functions and other functionality can be wrapped as tools using helper functions.
//
// Framework callers (agents) should ensure Execute is called after PrepareExecution
// to maintain proper Execution/Node wiring. Tools can safely assume e is non-nil and fully
// initialized in normal framework use, as PrepareExecution will have prepared it.
//
// Note: e *Execution may be nil or blank only if the caller skipped PrepareExecution and
// used Tool.Execute directly. In the intended framework usage, callers should run
// PrepareExecution first, so tools normally see a fully initialized Execution.
type Tool interface {
	// Name returns the name of the tool - must be globally unique
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Parameters returns the list of parameters for the tool.
	Parameters() []Parameter

	// Execute executes the tool with the given arguments and returns a result.
	// The e *Execution argument provides access to shared and scoped ledgers
	// (via e.Data(), e.GlobalData(), per-node scopes, etc.).
	// Errors are returned as part of the ResultInterface, not as a separate error.
	// Framework callers should use PrepareExecution before calling Execute to ensure
	// proper Execution/Node setup.
	Execute(e *Execution, args Arguments) ResultInterface
}

// tool wraps a function to make it implement the Tool interface.
type tool struct {
	name        string
	description string
	// parametersByName is used for quick lookups by parameter name
	parametersByName map[string]Parameter
	// parametersOrdered preserves the declaration order provided at construction
	parametersOrdered []Parameter
	handler           func(e *Execution, args Arguments) ResultInterface
}

// Name implements Tool.
func (t *tool) Name() string {
	return t.name
}

// Description implements Tool.
func (t *tool) Description() string {
	return t.description
}

// Parameters implements Tool.
func (t *tool) Parameters() []Parameter {
	// Return a copy to preserve encapsulation while maintaining declaration order
	result := make([]Parameter, len(t.parametersOrdered))
	copy(result, t.parametersOrdered)
	return result
}

// validateArguments validates arguments against declared parameters.
// It returns validated arguments and an error if validation fails.
func validateArguments(
	args Arguments,
	parametersOrdered []Parameter,
	parametersByName map[string]Parameter,
) (Arguments, error) {
	validated := make(Arguments, len(args))

	// Validate against declared parameters using Parameter.Value (defaults + type checking)
	for _, param := range parametersOrdered {
		name := param.Name()
		raw, has := args[name]
		if !has {
			raw = nil
		}
		finalValue, err := param.Value(raw)
		if err != nil {
			// Preserve a clear error message per parameter
			return nil, fmt.Errorf("argument %s: %w", name, err)
		}
		// Only set if explicitly provided or a default applied
		if finalValue != nil {
			validated[name] = finalValue
		} else if param.Required() {
			// Defensive: should be unreachable due to Value/TypeCheck semantics
			return nil, fmt.Errorf("missing required argument: %s", name)
		}
	}

	// Reject any extra, undeclared arguments to prevent silent typos/misuse
	for name, value := range args {
		_ = value
		if _, known := parametersByName[name]; !known {
			return nil, fmt.Errorf("unknown argument: %s", name)
		}
	}

	return validated, nil
}

// Execute implements Tool
func (t *tool) Execute(ctx *Node, args Arguments) ResultInterface {
	if ctx == nil {
		e, node, err := PrepareExecution(nil, "", t, args)
		if err != nil {
			return NewError(err)
		}
	}

	validated, err := validateArguments(args, t.parametersOrdered, t.parametersByName)
	if err != nil {
		return NewError(err)
	}

	// Execute the handler with validated arguments
	return t.handler(ctx, validated)
}

// NewTool creates a type-safe tool that automatically wraps the result.
// This is the primary way to create tools with custom return types.
// The handler function can return any type T and an error. If an error occurs,
// it will be automatically converted to an error result. Otherwise, the value
// will be wrapped in a successful result.
//
// The e *Execution argument allows tools to access shared and scoped ledgers
// (via e.Data(), e.GlobalData(), per-node scopes, etc.). Framework callers
// should use PrepareExecution before calling Execute to ensure proper Execution/Node setup.
//
// Example:
//
//	type WeatherData struct {
//	    Temperature float64
//	    Condition   string
//	}
//
//	tool := NewTool[WeatherData](
//	    "get_weather",
//	    "Gets the current weather",
//	    []Parameter{...},
//	    func(e *Execution, args Arguments) (WeatherData, error) {
//	        location := args["location"].(string)
//	        // Can access e.Data(), e.GlobalData(), etc. if needed
//	        return fetchWeather(location), nil
//	    },
//	)
func NewTool[T any](
	name, description string,
	parameters []Parameter,
	handler func(e *Execution, args Arguments) (T, error),
) Tool {
	// Build lookup map and preserve declaration order
	paramsMap := make(map[string]Parameter, len(parameters))
	ordered := make([]Parameter, 0, len(parameters))
	for _, p := range parameters {
		paramsMap[p.Name()] = p
		ordered = append(ordered, p)
	}

	return &tool{
		name:              name,
		description:       description,
		parametersByName:  paramsMap,
		parametersOrdered: ordered,
		handler: func(e *Execution, args Arguments) ResultInterface {
			result, err := handler(e, args)
			if err != nil {
				return NewError(err)
			}
			return NewOK(result)
		},
	}
}
