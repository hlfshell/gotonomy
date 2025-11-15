// Package tool provides a tool-using agent implementation.
package tool

import (
	"github.com/hlfshell/gogentic/pkg/agent"
)

// ToolBuilder helps build tool definitions with proper JSON schema.
type ToolBuilder struct {
	// name is the name of the tool.
	name string
	// description is a description of what the tool does.
	description string
	// parameters is a map of parameter definitions.
	parameters map[string]interface{}
	// required is a list of required parameter names.
	required []string
	// handler is the function that handles the tool call.
	// Can be ToolHandler (legacy) or ToolHandlerInterface (new).
	handler interface{}
}

// NewToolBuilder creates a new tool builder.
func NewToolBuilder(name, description string) *ToolBuilder {
	return &ToolBuilder{
		name:        name,
		description: description,
		parameters:  map[string]interface{}{},
		required:    []string{},
	}
}

// AddParameter adds a parameter to the tool.
func (b *ToolBuilder) AddParameter(name, description, type_name string, required bool) *ToolBuilder {
	// Add the parameter
	b.parameters[name] = map[string]interface{}{
		"type":        type_name,
		"description": description,
	}

	// Add to required list if needed
	if required {
		b.required = append(b.required, name)
	}

	return b
}

// AddStringParameter adds a string parameter to the tool.
func (b *ToolBuilder) AddStringParameter(name, description string, required bool) *ToolBuilder {
	return b.AddParameter(name, description, "string", required)
}

// AddNumberParameter adds a number parameter to the tool.
func (b *ToolBuilder) AddNumberParameter(name, description string, required bool) *ToolBuilder {
	return b.AddParameter(name, description, "number", required)
}

// AddIntegerParameter adds an integer parameter to the tool.
func (b *ToolBuilder) AddIntegerParameter(name, description string, required bool) *ToolBuilder {
	return b.AddParameter(name, description, "integer", required)
}

// AddBooleanParameter adds a boolean parameter to the tool.
func (b *ToolBuilder) AddBooleanParameter(name, description string, required bool) *ToolBuilder {
	return b.AddParameter(name, description, "boolean", required)
}

// AddArrayParameter adds an array parameter to the tool.
func (b *ToolBuilder) AddArrayParameter(name, description string, items map[string]interface{}, required bool) *ToolBuilder {
	// Add the parameter
	b.parameters[name] = map[string]interface{}{
		"type":        "array",
		"description": description,
		"items":       items,
	}

	// Add to required list if needed
	if required {
		b.required = append(b.required, name)
	}

	return b
}

// AddObjectParameter adds an object parameter to the tool.
func (b *ToolBuilder) AddObjectParameter(name, description string, properties map[string]interface{}, required_props []string, required bool) *ToolBuilder {
	// Add the parameter
	param := map[string]interface{}{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}

	if len(required_props) > 0 {
		param["required"] = required_props
	}

	b.parameters[name] = param

	// Add to required list if needed
	if required {
		b.required = append(b.required, name)
	}

	return b
}

// SetHandler sets the handler function for the tool.
// Accepts either ToolHandler (legacy) or ToolHandlerInterface (new).
func (b *ToolBuilder) SetHandler(handler interface{}) *ToolBuilder {
	b.handler = handler
	return b
}

// Build builds the tool.
func (b *ToolBuilder) Build() agent.Tool {
	// Create the parameters schema
	schema := map[string]interface{}{
		"type":       "object",
		"properties": b.parameters,
	}

	if len(b.required) > 0 {
		schema["required"] = b.required
	}

	// Wrap legacy ToolHandler if needed
	var handler interface{} = b.handler
	if toolHandler, ok := b.handler.(agent.ToolHandler); ok {
		// Wrap legacy handler to implement ToolHandlerInterface
		handler = agent.NewStringToolHandler(b.name, toolHandler)
	}

	// Create and return the tool
	return agent.Tool{
		Name:        b.name,
		Description: b.description,
		Parameters:  schema,
		Handler:     handler,
	}
}

