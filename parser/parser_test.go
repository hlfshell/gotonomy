package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hlfshell/structured-parse/go/structuredparse"
)

// Test types for parsing
type Task struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type Person struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

// Test StructuredLabelsParser

func TestNewStructuredParse(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "name", Required: true},
		{Name: "age"},
	}

	convertFunc := func(m map[string]interface{}) (Person, error) {
		var p Person
		jsonBytes, _ := json.Marshal(m)
		return p, json.Unmarshal(jsonBytes, &p)
	}

	parser, err := NewStructuredParse(labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParse failed: %v", err)
	}

	if parser == nil {
		t.Fatal("parser is nil")
	}
	if parser.parser == nil {
		t.Fatal("internal parser is nil")
	}
	if parser.convertFunc == nil {
		t.Fatal("convertFunc is nil")
	}
}

func TestStructuredLabelsParser_Parse_Success(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Name", Required: true},
		{Name: "Age"},
		{Name: "Email"},
	}

	convertFunc := func(m map[string]interface{}) (Person, error) {
		p := Person{}
		if name, ok := m["Name"].(string); ok {
			p.Name = name
		}
		if age, ok := m["Age"].(string); ok {
			// Parse age as int from string
			var ageInt int
			_, err := fmt.Sscanf(age, "%d", &ageInt)
			if err == nil {
				p.Age = ageInt
			}
		}
		if email, ok := m["Email"].(string); ok {
			p.Email = email
		}
		return p, nil
	}

	parser, err := NewStructuredParse(labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParse failed: %v", err)
	}

	input := `
Name: John Doe
Age: 30
Email: john@example.com
`

	result, err := parser.Parse(input)
	if err != nil {
		// Check if it's just non-fatal errors
		var parseErr *ParseErrors
		if errors.As(err, &parseErr) {
			// Non-fatal errors are okay
		} else {
			t.Fatalf("Parse failed: %v", err)
		}
	}

	if result.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", result.Name)
	}
	if result.Age != 30 {
		t.Errorf("Expected age 30, got %d", result.Age)
	}
	if result.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", result.Email)
	}
}

func TestStructuredLabelsParser_Parse_WithNonFatalErrors(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Name", Required: true},
		{Name: "Age"},
	}

	convertFunc := func(m map[string]interface{}) (Person, error) {
		p := Person{}
		if name, ok := m["Name"].(string); ok {
			p.Name = name
		}
		if age, ok := m["Age"].(string); ok {
			var ageInt int
			_, err := fmt.Sscanf(age, "%d", &ageInt)
			if err == nil {
				p.Age = ageInt
			}
		}
		return p, nil
	}

	parser, err := NewStructuredParse(labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParse failed: %v", err)
	}

	input := `
Name: Jane Doe
Age: 25
`

	result, err := parser.Parse(input)
	// Should return result even with non-fatal errors
	if result.Name != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got '%s'", result.Name)
	}

	// Check if we got ParseErrors (non-fatal errors are okay)
	var parseErr *ParseErrors
	if errors.As(err, &parseErr) {
		// Non-fatal errors are expected and acceptable
	}
}

func TestStructuredLabelsParser_Parse_ConvertError(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Name", Required: true},
	}

	convertFunc := func(m map[string]interface{}) (Person, error) {
		return Person{}, errors.New("conversion failed")
	}

	parser, err := NewStructuredParse(labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParse failed: %v", err)
	}

	input := `Name: Test`

	_, err = parser.Parse(input)
	if err == nil {
		t.Fatal("Expected error from convertFunc, got nil")
	}

	if err.Error() != "failed to convert parsed data: conversion failed" {
		t.Errorf("Expected conversion error, got: %v", err)
	}
}

// Test StructuredLabelBlocksParser

func TestNewStructuredParseBlocks(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Task", IsBlockStart: true, Required: true},
		{Name: "Status"},
		{Name: "Result"},
	}

	convertFunc := func(m map[string]interface{}) (Task, error) {
		var task Task
		jsonBytes, _ := json.Marshal(m)
		return task, json.Unmarshal(jsonBytes, &task)
	}

	parser, err := NewStructuredParseBlocks[[]Task, Task](labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParseBlocks failed: %v", err)
	}

	if parser == nil {
		t.Fatal("parser is nil")
	}
	if parser.parser == nil {
		t.Fatal("internal parser is nil")
	}
	if parser.convertFunc == nil {
		t.Fatal("convertFunc is nil")
	}
}

func TestStructuredLabelBlocksParser_Parse_Success(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Task", IsBlockStart: true, Required: true},
		{Name: "Status"},
		{Name: "Result"},
	}

	convertFunc := func(m map[string]interface{}) (Task, error) {
		task := Task{}
		if name, ok := m["Task"].(string); ok {
			task.Name = name
		}
		if status, ok := m["Status"].(string); ok {
			task.Status = status
		}
		if result, ok := m["Result"].(string); ok {
			task.Result = result
		}
		return task, nil
	}

	parser, err := NewStructuredParseBlocks[[]Task, Task](labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParseBlocks failed: %v", err)
	}

	input := `
Task: Data Collection
Status: Complete
Result: Success

Task: Trend Analysis
Status: In Progress
Result: Pending
`

	result, err := parser.Parse(input)
	if err != nil {
		// Check if it's just non-fatal errors
		var parseErr *ParseErrors
		if !errors.As(err, &parseErr) {
			t.Fatalf("Parse failed with unexpected error: %v", err)
		}
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(result))
	}

	if result[0].Name != "Data Collection" {
		t.Errorf("Expected first task name 'Data Collection', got '%s'", result[0].Name)
	}
	if result[0].Status != "Complete" {
		t.Errorf("Expected first task status 'Complete', got '%s'", result[0].Status)
	}
	if result[0].Result != "Success" {
		t.Errorf("Expected first task result 'Success', got '%s'", result[0].Result)
	}

	if result[1].Name != "Trend Analysis" {
		t.Errorf("Expected second task name 'Trend Analysis', got '%s'", result[1].Name)
	}
	if result[1].Status != "In Progress" {
		t.Errorf("Expected second task status 'In Progress', got '%s'", result[1].Status)
	}
	if result[1].Result != "Pending" {
		t.Errorf("Expected second task result 'Pending', got '%s'", result[1].Result)
	}
}

func TestStructuredLabelBlocksParser_Parse_WithConversionErrors(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Task", IsBlockStart: true, Required: true},
		{Name: "Status"},
	}

	callCount := 0
	convertFunc := func(m map[string]interface{}) (Task, error) {
		callCount++
		// Fail conversion for first block, succeed for second
		if callCount == 1 {
			return Task{}, errors.New("conversion error")
		}
		task := Task{}
		if name, ok := m["Task"].(string); ok {
			task.Name = name
		}
		if status, ok := m["Status"].(string); ok {
			task.Status = status
		}
		return task, nil
	}

	parser, err := NewStructuredParseBlocks[[]Task, Task](labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParseBlocks failed: %v", err)
	}

	input := `
Task: First Task
Status: Complete

Task: Second Task
Status: In Progress
`

	result, err := parser.Parse(input)
	// Should still return results for successfully converted blocks
	if len(result) != 1 {
		t.Errorf("Expected 1 successfully converted task, got %d", len(result))
	}

	if result[0].Name != "Second Task" {
		t.Errorf("Expected task name 'Second Task', got '%s'", result[0].Name)
	}

	// Should have errors
	var parseErr *ParseErrors
	if !errors.As(err, &parseErr) {
		t.Fatal("Expected ParseErrors, got different error type")
	}

	if len(parseErr.AllErrors()) == 0 {
		t.Error("Expected conversion errors, got none")
	}
}

func TestStructuredLabelBlocksParser_ImplementsParserInterface(t *testing.T) {
	labels := []structuredparse.Label{
		{Name: "Task", IsBlockStart: true, Required: true},
		{Name: "Status"},
	}

	convertFunc := func(m map[string]interface{}) (Task, error) {
		task := Task{}
		if name, ok := m["Task"].(string); ok {
			task.Name = name
		}
		if status, ok := m["Status"].(string); ok {
			task.Status = status
		}
		return task, nil
	}

	parser, err := NewStructuredParseBlocks[[]Task, Task](labels, convertFunc)
	if err != nil {
		t.Fatalf("NewStructuredParseBlocks failed: %v", err)
	}

	// Test that it implements Parser[[]Task]
	var _ Parser[[]Task] = parser

	input := `
Task: Test Task
Status: Complete
`

	result, err := parser.Parse(input)
	if err != nil {
		var parseErr *ParseErrors
		if !errors.As(err, &parseErr) {
			t.Fatalf("Parse failed: %v", err)
		}
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 task, got %d", len(result))
	}
}

// Test ParseErrors

func TestParseErrors_Error(t *testing.T) {
	err := &ParseErrors{
		Errors: []string{"error1", "error2"},
		Result: map[string]interface{}{"key": "value"},
	}

	errStr := err.Error()
	if errStr != "error1" {
		t.Errorf("Expected first error 'error1', got '%s'", errStr)
	}
}

func TestParseErrors_AllErrors(t *testing.T) {
	err := &ParseErrors{
		Errors: []string{"error1", "error2", "error3"},
		Result: nil,
	}

	allErrors := err.AllErrors()
	if len(allErrors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(allErrors))
	}

	if allErrors[0] != "error1" {
		t.Errorf("Expected first error 'error1', got '%s'", allErrors[0])
	}
}

func TestParseErrors_EmptyErrors(t *testing.T) {
	err := &ParseErrors{
		Errors: []string{},
		Result: nil,
	}

	errStr := err.Error()
	if errStr != "parse errors" {
		t.Errorf("Expected 'parse errors' for empty errors, got '%s'", errStr)
	}
}

// Test JSONParser

func TestJSONParser_Parse(t *testing.T) {
	parser := NewJSONParser()

	input := `{"name": "John", "age": 30, "email": "john@example.com"}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("Expected name 'John', got '%v'", result["name"])
	}
	if result["age"] != float64(30) { // JSON numbers are float64
		t.Errorf("Expected age 30, got %v", result["age"])
	}
}

func TestJSONParser_Parse_InvalidJSON(t *testing.T) {
	parser := NewJSONParser()

	input := `{invalid json}`
	_, err := parser.Parse(input)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

// Test JSONTypedParser

func TestNewJSONTypedParser(t *testing.T) {
	parser := NewJSONTypedParser[Person]()
	if parser == nil {
		t.Fatal("parser is nil")
	}
}

func TestJSONTypedParser_Parse_Success(t *testing.T) {
	parser := NewJSONTypedParser[Person]()

	input := `{"name": "John Doe", "age": 30, "email": "john@example.com"}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", result.Name)
	}
	if result.Age != 30 {
		t.Errorf("Expected age 30, got %d", result.Age)
	}
	if result.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", result.Email)
	}
}

func TestJSONTypedParser_Parse_WithTask(t *testing.T) {
	parser := NewJSONTypedParser[Task]()

	input := `{"name": "Test Task", "status": "Complete", "result": "Success"}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Name != "Test Task" {
		t.Errorf("Expected name 'Test Task', got '%s'", result.Name)
	}
	if result.Status != "Complete" {
		t.Errorf("Expected status 'Complete', got '%s'", result.Status)
	}
	if result.Result != "Success" {
		t.Errorf("Expected result 'Success', got '%s'", result.Result)
	}
}

func TestJSONTypedParser_Parse_InvalidJSON(t *testing.T) {
	parser := NewJSONTypedParser[Person]()

	input := `{invalid json}`
	_, err := parser.Parse(input)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestJSONTypedParser_Parse_EmptyJSON(t *testing.T) {
	parser := NewJSONTypedParser[Person]()

	input := `{}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should return zero value Person
	if result.Name != "" {
		t.Errorf("Expected empty name, got '%s'", result.Name)
	}
	if result.Age != 0 {
		t.Errorf("Expected age 0, got %d", result.Age)
	}
}

func TestJSONTypedParser_Parse_PartialFields(t *testing.T) {
	parser := NewJSONTypedParser[Person]()

	input := `{"name": "Jane Doe"}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Name != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got '%s'", result.Name)
	}
	if result.Age != 0 {
		t.Errorf("Expected age 0, got %d", result.Age)
	}
	if result.Email != "" {
		t.Errorf("Expected empty email, got '%s'", result.Email)
	}
}

func TestJSONTypedParser_ImplementsParserInterface(t *testing.T) {
	parser := NewJSONTypedParser[Person]()

	// Test that it implements Parser[Person]
	var _ Parser[Person] = parser

	input := `{"name": "Test", "age": 25, "email": "test@example.com"}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Name != "Test" {
		t.Errorf("Expected name 'Test', got '%s'", result.Name)
	}
}

func TestJSONTypedParser_Parse_WithSlice(t *testing.T) {
	parser := NewJSONTypedParser[[]Task]()

	input := `[{"name": "Task 1", "status": "Complete", "result": "Success"}, {"name": "Task 2", "status": "In Progress", "result": "Pending"}]`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(result))
	}

	if result[0].Name != "Task 1" {
		t.Errorf("Expected first task name 'Task 1', got '%s'", result[0].Name)
	}
	if result[1].Name != "Task 2" {
		t.Errorf("Expected second task name 'Task 2', got '%s'", result[1].Name)
	}
}

func TestJSONTypedParser_Parse_WithNestedStructure(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
		Zip    string `json:"zip"`
	}
	type PersonWithAddress struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Address Address `json:"address"`
	}

	parser := NewJSONTypedParser[PersonWithAddress]()

	input := `{"name": "John", "age": 30, "address": {"street": "123 Main St", "city": "Springfield", "zip": "12345"}}`
	result, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Name != "John" {
		t.Errorf("Expected name 'John', got '%s'", result.Name)
	}
	if result.Address.Street != "123 Main St" {
		t.Errorf("Expected street '123 Main St', got '%s'", result.Address.Street)
	}
	if result.Address.City != "Springfield" {
		t.Errorf("Expected city 'Springfield', got '%s'", result.Address.City)
	}
}
