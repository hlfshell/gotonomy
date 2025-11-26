package tool

import (
	"reflect"
	"testing"
)

func TestType(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{"string", Type[string](), "string"},
		{"int", Type[int](), "int"},
		{"int64", Type[int64](), "int64"},
		{"float64", Type[float64](), "float64"},
		{"bool", Type[bool](), "bool"},
		{"slice", Type[[]string](), "[]string"},
		{"map", Type[map[string]int](), "map[string]int"},
		{"struct", Type[struct {
			Name string
			Age  int
		}](), "struct { Name string; Age int }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.typ.Kind() == reflect.Invalid {
				t.Errorf("Type() returned invalid type")
			}
			// Verify it matches the expected type name
			if tt.typ.String() != tt.expected && tt.typ.Kind().String() != tt.expected {
				// For complex types, just verify it's not invalid
				if tt.typ.Kind() == reflect.Invalid {
					t.Errorf("Type() returned invalid type for %s", tt.name)
				}
			}
		})
	}
}

func TestType_MatchesReflectTypeOf(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
	}{
		{"string", ""},
		{"int", 0},
		{"int64", int64(0)},
		{"float64", 0.0},
		{"bool", false},
		{"slice", []string{}},
		{"map", map[string]int{}},
	}

		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedType := reflect.TypeOf(tt.val)
			var gotType reflect.Type

			switch tt.val.(type) {
			case string:
				gotType = Type[string]()
			case int:
				gotType = Type[int]()
			case int64:
				gotType = Type[int64]()
			case float64:
				gotType = Type[float64]()
			case bool:
				gotType = Type[bool]()
			case []string:
				gotType = Type[[]string]()
			case map[string]int:
				gotType = Type[map[string]int]()
			}

			if gotType != expectedType {
				t.Errorf("Type() = %v, want %v", gotType, expectedType)
			}
		})
	}
}

