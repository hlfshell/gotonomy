# Tool Creation Guide

This guide shows how to create type-safe tools for the agent framework.

## Method 1: NewTypedTool (Recommended)

The `NewTypedTool` function provides a clean, type-safe way to create tools without manually wrapping results.

### Basic Example

```go
package main

import (
    "context"
    "github.com/hlfshell/gogentic/agent"
)

type WeatherData struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Location    string  `json:"location"`
}

func main() {
    // Create a type-safe tool
    weatherTool := agent.NewTypedTool(
        "get_weather",
        "Gets the current weather for a location",
        map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "location": map[string]interface{}{
                    "type":        "string",
                    "description": "The city name",
                },
            },
            "required": []string{"location"},
        },
        func(ctx context.Context, args agent.Arguments) (WeatherData, error) {
            location := args["location"].(string)
            
            // Just return your custom type directly!
            return WeatherData{
                Temperature: 72.5,
                Condition:   "Sunny",
                Location:    location,
            }, nil
        },
    )
    
    // Use in agent config
    config := agent.AgentConfig{
        Tools: []agent.Tool{weatherTool},
        // ... other config
    }
}
```

### Benefits

- ✅ **Type-safe**: Return your custom types directly
- ✅ **No wrapping**: No need to manually call `NewToolResult`
- ✅ **Clean code**: Focus on business logic, not plumbing
- ✅ **Error handling**: Errors are automatically wrapped

## Method 2: go generate (Advanced)

For even cleaner code, use `go generate` to auto-generate tool wrappers.

### Step 1: Define Your Tool

```go
package mytools

import "context"

// Add the go:generate directive
//go:generate go run github.com/hlfshell/gogentic/agent/cmd/gentool -type=CalculatorTool

type CalculatorTool struct{}

// Define an Execute method with your parameters
func (c CalculatorTool) Execute(ctx context.Context, operation string, a, b float64) (float64, error) {
    switch operation {
    case "add":
        return a + b, nil
    case "subtract":
        return a - b, nil
    case "multiply":
        return a * b, nil
    case "divide":
        if b == 0 {
            return 0, fmt.Errorf("division by zero")
        }
        return a / b, nil
    default:
        return 0, fmt.Errorf("unknown operation: %s", operation)
    }
}
```

### Step 2: Run go generate

```bash
go generate ./...
```

This creates a `calculatortool_tool_gen.go` file with a wrapper function.

### Step 3: Use the Generated Tool

```go
// The generated code provides NewCalculatorTool
calcTool := NewCalculatorTool(
    "calculator",
    "Performs arithmetic operations",
    map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "operation": map[string]interface{}{"type": "string"},
            "a": map[string]interface{}{"type": "number"},
            "b": map[string]interface{}{"type": "number"},
        },
        "required": []string{"operation", "a", "b"},
    },
)
```

## Extracting Results

When consuming tool results, you have several options:

### Option 1: Direct Type Assertion (Fastest)

```go
result, _ := tool.Execute(ctx, args)
if weather, ok := result.GetResult().(WeatherData); ok {
    fmt.Printf("Temperature: %.1f\n", weather.Temperature)
}
```

### Option 2: Safe Type Assertion

```go
result, _ := tool.Execute(ctx, args)
switch v := result.GetResult().(type) {
case WeatherData:
    fmt.Printf("Temperature: %.1f\n", v.Temperature)
case string:
    fmt.Println("Got string:", v)
default:
    fmt.Println("Unknown type")
}
```

### Option 3: JSON Round-Trip (Safest)

```go
result, _ := tool.Execute(ctx, args)
jsonBytes, _ := result.ToJSON()

var weather WeatherData
json.Unmarshal(jsonBytes, &weather)
```

## Comparison with Manual Approach

### Before (Manual wrapping)

```go
tool := agent.NewFunctionTool(
    "get_weather",
    "Gets weather",
    params,
    func(ctx context.Context, args agent.Arguments) (agent.ResultInterface, error) {
        location := args["location"].(string)
        weather := fetchWeather(location)
        // Manual wrapping required
        return agent.NewToolResult("get_weather", weather), nil
    },
)
```

### After (NewTypedTool)

```go
tool := agent.NewTypedTool(
    "get_weather",
    "Gets weather",
    params,
    func(ctx context.Context, args agent.Arguments) (WeatherData, error) {
        location := args["location"].(string)
        return fetchWeather(location), nil  // Just return directly!
    },
)
```

## Best Practices

1. **Use `NewTypedTool` for most cases** - It's the sweet spot of simplicity and type safety

2. **Use `go generate` for complex tools** - When you have many similar tools or complex parameter extraction

3. **Keep handlers simple** - Extract arguments, call business logic, return results

4. **Handle errors properly** - Return errors from handlers, they'll be wrapped automatically

5. **Use meaningful types** - Define structs for complex return values instead of maps

6. **Document your tools** - Use clear descriptions and parameter schemas

## Examples

See `agent/tool_example_test.go` for complete working examples.

