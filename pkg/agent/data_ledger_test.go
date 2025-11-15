package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSetData(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set data
	err = SetData(execCtx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data
	value, ok := GetData[string](execCtx, "key1")
	if !ok {
		t.Fatal("GetData should return true")
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}

	// Set different types
	err = SetData(execCtx, "key2", 42)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = SetData(execCtx, "key3", true)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify different types
	value2, ok := GetData[int](execCtx, "key2")
	if !ok {
		t.Fatal("GetData should return true for key2")
	}
	if value2 != 42 {
		t.Errorf("Expected value 42, got %d", value2)
	}

	value3, ok := GetData[bool](execCtx, "key3")
	if !ok {
		t.Fatal("GetData should return true for key3")
	}
	if value3 != true {
		t.Errorf("Expected value true, got %v", value3)
	}
}

func TestGetData(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test getting data when none is set
	_, ok := GetData[string](execCtx, "nonexistent")
	if ok {
		t.Fatal("GetData should return false for nonexistent key")
	}

	// Create child and set data
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	err = SetData(execCtx, "test", "value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Test getting data
	value, ok := GetData[string](execCtx, "test")
	if !ok {
		t.Fatal("GetData should return true")
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got %q", value)
	}
}

func TestGetDataHistory(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set data multiple times
	err = SetData(execCtx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = SetData(execCtx, "key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = SetData(execCtx, "key1", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get history
	history := execCtx.GetDataHistory("key1")
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}

	// Verify most recent value
	value, ok := GetData[string](execCtx, "key1")
	if !ok {
		t.Fatal("GetData should return true")
	}
	if value != "value3" {
		t.Errorf("Expected most recent value 'value3', got %q", value)
	}

	// Verify all entries have correct key
	for _, entry := range history {
		if entry.Key != "key1" {
			t.Errorf("Expected key 'key1', got %q", entry.Key)
		}
		if entry.Operation != "set" {
			t.Errorf("Expected operation 'set', got %q", entry.Operation)
		}
	}
}

func TestSetExecutionData(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Set execution data
	err := SetExecutionData(execCtx, "exec_key1", "exec_value1")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	// Verify execution data
	value, ok := GetExecutionData[string](execCtx, "exec_key1")
	if !ok {
		t.Fatal("GetExecutionData should return true")
	}
	if value != "exec_value1" {
		t.Errorf("Expected value 'exec_value1', got %q", value)
	}

	// Set different types
	err = SetExecutionData(execCtx, "exec_key2", 100)
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	err = SetExecutionData(execCtx, "exec_key3", map[string]interface{}{"nested": "value"})
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	// Verify different types
	value2, ok := GetExecutionData[int](execCtx, "exec_key2")
	if !ok {
		t.Fatal("GetExecutionData should return true for exec_key2")
	}
	if value2 != 100 {
		t.Errorf("Expected value 100, got %d", value2)
	}

	value3, ok := GetExecutionData[map[string]interface{}](execCtx, "exec_key3")
	if !ok {
		t.Fatal("GetExecutionData should return true for exec_key3")
	}
	if value3["nested"] != "value" {
		t.Errorf("Expected nested value 'value', got %v", value3["nested"])
	}
}

func TestGetExecutionData(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test getting execution data when none is set
	_, ok := GetExecutionData[string](execCtx, "nonexistent")
	if ok {
		t.Fatal("GetExecutionData should return false for nonexistent key")
	}

	// Set execution data
	err := SetExecutionData(execCtx, "test", "value")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	// Test getting execution data
	value, ok := GetExecutionData[string](execCtx, "test")
	if !ok {
		t.Fatal("GetExecutionData should return true")
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got %q", value)
	}
}

func TestGetExecutionDataHistory(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Set execution data multiple times
	err := SetExecutionData(execCtx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = SetExecutionData(execCtx, "key1", "value2")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = SetExecutionData(execCtx, "key2", "other_value")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	// Get history for key1
	history := execCtx.GetExecutionDataHistory("key1")
	if len(history) != 2 {
		t.Errorf("Expected history length 2 for key1, got %d", len(history))
	}

	// Verify all entries have correct key
	for _, entry := range history {
		if entry.Key != "key1" {
			t.Errorf("Expected key 'key1', got %q", entry.Key)
		}
		if entry.Operation != "set" {
			t.Errorf("Expected operation 'set', got %q", entry.Operation)
		}
	}

	// Get history for key2
	history2 := execCtx.GetExecutionDataHistory("key2")
	if len(history2) != 1 {
		t.Errorf("Expected history length 1 for key2, got %d", len(history2))
	}

	// Get history for nonexistent key
	history3 := execCtx.GetExecutionDataHistory("nonexistent")
	if len(history3) != 0 {
		t.Errorf("Expected history length 0 for nonexistent key, got %d", len(history3))
	}
}

func TestDeleteData(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set data
	err = SetData(execCtx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data exists
	_, ok := GetData[string](execCtx, "key1")
	if !ok {
		t.Fatal("Data should exist before deletion")
	}

	// Delete data
	execCtx.DeleteData("key1")

	// Verify data is deleted
	_, ok = GetData[string](execCtx, "key1")
	if ok {
		t.Fatal("Data should not exist after deletion")
	}

	// Verify delete entry is in history
	history := execCtx.GetDataHistory("key1")
	if len(history) != 2 {
		t.Errorf("Expected history length 2 (set + delete), got %d", len(history))
	}

	lastEntry := history[len(history)-1]
	if lastEntry.Operation != "delete" {
		t.Errorf("Expected last operation to be 'delete', got %q", lastEntry.Operation)
	}
}

func TestDeleteExecutionData(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Set execution data
	err := SetExecutionData(execCtx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	// Verify data exists
	_, ok := GetExecutionData[string](execCtx, "key1")
	if !ok {
		t.Fatal("Execution data should exist before deletion")
	}

	// Delete execution data
	execCtx.DeleteExecutionData("key1")

	// Verify data is deleted
	_, ok = GetExecutionData[string](execCtx, "key1")
	if ok {
		t.Fatal("Execution data should not exist after deletion")
	}

	// Verify delete entry is in history
	history := execCtx.GetExecutionDataHistory("key1")
	if len(history) != 2 {
		t.Errorf("Expected history length 2 (set + delete), got %d", len(history))
	}

	lastEntry := history[len(history)-1]
	if lastEntry.Operation != "delete" {
		t.Errorf("Expected last operation to be 'delete', got %q", lastEntry.Operation)
	}
}

func TestDataLedgerEntry(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set data and verify entry structure
	err = SetData(execCtx, "test_key", "test_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	history := execCtx.GetDataHistory("test_key")
	if len(history) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(history))
	}

	entry := history[0]
	if entry.Key != "test_key" {
		t.Errorf("Expected key 'test_key', got %q", entry.Key)
	}

	if entry.Operation != "set" {
		t.Errorf("Expected operation 'set', got %q", entry.Operation)
	}

	if entry.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}

	// Verify value can be unmarshaled
	var value string
	err = json.Unmarshal(entry.Value, &value)
	if err != nil {
		t.Fatalf("Failed to unmarshal entry value: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got %q", value)
	}
}

func TestDataIsolationBetweenNodes(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create first child node and set as current
	child1, err := execCtx.CreateChildNode(nil, "agent", "agent1", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}
	if err := execCtx.SetCurrentNode(child1); err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	// Set data in first node
	err = SetData(execCtx, "node1_key", "node1_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Reset to root and create second child node
	err = execCtx.SetCurrentNode(execCtx.GetRootNode())
	if err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	child2, err := execCtx.CreateChildNode(nil, "agent", "agent2", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}
	if err := execCtx.SetCurrentNode(child2); err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	// Verify data from first node is not accessible in second node
	_, ok := GetData[string](execCtx, "node1_key")
	if ok {
		t.Fatal("Data from first node should not be accessible in second node")
	}

	// Set data in second node
	err = SetData(execCtx, "node2_key", "node2_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data is set in second node
	value, ok := GetData[string](execCtx, "node2_key")
	if !ok {
		t.Fatal("Data should be accessible in second node")
	}
	if value != "node2_value" {
		t.Errorf("Expected value 'node2_value', got %q", value)
	}

	// Verify we can switch back to first node and access its data
	err = execCtx.SetCurrentNode(child1)
	if err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	value1, ok := GetData[string](execCtx, "node1_key")
	if !ok {
		t.Fatal("Data should be accessible when switching back to first node")
	}
	if value1 != "node1_value" {
		t.Errorf("Expected value 'node1_value', got %q", value1)
	}

	// Verify second node's data is not accessible from first node
	_, ok = GetData[string](execCtx, "node2_key")
	if ok {
		t.Fatal("Data from second node should not be accessible in first node")
	}
}

func TestExecutionDataSharedAcrossNodes(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Set execution data
	err := SetExecutionData(execCtx, "shared_key", "shared_value")
	if err != nil {
		t.Fatalf("SetExecutionData failed: %v", err)
	}

	// Create first child node and set as current
	child1, err := execCtx.CreateChildNode(nil, "agent", "agent1", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}
	if err := execCtx.SetCurrentNode(child1); err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	// Verify execution data is accessible
	value, ok := GetExecutionData[string](execCtx, "shared_key")
	if !ok {
		t.Fatal("Execution data should be accessible in first node")
	}
	if value != "shared_value" {
		t.Errorf("Expected value 'shared_value', got %q", value)
	}

	// Reset to root and create second child node
	err = execCtx.SetCurrentNode(execCtx.GetRootNode())
	if err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	child2, err := execCtx.CreateChildNode(nil, "agent", "agent2", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}
	if err := execCtx.SetCurrentNode(child2); err != nil {
		t.Fatalf("SetCurrentNode failed: %v", err)
	}

	// Verify execution data is still accessible
	value, ok = GetExecutionData[string](execCtx, "shared_key")
	if !ok {
		t.Fatal("Execution data should be accessible in second node")
	}
	if value != "shared_value" {
		t.Errorf("Expected value 'shared_value', got %q", value)
	}
}

func TestDataUpdateHistory(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set data multiple times
	values := []string{"value1", "value2", "value3"}
	for i, v := range values {
		err = SetData(execCtx, "key", v)
		if err != nil {
			t.Fatalf("SetData failed for value %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Verify most recent value
	value, ok := GetData[string](execCtx, "key")
	if !ok {
		t.Fatal("GetData should return true")
	}
	if value != "value3" {
		t.Errorf("Expected most recent value 'value3', got %q", value)
	}

	// Verify history contains all values
	history := execCtx.GetDataHistory("key")
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}

	// Verify timestamps are in order
	for i := 1; i < len(history); i++ {
		if history[i].Timestamp.Before(history[i-1].Timestamp) {
			t.Errorf("Timestamps should be in chronological order")
		}
	}
}

func TestComplexDataTypes(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Test with struct
	type TestStruct struct {
		Name  string
		Value int
	}

	testStruct := TestStruct{Name: "test", Value: 42}
	err = SetData(execCtx, "struct_key", testStruct)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	value, ok := GetData[TestStruct](execCtx, "struct_key")
	if !ok {
		t.Fatal("GetData should return true")
	}
	if value.Name != "test" || value.Value != 42 {
		t.Errorf("Expected struct {Name: 'test', Value: 42}, got %+v", value)
	}

	// Test with slice
	slice := []int{1, 2, 3}
	err = SetData(execCtx, "slice_key", slice)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	valueSlice, ok := GetData[[]int](execCtx, "slice_key")
	if !ok {
		t.Fatal("GetData should return true for slice")
	}
	if len(valueSlice) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(valueSlice))
	}

	// Test with map
	testMap := map[string]interface{}{"key1": "value1", "key2": 42}
	err = SetData(execCtx, "map_key", testMap)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	valueMap, ok := GetData[map[string]interface{}](execCtx, "map_key")
	if !ok {
		t.Fatal("GetData should return true for map")
	}
	if valueMap["key1"] != "value1" {
		t.Errorf("Expected map value 'value1', got %v", valueMap["key1"])
	}
}

func TestConcurrentDataAccess(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode(nil, "agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			err := SetData(execCtx, "concurrent_key", id)
			if err != nil {
				t.Errorf("SetData failed: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, _ = GetData[int](execCtx, "concurrent_key")
			_ = execCtx.GetDataHistory("concurrent_key")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

