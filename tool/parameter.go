package tool

import (
	"errors"
	"fmt"
	"reflect"
)

type Parameter struct {
	name           string
	description    string
	value_type     reflect.Type
	required       bool
	defaultValue   interface{}
	stringFunction func(value interface{}) (string, error)
}

// String returns a human-readable string for the given value using the parameter's
// stringFunction if provided. Falls back to fmt.Sprintf("%v", value) otherwise.
func (p *Parameter) String(value interface{}) (string, error) {
	if p.stringFunction == nil {
		return fmt.Sprintf("%v", value), nil
	}
	return p.stringFunction(value)
}

// Name returns the parameter name.
func (p *Parameter) Name() string {
	return p.name
}

// Description returns the parameter description.
func (p *Parameter) Description() string {
	return p.description
}

// Type returns the parameter type.
func (p *Parameter) Type() reflect.Type {
	return p.value_type
}

// Default returns the default value for the parameter.
func (p *Parameter) Default() interface{} {
	return p.defaultValue
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
		name:         name,
		description:  description,
		value_type:   Type[T](),
		required:     required,
		defaultValue: defaultValue,
		stringFunction: func(value interface{}) (string, error) {
			// Strict type assertion - fail clearly if type doesn't match
			v, ok := value.(T)
			if !ok {
				expected := Type[T]()
				return "", fmt.Errorf("parameter %s expects %s, got %T", name, expected, value)
			}
			return stringFunc(v)
		},
	}
}

// Required returns whether the argument is required.
func (a *Parameter) Required() bool {
	return a.required
}

// TypeCheck validates that the value matches the parameter's type.
// Returns nil if valid, or an error if the type doesn't match.
func (a *Parameter) TypeCheck(value interface{}) error {
	if a.Type() == nil {
		return errors.New("parameter type is nil")
	}

	// Handle nil values
	if value == nil {
		if a.Required() {
			return fmt.Errorf("parameter %s: required but value is nil", a.Name())
		}
		// Optional parameters can be nil
		return nil
	}

	valType := reflect.TypeOf(value)
	if !valType.AssignableTo(a.Type()) {
		return fmt.Errorf("parameter %s: expected type %s, got %s", a.Name(), a.Type().String(), valType.String())
	}
	return nil
}

// Value applies defaulting and type validation and returns the final value.
// Semantics:
// - If value is nil and a non-nil default exists, the default is used.
// - If after defaulting the value is nil and the parameter is required, an error is returned.
// - A required parameter with a non-nil default is considered satisfied after defaults are applied.
func (a *Parameter) Value(value interface{}) (interface{}, error) {
	if value == nil && a.Default() != nil {
		value = a.Default()
	}
	if err := a.TypeCheck(value); err != nil {
		return nil, err
	}
	return value, nil
}

// typeToJSONSchemaType maps Go reflect.Kind to JSON schema type string.
func typeToJSONSchemaType(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string" // default fallback
	}
}

// ParametersToJSONSchema converts a list of Parameters to a JSON schema map.
func ParametersToJSONSchema(params []Parameter) map[string]any {
	properties := make(map[string]any, len(params))
	required := make([]string, 0, len(params))

	for _, param := range params {
		prop := map[string]any{
			"type":        typeToJSONSchemaType(param.Type().Kind()),
			"description": param.Description(),
		}
		if param.Default() != nil {
			prop["default"] = param.Default()
		}
		properties[param.Name()] = prop

		if param.Required() {
			required = append(required, param.Name())
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}
