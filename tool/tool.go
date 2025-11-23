package tool

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/hlfshell/gogentic/result"
)

// Type is a helper to grab the type
// without instantiating a zero value
func Type[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

type Parameter struct {
	Name        string
	Description string
	Type        reflect.Type
	Required    bool
	Default     interface{}
	String      func(value interface{}) (string, error)
}

// NewParameter creates a type-friendly Argument using generics.
// The Argument struct itself remains untyped (using interface{}),
// but this function allows type-safe creation with compile-time type checking.
//
// Example:
//
//	arg := NewParameter[string](
//		"location",
//		"The location to get weather for",
//		true,  // required
//		"",    // default value
//		func(v string) (string, error) { return v, nil }, // string conversion
//	)
//
//	arg := NewParameter[int](
//		"count",
//		"Number of items",
//		false,
//		10,
//		func(v int) (string, error) { return fmt.Sprintf("%d", v), nil },
//	)
func NewParameter[T any](
	name, description string,
	required bool,
	defaultValue T,
	stringFunc func(T) (string, error),
) Parameter {
	return Parameter{
		Name:        name,
		Description: description,
		Type:        Type[T](),
		Required:    required,
		Default:     defaultValue,
		String: func(value interface{}) (string, error) {
			// Type assert to T and call the type-safe string function
			if v, ok := value.(T); ok {
				return stringFunc(v)
			}
			// Fallback: try to convert using reflection if type assertion fails
			// This handles cases where value might be a different but compatible type
			val := reflect.ValueOf(value)
			if val.IsValid() && val.Type().AssignableTo(Type[T]()) {
				if val.Type().ConvertibleTo(Type[T]()) {
					converted := val.Convert(Type[T]())
					if converted.CanInterface() {
						if v, ok := converted.Interface().(T); ok {
							return stringFunc(v)
						}
					}
				}
			}
			// Last resort: use fmt.Sprintf
			return fmt.Sprintf("%v", value), nil
		},
	}
}

// Required returns whether the argument is required.
func (a *Parameter) Required() bool {
	return a.Required
}

func (a *Parameter) TypeCheck(value interface{}) error {
	if a.Type == nil {
		return errors.New("type is nil")
	}
	if !reflect.TypeOf(value).AssignableTo(a.Type) {
		return fmt.Errorf("value type %s is not type %s", reflect.TypeOf(value).Name(), a.Type.Name())
	}
	return nil
}

func (a *Parameter) Value(value interface{}) (interface{}, error) {
	if value == nil && a.Default != nil {
		value = a.Default
	}
	if err := a.TypeCheck(value); err != nil {
		return nil, err
	}
	return value, nil
}

type Arguments map[string]interface{}

// GotonomyTool represents a tool that an agent can use.
// Agents implement this interface directly, allowing them to be used as tools.
// Functions and other functionality can be wrapped as tools using helper functions.
type GotonomyTool interface {
	// Name returns the name of the tool - must be globally unique
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Parameters returns the JSON schema for the tool's parameters.
	Parameters() []Parameter

	// Execute executes the tool with the given arguments and returns a result.
	// Errors are returned as part of the ResultInterface, not as a separate error.
	Execute(ctx context.Context, args Arguments) result.ResultInterface
}

// tool wraps a function to make it implement the Tool interface.
type tool struct {
	name        string
	description string
	parameters  map[string]Parameter
	handler     func(ctx context.Context, args Arguments) ResultInterface
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
	return values(f.parameters)
}

func (f *tool) verifyArguments(args Arguments) error {
	// First we check for missing required arguments
	for name, arg := range f.parameters {
		if arg.Required() && args[name] == nil {
			return fmt.Errorf("missing required argument: %s", name)
		}
	}
}

func (f *tool) provideDefaults(args Arguments) map[string]interface{} {
	for name, param := range f.parameters {
		if _, ok := args[name]; !ok && param.Default != nil {
			args[name] = param.Default
		}
	}
	return args
}

// Execute implements GotonomyTool.
func (f *tool) Execute(ctx context.Context, args Arguments) ResultInterface {
	// First we verify that the arguments meet the requirements
	// for the given tool

	if err := f.verifyArguments(args); err != nil {
		return NewToolResultError(f.name, err)
	}

	return f.handler(ctx, args)
}

// Tool creates a type-safe tool that automatically wraps the result.
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
//	tool := Tool[WeatherData](
//	    "get_weather",
//	    "Gets the current weather",
//	    arguments,
//	    func(ctx context.Context, args Arguments) (WeatherData, error) {
//	        location := args["location"].(string)
//	        return fetchWeather(location), nil
//	    },
//	)
func Tool[T any](
	name, description string,
	arguments []Argument,
	handler func(ctx context.Context, args Arguments) (T, error),
) GotonomyTool {
	return &tool{
		name:        name,
		description: description,
		arguments:   arguments,
		handler: func(ctx context.Context, args Arguments) Result[T] {
			result, err := handler(ctx, args)
			if err != nil {
				return BlankResult(nil, err)
			}
			return Result[T]{
				Result: result,
				Error:  nil,
			}
		},
	}
}
