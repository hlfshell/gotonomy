package tool

import (
	"context"
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
type Tool interface {
	// Name returns the name of the tool - must be globally unique
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Parameters returns the list of parameters for the tool.
	Parameters() []Parameter

	// Execute executes the tool with the given arguments and returns a result.
	// Errors are returned as part of the ResultInterface, not as a separate error.
	Execute(ctx context.Context, args Arguments) ResultInterface
}

// tool wraps a function to make it implement the Tool interface.
type tool struct {
	name        string
	description string
	// parametersByName is used for quick lookups by parameter name
	parametersByName map[string]Parameter
	// parametersOrdered preserves the declaration order provided at construction
	parametersOrdered []Parameter
	handler           func(ctx context.Context, args Arguments) ResultInterface
}

// Name implements Tool.
func (f *tool) Name() string {
	return f.name
}

// Description implements Tool.
func (f *tool) Description() string {
	return f.description
}

// Parameters implements Tool.
func (f *tool) Parameters() []Parameter {
	// Return a copy to preserve encapsulation while maintaining declaration order
	result := make([]Parameter, len(f.parametersOrdered))
	copy(result, f.parametersOrdered)
	return result
}

// Execute implements Tool.
func (f *tool) Execute(ctx context.Context, args Arguments) ResultInterface {
	// Validate against declared parameters using Parameter.Value (defaults + type checking)
	validated := make(Arguments, len(args))

	for _, param := range f.parametersOrdered {
		name := param.Name()
		raw, has := args[name]
		if !has {
			raw = nil
		}
		finalValue, err := param.Value(raw)
		if err != nil {
			// Preserve a clear error message per parameter
			return NewError(fmt.Errorf("argument %s: %w", name, err))
		}
		// Only set if explicitly provided or a default applied
		if finalValue != nil {
			validated[name] = finalValue
		} else if param.Required() {
			// Defensive: should be unreachable due to Value/TypeCheck semantics
			return NewError(fmt.Errorf("missing required argument: %s", name))
		}
	}

	// Reject any extra, undeclared arguments to prevent silent typos/misuse
	for name, value := range args {
		_ = value
		if _, known := f.parametersByName[name]; !known {
			return NewError(fmt.Errorf("unknown argument: %s", name))
		}
	}

	// Execute the handler with validated arguments
	return f.handler(ctx, validated)
}

// NewTool creates a type-safe tool that automatically wraps the result.
// This is the primary way to create tools with custom return types.
// The handler function can return any type T and an error. If an error occurs,
// it will be automatically converted to an error result. Otherwise, the value
// will be wrapped in a successful result.
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
//	    func(ctx context.Context, args Arguments) (WeatherData, error) {
//	        location := args["location"].(string)
//	        return fetchWeather(location), nil
//	    },
//	)
func NewTool[T any](
	name, description string,
	parameters []Parameter,
	handler func(ctx context.Context, args Arguments) (T, error),
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
		handler: func(ctx context.Context, args Arguments) ResultInterface {
			result, err := handler(ctx, args)
			if err != nil {
				return NewError(err)
			}
			return NewOK(result)
		},
	}
}
