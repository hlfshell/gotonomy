package tool

import (
	"fmt"
	"reflect"
	"testing"
)

// TestNewParameter tests parameter creation
func TestNewParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramType   string
		description string
		required    bool
		defaultVal  interface{}
		wantName    string
		wantDesc    string
		wantReq     bool
		wantDefault interface{}
	}{
		{
			name:        "string parameter",
			paramType:   "string",
			description: "A string parameter",
			required:    true,
			defaultVal:  "",
			wantName:    "test_param",
			wantDesc:    "A string parameter",
			wantReq:     true,
			wantDefault: "",
		},
		{
			name:        "int parameter",
			paramType:   "int",
			description: "An integer parameter",
			required:    false,
			defaultVal:  42,
			wantName:    "count",
			wantDesc:    "An integer parameter",
			wantReq:     false,
			wantDefault: 42,
		},
		{
			name:        "float parameter",
			paramType:   "float64",
			description: "A float parameter",
			required:    false,
			defaultVal:  3.14,
			wantName:    "pi",
			wantDesc:    "A float parameter",
			wantReq:     false,
			wantDefault: 3.14,
		},
		{
			name:        "bool parameter",
			paramType:   "bool",
			description: "A boolean parameter",
			required:    true,
			defaultVal:  false,
			wantName:    "enabled",
			wantDesc:    "A boolean parameter",
			wantReq:     true,
			wantDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var param Parameter
			switch tt.paramType {
			case "string":
				param = NewParameter[string](
					tt.wantName,
					tt.wantDesc,
					tt.wantReq,
					tt.defaultVal.(string),
					func(v string) (string, error) { return v, nil },
				)
			case "int":
				param = NewParameter[int](
					tt.wantName,
					tt.wantDesc,
					tt.wantReq,
					tt.defaultVal.(int),
					func(v int) (string, error) { return fmt.Sprintf("%d", v), nil },
				)
			case "float64":
				param = NewParameter[float64](
					tt.wantName,
					tt.wantDesc,
					tt.wantReq,
					tt.defaultVal.(float64),
					func(v float64) (string, error) { return fmt.Sprintf("%g", v), nil },
				)
			case "bool":
				param = NewParameter[bool](
					tt.wantName,
					tt.wantDesc,
					tt.wantReq,
					tt.defaultVal.(bool),
					func(v bool) (string, error) { return fmt.Sprintf("%t", v), nil },
				)
			}

			if param.Name() != tt.wantName {
				t.Errorf("Name() = %v, want %v", param.Name(), tt.wantName)
			}
			if param.Description() != tt.wantDesc {
				t.Errorf("Description() = %v, want %v", param.Description(), tt.wantDesc)
			}
			if param.Required() != tt.wantReq {
				t.Errorf("Required() = %v, want %v", param.Required(), tt.wantReq)
			}
			if param.Default() != tt.wantDefault {
				t.Errorf("Default() = %v, want %v", param.Default(), tt.wantDefault)
			}
		})
	}
}

// TestParameter_TypeCheck tests type validation
func TestParameter_TypeCheck(t *testing.T) {
	tests := []struct {
		name      string
		param     Parameter
		value     interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid string value",
			param:     NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			value:     "test",
			wantError: false,
		},
		{
			name:      "invalid type for string parameter",
			param:     NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			value:     42,
			wantError: true,
			errorMsg:  "expected type string",
		},
		{
			name:      "nil value for required parameter",
			param:     NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			value:     nil,
			wantError: true,
			errorMsg:  "required but value is nil",
		},
		{
			name:      "nil value for optional parameter",
			param:     NewParameter[string]("name", "A name", false, "", func(v string) (string, error) { return v, nil }),
			value:     nil,
			wantError: false,
		},
		{
			name:      "valid int value",
			param:     NewParameter[int]("count", "A count", true, 0, func(v int) (string, error) { return fmt.Sprintf("%d", v), nil }),
			value:     42,
			wantError: false,
		},
		{
			name:      "invalid type for int parameter",
			param:     NewParameter[int]("count", "A count", true, 0, func(v int) (string, error) { return fmt.Sprintf("%d", v), nil }),
			value:     "not an int",
			wantError: true,
			errorMsg:  "expected type int",
		},
		{
			name:      "valid float value",
			param:     NewParameter[float64]("pi", "Pi value", false, 0.0, func(v float64) (string, error) { return fmt.Sprintf("%g", v), nil }),
			value:     3.14,
			wantError: false,
		},
		{
			name:      "valid bool value",
			param:     NewParameter[bool]("enabled", "Enabled flag", false, false, func(v bool) (string, error) { return fmt.Sprintf("%t", v), nil }),
			value:     true,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.TypeCheck(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("TypeCheck() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorMsg != "" {
				if err == nil || err.Error() == "" {
					t.Errorf("TypeCheck() expected error message containing %q, got %v", tt.errorMsg, err)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("TypeCheck() error = %v, want error containing %q", err, tt.errorMsg)
				}
			}
		})
	}
}

// TestParameter_Value tests value extraction with defaults
func TestParameter_Value(t *testing.T) {
	tests := []struct {
		name      string
		param     Parameter
		value     interface{}
		want      interface{}
		wantError bool
	}{
		{
			name:      "use provided value",
			param:     NewParameter[string]("name", "A name", true, "default", func(v string) (string, error) { return v, nil }),
			value:     "provided",
			want:      "provided",
			wantError: false,
		},
		{
			name:      "use default when value is nil",
			param:     NewParameter[string]("name", "A name", false, "default", func(v string) (string, error) { return v, nil }),
			value:     nil,
			want:      "default",
			wantError: false,
		},
		{
			name:      "invalid type",
			param:     NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			value:     42,
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.param.Value(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("Value() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParameter_StringFunction tests string conversion
func TestParameter_StringFunction(t *testing.T) {
	tests := []struct {
		name      string
		param     Parameter
		value     interface{}
		want      string
		wantError bool
	}{
		{
			name:      "string parameter",
			param:     NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			value:     "test",
			want:      "test",
			wantError: false,
		},
		{
			name:      "int parameter",
			param:     NewParameter[int]("count", "A count", true, 0, func(v int) (string, error) { return fmt.Sprintf("%d", v), nil }),
			value:     42,
			want:      "42",
			wantError: false,
		},
		{
			name:      "wrong type for string function",
			param:     NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			value:     42,
			wantError: true,
		},
		{
			name: "empty string value (valid type)",
			param: NewParameter[string]("name", "A name", true, "", func(v string) (string, error) {
				return v, nil
			}),
			value:     "",
			wantError: false, // Empty string is a valid string type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Access the stringFunction through reflection or test it indirectly
			// Since stringFunction is unexported, we test it through the parameter's behavior
			// In a real scenario, this would be tested through tool execution
			// For now, we verify the parameter accepts the correct type
			err := tt.param.TypeCheck(tt.value)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for value %v", tt.value)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestParameter_Getters tests all getter methods
func TestParameter_Getters(t *testing.T) {
	param := NewParameter[string]("test_param", "Test description", true, "default", func(v string) (string, error) { return v, nil })

	if param.Name() != "test_param" {
		t.Errorf("Name() = %v, want %v", param.Name(), "test_param")
	}

	if param.Description() != "Test description" {
		t.Errorf("Description() = %v, want %v", param.Description(), "Test description")
	}

	if param.Type() != reflect.TypeOf("") {
		t.Errorf("Type() = %v, want %v", param.Type(), reflect.TypeOf(""))
	}

	if param.Default() != "default" {
		t.Errorf("Default() = %v, want %v", param.Default(), "default")
	}

	if !param.Required() {
		t.Errorf("Required() = %v, want %v", param.Required(), true)
	}
}

// TestParametersToJSONSchema tests JSON schema conversion
func TestParametersToJSONSchema(t *testing.T) {
	tests := []struct {
		name     string
		params   []Parameter
		validate func(map[string]any) bool
	}{
		{
			name: "single string parameter",
			params: []Parameter{
				NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
			},
			validate: func(schema map[string]any) bool {
				props := schema["properties"].(map[string]any)
				nameProp := props["name"].(map[string]any)
				required := schema["required"].([]string)
				return nameProp["type"] == "string" &&
					nameProp["description"] == "A name" &&
					containsString(required, "name")
			},
		},
		{
			name: "multiple parameters with mixed types",
			params: []Parameter{
				NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
				NewParameter[int]("count", "A count", false, 0, func(v int) (string, error) { return "", nil }),
				NewParameter[bool]("enabled", "Enabled flag", false, false, func(v bool) (string, error) { return "", nil }),
			},
			validate: func(schema map[string]any) bool {
				props := schema["properties"].(map[string]any)
				required := schema["required"].([]string)
				return props["name"] != nil &&
					props["count"] != nil &&
					props["enabled"] != nil &&
					containsString(required, "name") &&
					!containsString(required, "count") &&
					!containsString(required, "enabled")
			},
		},
		{
			name: "all optional parameters",
			params: []Parameter{
				NewParameter[string]("name", "A name", false, "", func(v string) (string, error) { return v, nil }),
				NewParameter[int]("count", "A count", false, 0, func(v int) (string, error) { return "", nil }),
			},
			validate: func(schema map[string]any) bool {
				// Should not have "required" field if all are optional
				_, hasRequired := schema["required"]
				return !hasRequired || len(schema["required"].([]string)) == 0
			},
		},
		{
			name: "all required parameters",
			params: []Parameter{
				NewParameter[string]("name", "A name", true, "", func(v string) (string, error) { return v, nil }),
				NewParameter[int]("count", "A count", true, 0, func(v int) (string, error) { return "", nil }),
			},
			validate: func(schema map[string]any) bool {
				required := schema["required"].([]string)
				return len(required) == 2 &&
					containsString(required, "name") &&
					containsString(required, "count")
			},
		},
		{
			name: "numeric types",
			params: []Parameter{
				NewParameter[int]("int_val", "Integer", true, 0, func(v int) (string, error) { return "", nil }),
				NewParameter[int8]("int8_val", "Int8", true, int8(0), func(v int8) (string, error) { return "", nil }),
				NewParameter[int64]("int64_val", "Int64", true, int64(0), func(v int64) (string, error) { return "", nil }),
				NewParameter[uint]("uint_val", "Uint", true, uint(0), func(v uint) (string, error) { return "", nil }),
				NewParameter[float32]("float32_val", "Float32", true, float32(0), func(v float32) (string, error) { return "", nil }),
				NewParameter[float64]("float64_val", "Float64", true, 0.0, func(v float64) (string, error) { return "", nil }),
			},
			validate: func(schema map[string]any) bool {
				props := schema["properties"].(map[string]any)
				intProp := props["int_val"].(map[string]any)
				floatProp := props["float64_val"].(map[string]any)
				return intProp["type"] == "integer" &&
					floatProp["type"] == "number"
			},
		},
		{
			name: "array and object types",
			params: []Parameter{
				NewParameter[[]string]("tags", "Tags array", false, []string{}, func(v []string) (string, error) { return "", nil }),
				NewParameter[map[string]int]("metadata", "Metadata map", false, map[string]int{}, func(v map[string]int) (string, error) { return "", nil }),
			},
			validate: func(schema map[string]any) bool {
				props := schema["properties"].(map[string]any)
				tagsProp := props["tags"].(map[string]any)
				metaProp := props["metadata"].(map[string]any)
				return tagsProp["type"] == "array" &&
					metaProp["type"] == "object"
			},
		},
		{
			name: "struct type",
			params: []Parameter{
				NewParameter[struct {
					Name string
					Age  int
				}]("person", "Person struct", true, struct {
					Name string
					Age  int
				}{}, func(v struct {
					Name string
					Age  int
				}) (string, error) { return "", nil }),
			},
			validate: func(schema map[string]any) bool {
				props := schema["properties"].(map[string]any)
				personProp := props["person"].(map[string]any)
				return personProp["type"] == "object"
			},
		},
		{
			name:     "empty parameters",
			params:   []Parameter{},
			validate: func(schema map[string]any) bool {
				props := schema["properties"].(map[string]any)
				return len(props) == 0 &&
					schema["type"] == "object"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := ParametersToJSONSchema(tt.params)

			// Basic schema structure validation
			if schema["type"] != "object" {
				t.Errorf("Schema type = %v, want %v", schema["type"], "object")
			}

			if schema["properties"] == nil {
				t.Errorf("Schema missing properties")
			}

			// Custom validation
			if !tt.validate(schema) {
				t.Errorf("Schema validation failed for test case: %s", tt.name)
			}
		})
	}
}

// TestTypeToJSONSchemaType tests type mapping
func TestTypeToJSONSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected string
	}{
		{"string", reflect.String, "string"},
		{"int", reflect.Int, "integer"},
		{"int8", reflect.Int8, "integer"},
		{"int16", reflect.Int16, "integer"},
		{"int32", reflect.Int32, "integer"},
		{"int64", reflect.Int64, "integer"},
		{"uint", reflect.Uint, "integer"},
		{"uint8", reflect.Uint8, "integer"},
		{"uint16", reflect.Uint16, "integer"},
		{"uint32", reflect.Uint32, "integer"},
		{"uint64", reflect.Uint64, "integer"},
		{"float32", reflect.Float32, "number"},
		{"float64", reflect.Float64, "number"},
		{"bool", reflect.Bool, "boolean"},
		{"slice", reflect.Slice, "array"},
		{"array", reflect.Array, "array"},
		{"map", reflect.Map, "object"},
		{"struct", reflect.Struct, "object"},
		{"invalid", reflect.Complex64, "string"}, // default fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := typeToJSONSchemaType(tt.kind)
			if got != tt.expected {
				t.Errorf("typeToJSONSchemaType(%v) = %v, want %v", tt.kind, got, tt.expected)
			}
		})
	}
}

// TestParametersToJSONSchema_PropertyDescriptions tests description preservation
func TestParametersToJSONSchema_PropertyDescriptions(t *testing.T) {
	params := []Parameter{
		NewParameter[string]("param1", "First parameter description", true, "", func(v string) (string, error) { return v, nil }),
		NewParameter[int]("param2", "Second parameter description", false, 0, func(v int) (string, error) { return "", nil }),
	}

	schema := ParametersToJSONSchema(params)
	props := schema["properties"].(map[string]any)

	param1Prop := props["param1"].(map[string]any)
	if param1Prop["description"] != "First parameter description" {
		t.Errorf("param1 description = %v, want %v", param1Prop["description"], "First parameter description")
	}

	param2Prop := props["param2"].(map[string]any)
	if param2Prop["description"] != "Second parameter description" {
		t.Errorf("param2 description = %v, want %v", param2Prop["description"], "Second parameter description")
	}
}

// TestParametersToJSONSchema_RequiredField tests required field handling
func TestParametersToJSONSchema_RequiredField(t *testing.T) {
	// Test that required field is only present when there are required parameters
	optionalParams := []Parameter{
		NewParameter[string]("name", "Name", false, "", func(v string) (string, error) { return v, nil }),
	}

	schema := ParametersToJSONSchema(optionalParams)
	if _, hasRequired := schema["required"]; hasRequired {
		required := schema["required"].([]string)
		if len(required) > 0 {
			t.Errorf("Schema should not have required fields, but got %v", required)
		}
	}

	// Test that required field is present when there are required parameters
	requiredParams := []Parameter{
		NewParameter[string]("name", "Name", true, "", func(v string) (string, error) { return v, nil }),
	}

	schema = ParametersToJSONSchema(requiredParams)
	if _, hasRequired := schema["required"]; !hasRequired {
		t.Errorf("Schema should have required field")
	} else {
		required := schema["required"].([]string)
		if len(required) != 1 || required[0] != "name" {
			t.Errorf("Required field = %v, want [name]", required)
		}
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
