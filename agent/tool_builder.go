package agent

import (
	"context"

	"github.com/hlfshell/gogentic/tool"
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
	handler func(ctx context.Context, args tool.Arguments) tool.ResultInterface
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
// The handler receives tool.Arguments and returns a tool.ResultInterface.
func (b *ToolBuilder) SetHandler(handler func(ctx context.Context, args tool.Arguments) tool.ResultInterface) *ToolBuilder {
	b.handler = handler
	return b
}

// Build builds the tool.
// TODO: This needs to be updated to use the new tool.NewTool API with []tool.Parameter
// For now, this is a placeholder that needs to be fully implemented.
func (b *ToolBuilder) Build() tool.Tool {
	// TODO: Convert b.parameters (map[string]interface{}) to []tool.Parameter
	// and use tool.NewTool to create the tool properly
	// This is a breaking change that requires refactoring the ToolBuilder API
	panic("ToolBuilder.Build() needs to be updated to use the new tool.Parameter API")
}
