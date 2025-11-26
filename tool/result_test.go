package tool

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestResult_Errored(t *testing.T) {
	tests := []struct {
		name   string
		result Result[string]
		want   bool
	}{
		{
			name:   "result with error",
			result: Result[string]{Result: "test", Error: errors.New("test error")},
			want:   true,
		},
		{
			name:   "result without error",
			result: Result[string]{Result: "test", Error: nil},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Errored(); got != tt.want {
				t.Errorf("Errored() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_GetError(t *testing.T) {
	testError := errors.New("test error")
	result := Result[string]{Result: "test", Error: testError}

	if got := result.GetError(); got != testError {
		t.Errorf("GetError() = %v, want %v", got, testError)
	}

	noErrorResult := Result[string]{Result: "test", Error: nil}
	if got := noErrorResult.GetError(); got != nil {
		t.Errorf("GetError() = %v, want nil", got)
	}
}

func TestResult_GetResult(t *testing.T) {
	tests := []struct {
		name   string
		result Result[interface{}]
		want   interface{}
	}{
		{
			name:   "string result",
			result: Result[interface{}]{Result: "test", Error: nil},
			want:   "test",
		},
		{
			name:   "int result",
			result: Result[interface{}]{Result: 42, Error: nil},
			want:   42,
		},
		{
			name:   "map result",
			result: Result[interface{}]{Result: map[string]int{"key": 1}, Error: nil},
			want:   map[string]int{"key": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.GetResult(); !equal(got, tt.want) {
				t.Errorf("GetResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_ToJSON(t *testing.T) {
	tests := []struct {
		name      string
		result    Result[interface{}]
		wantError bool
	}{
		{
			name:      "successful string result",
			result:    Result[interface{}]{Result: "test", Error: nil},
			wantError: false,
		},
		{
			name:      "successful int result",
			result:    Result[interface{}]{Result: 42, Error: nil},
			wantError: false,
		},
		{
			name:      "successful map result",
			result:    Result[interface{}]{Result: map[string]int{"key": 1}, Error: nil},
			wantError: false,
		},
		{
			name:      "result with error",
			result:    Result[interface{}]{Result: nil, Error: errors.New("test error")},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.result.ToJSON()
			if (err != nil) != tt.wantError {
				t.Errorf("ToJSON() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Verify it's valid JSON
				var decoded interface{}
				if err := json.Unmarshal(got, &decoded); err != nil {
					t.Errorf("ToJSON() returned invalid JSON: %v", err)
				}
			}
		})
	}
}

func TestResult_String(t *testing.T) {
	tests := []struct {
		name      string
		result    Result[interface{}]
		wantError bool
		validate  func(string) bool
	}{
		{
			name:      "string result",
			result:    Result[interface{}]{Result: "test", Error: nil},
			wantError: false,
			validate:  func(s string) bool { return s == `"test"` },
		},
		{
			name:      "int result",
			result:    Result[interface{}]{Result: 42, Error: nil},
			wantError: false,
			validate:  func(s string) bool { return s == "42" },
		},
		{
			name:      "float result",
			result:    Result[interface{}]{Result: 3.14, Error: nil},
			wantError: false,
			validate:  func(s string) bool { return s == "3.14" },
		},
		{
			name:      "bool result",
			result:    Result[interface{}]{Result: true, Error: nil},
			wantError: false,
			validate:  func(s string) bool { return s == "true" },
		},
		{
			name:      "map result",
			result:    Result[interface{}]{Result: map[string]int{"key": 1}, Error: nil},
			wantError: false,
			validate:  func(s string) bool { return len(s) > 0 && s[0] == '{' },
		},
		{
			name:      "slice result",
			result:    Result[interface{}]{Result: []int{1, 2, 3}, Error: nil},
			wantError: false,
			validate:  func(s string) bool { return len(s) > 0 && s[0] == '[' },
		},
		{
			name:      "result with error",
			result:    Result[interface{}]{Result: nil, Error: errors.New("test error")},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.result.String()
			if (err != nil) != tt.wantError {
				t.Errorf("String() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && tt.validate != nil {
				if !tt.validate(got) {
					t.Errorf("String() = %v, validation failed", got)
				}
			}
		})
	}
}

func TestResult_MarshalJSON(t *testing.T) {
	result := Result[string]{Result: "test", Error: nil}
	got, err := result.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}

	var decoded string
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Errorf("MarshalJSON() returned invalid JSON: %v", err)
		return
	}

	if decoded != "test" {
		t.Errorf("MarshalJSON() decoded = %v, want %v", decoded, "test")
	}
}

func TestResult_UnmarshalJSON(t *testing.T) {
	data := []byte(`"test"`)
	result := &Result[string]{}
	if err := result.UnmarshalJSON(data); err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}

	if result.Result != "test" {
		t.Errorf("UnmarshalJSON() Result = %v, want %v", result.Result, "test")
	}
}

func TestNewOK(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{}
		check  func(ResultInterface) bool
	}{
		{
			name:  "string value",
			value: "test",
			check: func(r ResultInterface) bool {
				return !r.Errored() && r.GetResult() == "test"
			},
		},
		{
			name:  "int value",
			value: 42,
			check: func(r ResultInterface) bool {
				return !r.Errored() && r.GetResult() == 42
			},
		},
		{
			name:  "map value",
			value: map[string]int{"key": 1},
			check: func(r ResultInterface) bool {
				return !r.Errored() && r.GetResult() != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ResultInterface
			switch v := tt.value.(type) {
			case string:
				result = NewOK(v)
			case int:
				result = NewOK(v)
			case map[string]int:
				result = NewOK(v)
			}

			if !tt.check(result) {
				t.Errorf("NewOK() result validation failed")
			}
		})
	}
}

func TestNewError(t *testing.T) {
	testError := errors.New("test error")
	result := NewError(testError)

	if !result.Errored() {
		t.Errorf("NewError() Errored() = false, want true")
	}

	if result.GetError() != testError {
		t.Errorf("NewError() GetError() = %v, want %v", result.GetError(), testError)
	}

	if result.GetResult() != nil {
		t.Errorf("NewError() GetResult() = %v, want nil", result.GetResult())
	}
}

// Helper function to compare values
func equal(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// For maps, do a simple comparison
	if ma, ok := a.(map[string]int); ok {
		if mb, ok := b.(map[string]int); ok {
			if len(ma) != len(mb) {
				return false
			}
			for k, v := range ma {
				if mb[k] != v {
					return false
				}
			}
			return true
		}
	}

	return a == b
}

