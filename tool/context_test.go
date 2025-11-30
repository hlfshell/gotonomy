package tool

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hlfshell/gotonomy/data/ledger"
)

func TestNode_ID(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	id := node.ID()
	if id == "" {
		t.Fatal("Node ID should not be empty")
	}

	// ID should be consistent
	if node.ID() != id {
		t.Fatal("ID() should return the same value")
	}
}

func TestNode_SetOutput(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	// Set output
	result := NewOK("test output")
	node.SetOutput(result)

	// Verify output was set
	if node.output == nil {
		t.Fatal("Output should not be nil after SetOutput")
	}

	if node.output != result {
		t.Fatal("Output should match the set value")
	}

	// Set nil output
	node.SetOutput(nil)
	if node.output != nil {
		t.Fatal("Output should be nil after setting nil")
	}
}



func TestNode_Data(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	nodeData := node.Data()
	if nodeData == nil {
		t.Fatal("Node Data() should not be nil")
	}

	globalData := node.GlobalData()
	if globalData == nil {
		t.Fatal("Node GlobalData() should not be nil")
	}

	// Verify nodeData and globalData are different
	if nodeData == globalData {
		t.Fatal("NodeData and GlobalData should be different scoped ledgers")
	}
}

func TestNode_ScopedData(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	// Get scoped data
	scope1 := "test-scope-1"
	scoped1 := node.ScopedData(scope1)
	if scoped1 == nil {
		t.Fatal("ScopedData should not return nil")
	}

	// Getting same scope should return same instance
	scoped1Again := node.ScopedData(scope1)
	if scoped1 != scoped1Again {
		t.Fatal("ScopedData should return the same instance for the same scope")
	}

	// Different scope should return different instance
	scope2 := "test-scope-2"
	scoped2 := node.ScopedData(scope2)
	if scoped2 == nil {
		t.Fatal("ScopedData should not return nil")
	}

	if scoped1 == scoped2 {
		t.Fatal("Different scopes should return different instances")
	}
}

func TestNode_ScopedData_Concurrent(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	// Concurrent access to ScopedData
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			scope := "scope"
			scoped := node.ScopedData(scope)
			if scoped == nil {
				t.Errorf("ScopedData should not return nil")
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All should get the same instance
	scoped := node.ScopedData("scope")
	if scoped == nil {
		t.Fatal("ScopedData should not return nil")
	}
}

func TestNode_Stats(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	stats := node.Stats()
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	// Stats should have startTime set after MarkStarted
	stats.MarkStarted()
	if stats.StartTime().IsZero() {
		t.Fatal("Stats startTime should be set")
	}

	// Mark finished
	stats.MarkFinished()
	if stats.EndTime().IsZero() {
		t.Fatal("Stats endTime should be set after MarkFinished")
	}

	duration := stats.ExecutionDuration()
	if duration <= 0 {
		t.Errorf("Duration should be positive, got %v", duration)
	}

	// Test value metrics
	stats.Set("test_metric", "value1")
	value := stats.Get("test_metric")
	if value == nil || value != "value1" {
		t.Errorf("Expected value metric 'value1', got %v", value)
	}

	// Test getting non-existent metric
	if stats.Get("nonexistent") != nil {
		t.Error("Get should return nil for non-existent metric")
	}
}

func TestNode_CreateChild(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Create child using Execution.createChild
	childTool := newMockTool("child-tool")
	child := e.createChild(root.ID(), childTool, Arguments{"test": "data"})
	if child == nil {
		t.Fatal("Child should not be nil")
	}

	if child.parent != root.ID() {
		t.Errorf("Expected parent %s, got %s", root.ID(), child.parent)
	}

	// Test error case: invalid parent ID
	invalidChild := e.createChild(ContextID("nonexistent"), childTool, Arguments{})
	if invalidChild != nil {
		t.Fatal("createChild should return nil for invalid parent")
	}
}

func TestNode_MarshalJSON(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{"input": "value"})

	// Marshal node
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Marshaled data should not be empty")
	}

	// Verify it's valid JSON
	var unmarshaled struct {
		ID             ContextID      `json:"id"`
		Parent         ContextID      `json:"parent"`
		Children       []ContextID    `json:"children"`
		ExecutionStats Stats          `json:"execution_stats"`
		Data           *ledger.Ledger `json:"data"`
	}
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ID != node.ID() {
		t.Errorf("Expected ID %s, got %s", node.ID(), unmarshaled.ID)
	}
}

func TestNode_UnmarshalJSON(t *testing.T) {
	// Create a node and marshal it
	rootTool := newMockTool("test-tool")
	_, original := NewExecution(rootTool, Arguments{})

	original.SetOutput(NewOK("test output"))
	original.Stats().Set("test_metric", "value")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal into a new node
	var unmarshaled Context
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if unmarshaled.ID() != original.ID() {
		t.Errorf("Expected ID %s, got %s", original.ID(), unmarshaled.ID())
	}

	if unmarshaled.parent != original.parent {
		t.Errorf("Expected parent %s, got %s", original.parent, unmarshaled.parent)
	}

	if len(unmarshaled.children) != len(original.children) {
		t.Errorf("Expected %d children, got %d", len(original.children), len(unmarshaled.children))
	}
}

func TestNode_UnmarshalJSON_InvalidData(t *testing.T) {
	var node Context
	err := json.Unmarshal([]byte("invalid json"), &node)
	if err == nil {
		t.Fatal("UnmarshalJSON should fail with invalid JSON")
	}
}

func TestNode_ConcurrentAccess(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	// Concurrent SetOutput
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			result := NewOK(id)
			node.SetOutput(result)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent ScopedData access
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			scope := "scope"
			_ = node.ScopedData(scope)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent Stats access
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			stats := node.Stats()
			stats.Set("test_metric", "value")
			_ = stats.Get("test_metric")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNode_DataSharing(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Create multiple children
	child1Tool := newMockTool("child1")
	child1 := e.createChild(root.ID(), child1Tool, Arguments{})
	if child1 == nil {
		t.Fatalf("createChild failed")
	}

	child2Tool := newMockTool("child2")
	child2 := e.createChild(root.ID(), child2Tool, Arguments{})
	if child2 == nil {
		t.Fatalf("createChild failed")
	}

	// All should share the same data ledger
	if root.data != child1.data {
		t.Fatal("Root and child1 should share same data ledger")
	}

	if root.data != child2.data {
		t.Fatal("Root and child2 should share same data ledger")
	}

	// All should share the same globalData
	if root.globalData != child1.globalData {
		t.Fatal("Root and child1 should share same globalData")
	}

	if root.globalData != child2.globalData {
		t.Fatal("Root and child2 should share same globalData")
	}


	// But contextData should be different
	if root.contextData == child1.contextData {
		t.Fatal("Root and child1 should have different contextData")
	}

	if child1.contextData == child2.contextData {
		t.Fatal("Child1 and child2 should have different contextData")
	}
}

func TestNode_StatsTiming(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	stats := node.Stats()
	stats.MarkStarted()
	startTime := stats.StartTime()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	stats.MarkFinished()
	endTime := stats.EndTime()

	if !endTime.After(startTime) {
		t.Fatal("EndTime should be after startTime")
	}

	duration := stats.ExecutionDuration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration should be at least 10ms, got %v", duration)
	}
}

func TestNode_EmptyChildren(t *testing.T) {
	rootTool := newMockTool("test-tool")
	_, node := NewExecution(rootTool, Arguments{})

	// New node should have empty children
	if len(node.children) != 0 {
		t.Errorf("New node should have no children, got %d", len(node.children))
	}
}

func TestNode_ParentRelationship(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Root should have empty parent
	if root.parent != "" {
		t.Errorf("Root should have empty parent, got %q", root.parent)
	}

	// Create child
	childTool := newMockTool("child-tool")
	child := e.createChild(root.ID(), childTool, Arguments{})
	if child == nil {
		t.Fatalf("createChild failed")
	}

	// Child should have root as parent
	if child.parent != root.ID() {
		t.Errorf("Child should have parent %s, got %s", root.ID(), child.parent)
	}
}
