package agent_test

import (
	"context"
	"fmt"

	"github.com/hlfshell/gogentic/agent"
)

// Example 1: Using NewTypedTool for simple type-safe tools
// This is the recommended approach for most use cases.

type WeatherData struct {
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	Location    string  `json:"location"`
}

func ExampleNewTypedTool() {
	// Create a type-safe tool - no manual Result wrapping needed!
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
			// Extract arguments
			location, ok := args["location"].(string)
			if !ok {
				return WeatherData{}, fmt.Errorf("location must be a string")
			}

			// Return your custom type directly - no wrapping needed!
			return WeatherData{
				Temperature: 72.5,
				Condition:   "Sunny",
				Humidity:    45,
				Location:    location,
			}, nil
		},
	)

	// Use the tool
	result := weatherTool.Execute(context.Background(), agent.Arguments{
		"location": "San Francisco",
	})

	// Extract the result
	if weather, ok := result.GetResult().(WeatherData); ok {
		fmt.Printf("Temperature: %.1f°F\n", weather.Temperature)
		fmt.Printf("Condition: %s\n", weather.Condition)
	}
}

// Example 2: Using go generate for even cleaner tool definitions
// This approach generates the boilerplate automatically.
//
// Step 1: Define your tool struct with an Execute method
// Step 2: Add a go:generate comment
// Step 3: Run: go generate ./...
//
// Note: This example shows the pattern, but won't actually generate
// until the gentool command is built and available.

//go:generate go run github.com/hlfshell/gogentic/agent/cmd/gentool -type=CalculatorTool
type CalculatorTool struct{}

// Execute performs a calculation
// The gentool will parse this signature and create a type-safe wrapper
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

// After running go generate, you can use:
// tool := NewCalculatorTool("calculator", "Performs calculations", parameters)

// Example 3: Extracting results from tools

func ExampleExtractingResults() {
	// Create and execute a tool
	weatherTool := agent.NewTypedTool(
		"get_weather",
		"Gets weather",
		nil,
		func(ctx context.Context, args agent.Arguments) (WeatherData, error) {
			return WeatherData{Temperature: 72.5, Condition: "Sunny"}, nil
		},
	)

	result := weatherTool.Execute(context.Background(), agent.Arguments{})

	// Method 1: Direct type assertion (fastest)
	if weather, ok := result.GetResult().(WeatherData); ok {
		fmt.Printf("Temp: %.1f\n", weather.Temperature)
	}

	// Method 2: JSON round-trip (safest for complex types)
	// This is useful when passing results between different parts of your system
	jsonBytes, _ := result.ToJSON()
	// var weather WeatherData
	// json.Unmarshal(jsonBytes, &weather)
	fmt.Printf("JSON: %s\n", string(jsonBytes))

	// Method 3: String representation (for logging/display)
	fmt.Printf("String: %s\n", result.String())
}

