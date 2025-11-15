package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNewExecutionContext(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	if execCtx == nil {
		t.Fatal("NewExecutionContext returned nil")
	}

	if execCtx.root == nil {
		t.Fatal("Root node should not be nil")
	}

	if execCtx.current == nil {
		t.Fatal("Current node should not be nil")
	}

	if execCtx.current != execCtx.root {
		t.Fatal("Current node should be root initially")
	}

	if execCtx.root.Type != "root" {
		t.Errorf("Expected root type to be 'root', got %s", execCtx.root.Type)
	}

	if execCtx.root.Name != "root" {
		t.Errorf("Expected root name to be 'root', got %s", execCtx.root.Name)
	}

	if execCtx.root.ID == "" {
		t.Fatal("Root node should have an ID")
	}

	if execCtx.executionDataLedger == nil {
		t.Fatal("Execution data ledger should not be nil")
	}
}

func TestAsExecutionContext(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test with ExecutionContext
	result, ok := AsExecutionContext(execCtx)
	if !ok {
		t.Fatal("AsExecutionContext should return true for ExecutionContext")
	}
	if result != execCtx {
		t.Fatal("AsExecutionContext should return the same ExecutionContext")
	}

	// Test with regular context
	regularCtx := context.Background()
	result, ok = AsExecutionContext(regularCtx)
	if ok {
		t.Fatal("AsExecutionContext should return false for regular context")
	}
	if result != nil {
		t.Fatal("AsExecutionContext should return nil for regular context")
	}
}

func TestGetOrCreateExecutionContext(t *testing.T) {
	ctx := context.Background()

	// Test creating new context
	execCtx1 := GetOrCreateExecutionContext(ctx)
	if execCtx1 == nil {
		t.Fatal("GetOrCreateExecutionContext should not return nil")
	}

	// Test getting existing context
	execCtx2 := GetOrCreateExecutionContext(execCtx1)
	if execCtx2 != execCtx1 {
		t.Fatal("GetOrCreateExecutionContext should return existing ExecutionContext")
	}
}

func TestCreateChildNode(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	child, err := execCtx.CreateChildNode("agent", "test-agent", map[string]string{"input": "test"})
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	if child == nil {
		t.Fatal("Child node should not be nil")
	}

	if child.ID == "" {
		t.Fatal("Child node should have an ID")
	}

	if child.ParentID != execCtx.root.ID {
		t.Errorf("Expected parent ID to be %s, got %s", execCtx.root.ID, child.ParentID)
	}

	if child.Type != "agent" {
		t.Errorf("Expected type to be 'agent', got %s", child.Type)
	}

	if child.Name != "test-agent" {
		t.Errorf("Expected name to be 'test-agent', got %s", child.Name)
	}

	if execCtx.current != child {
		t.Fatal("Current node should be the newly created child")
	}

	if len(execCtx.root.Children) != 1 {
		t.Errorf("Expected root to have 1 child, got %d", len(execCtx.root.Children))
	}

	// Test with nil input
	child2, err := execCtx.CreateChildNode("tool", "test-tool", nil)
	if err != nil {
		t.Fatalf("CreateChildNode with nil input failed: %v", err)
	}

	if child2.Input != nil && len(child2.Input) > 0 {
		t.Error("Child node with nil input should have empty Input")
	}
}

func TestCreateChildNodeWithInvalidInput(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a channel which cannot be marshaled to JSON
	ch := make(chan int)
	_, err := execCtx.CreateChildNode("agent", "test", ch)
	if err == nil {
		t.Fatal("CreateChildNode should fail with unmarshalable input")
	}
}

func TestSetOutput(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node first
	_, err := execCtx.CreateChildNode("agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set output
	output := "test output"
	err = SetOutput(execCtx, output)
	if err != nil {
		t.Fatalf("SetOutput failed: %v", err)
	}

	// Verify output
	result, ok := GetOutput[string](execCtx)
	if !ok {
		t.Fatal("GetOutput should return true")
	}
	if result != output {
		t.Errorf("Expected output %q, got %q", output, result)
	}

	// Verify EndTime is set
	if execCtx.current.EndTime == nil {
		t.Fatal("EndTime should be set after SetOutput")
	}
}

func TestGetOutput(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test getting output when none is set
	_, ok := GetOutput[string](execCtx)
	if ok {
		t.Fatal("GetOutput should return false when no output is set")
	}

	// Create child and set output
	_, err := execCtx.CreateChildNode("agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	err = SetOutput(execCtx, 42)
	if err != nil {
		t.Fatalf("SetOutput failed: %v", err)
	}

	// Test getting output
	result, ok := GetOutput[int](execCtx)
	if !ok {
		t.Fatal("GetOutput should return true")
	}
	if result != 42 {
		t.Errorf("Expected output 42, got %d", result)
	}
}

func TestSetError(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode("agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set error
	testErr := context.DeadlineExceeded
	execCtx.SetError(testErr)

	// Verify error
	errorStr := execCtx.GetError()
	if errorStr != testErr.Error() {
		t.Errorf("Expected error %q, got %q", testErr.Error(), errorStr)
	}

	// Verify EndTime is set
	if execCtx.current.EndTime == nil {
		t.Fatal("EndTime should be set after SetError")
	}

	// Clear error
	execCtx.SetError(nil)
	errorStr = execCtx.GetError()
	if errorStr != "" {
		t.Errorf("Expected empty error after clearing, got %q", errorStr)
	}
}

func TestGetError(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test getting error when none is set
	errorStr := execCtx.GetError()
	if errorStr != "" {
		t.Errorf("Expected empty error, got %q", errorStr)
	}

	// Set error
	execCtx.SetError(context.Canceled)
	errorStr = execCtx.GetError()
	if errorStr != context.Canceled.Error() {
		t.Errorf("Expected error %q, got %q", context.Canceled.Error(), errorStr)
	}
}

func TestSetMetadata(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a child node
	_, err := execCtx.CreateChildNode("agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Set metadata
	err = SetMetadata(execCtx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	// Verify metadata
	value, ok := GetMetadata[string](execCtx, "key1")
	if !ok {
		t.Fatal("GetMetadata should return true")
	}
	if value != "value1" {
		t.Errorf("Expected metadata value 'value1', got %q", value)
	}

	// Set another metadata
	err = SetMetadata(execCtx, "key2", 42)
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	value2, ok := GetMetadata[int](execCtx, "key2")
	if !ok {
		t.Fatal("GetMetadata should return true for key2")
	}
	if value2 != 42 {
		t.Errorf("Expected metadata value 42, got %d", value2)
	}
}

func TestGetMetadata(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test getting metadata when none is set
	_, ok := GetMetadata[string](execCtx, "nonexistent")
	if ok {
		t.Fatal("GetMetadata should return false for nonexistent key")
	}

	// Create child and set metadata
	_, err := execCtx.CreateChildNode("agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	err = SetMetadata(execCtx, "test", "value")
	if err != nil {
		t.Fatalf("SetMetadata failed: %v", err)
	}

	// Test getting metadata
	value, ok := GetMetadata[string](execCtx, "test")
	if !ok {
		t.Fatal("GetMetadata should return true")
	}
	if value != "value" {
		t.Errorf("Expected metadata value 'value', got %q", value)
	}
}

func TestGetCurrentNode(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Initially current should be root
	current := execCtx.GetCurrentNode()
	if current != execCtx.root {
		t.Fatal("Current node should be root initially")
	}

	// Create child
	child, err := execCtx.CreateChildNode("agent", "test", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Current should be child
	current = execCtx.GetCurrentNode()
	if current != child {
		t.Fatal("Current node should be the child")
	}
}

func TestGetRootNode(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	root := execCtx.GetRootNode()
	if root != execCtx.root {
		t.Fatal("GetRootNode should return the root node")
	}

	if root.Type != "root" {
		t.Errorf("Expected root type to be 'root', got %s", root.Type)
	}
}

func TestSerialize(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create child nodes
	_, err := execCtx.CreateChildNode("agent", "agent1", "input1")
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	_, err = execCtx.CreateChildNode("tool", "tool1", "input2")
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Serialize
	data, err := execCtx.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized data should not be empty")
	}

	// Verify it's valid JSON
	var aux struct {
		Root                *Node             `json:"root"`
		CurrentID           string            `json:"current_id"`
		ExecutionDataLedger []DataLedgerEntry `json:"execution_data_ledger"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		t.Fatalf("Serialized data should be valid JSON: %v", err)
	}

	if aux.CurrentID != execCtx.current.ID {
		t.Errorf("Expected CurrentID %s, got %s", execCtx.current.ID, aux.CurrentID)
	}
}

func TestDeserializeExecutionContext(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create child nodes and set some data
	_, err := execCtx.CreateChildNode("agent", "agent1", "input1")
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	err = SetOutput(execCtx, "output1")
	if err != nil {
		t.Fatalf("SetOutput failed: %v", err)
	}

	// Serialize
	data, err := execCtx.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Deserialize
	newCtx := context.Background()
	deserialized, err := DeserializeExecutionContext(newCtx, data)
	if err != nil {
		t.Fatalf("DeserializeExecutionContext failed: %v", err)
	}

	if deserialized == nil {
		t.Fatal("Deserialized context should not be nil")
	}

	if deserialized.root == nil {
		t.Fatal("Deserialized root should not be nil")
	}

	if deserialized.current == nil {
		t.Fatal("Deserialized current should not be nil")
	}

	if deserialized.current.ID != execCtx.current.ID {
		t.Errorf("Expected current ID %s, got %s", execCtx.current.ID, deserialized.current.ID)
	}

	// Verify output was preserved
	output, ok := GetOutput[string](deserialized)
	if !ok {
		t.Fatal("Output should be preserved after deserialization")
	}
	if output != "output1" {
		t.Errorf("Expected output 'output1', got %q", output)
	}
}

func TestGetExecutionChain(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Initially should have only root
	chain := execCtx.GetExecutionChain()
	if len(chain) != 1 {
		t.Errorf("Expected chain length 1, got %d", len(chain))
	}
	if chain[0] != execCtx.root {
		t.Fatal("Chain should start with root")
	}

	// Create child nodes
	child1, err := execCtx.CreateChildNode("agent", "agent1", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	child2, err := execCtx.CreateChildNode("tool", "tool1", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Get chain
	chain = execCtx.GetExecutionChain()
	if len(chain) != 3 {
		t.Errorf("Expected chain length 3, got %d", len(chain))
	}

	if chain[0] != execCtx.root {
		t.Fatal("Chain should start with root")
	}
	if chain[1] != child1 {
		t.Fatal("Chain should include child1")
	}
	if chain[2] != child2 {
		t.Fatal("Chain should include child2")
	}
}

func TestExecutionContextContextInterface(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	execCtx := NewExecutionContext(ctx)

	// Test Deadline
	deadline, ok := execCtx.Deadline()
	expectedDeadline, expectedOk := ctx.Deadline()
	if ok != expectedOk {
		t.Errorf("Expected Deadline ok=%v, got %v", expectedOk, ok)
	}
	if ok && !deadline.Equal(expectedDeadline) {
		t.Errorf("Expected deadline %v, got %v", expectedDeadline, deadline)
	}

	// Test Done
	done := execCtx.Done()
	if done == nil {
		t.Fatal("Done channel should not be nil")
	}

	// Test Err
	err := execCtx.Err()
	if err != nil {
		t.Errorf("Expected no error initially, got %v", err)
	}

	// Cancel context
	cancel()

	// Wait a bit for cancellation to propagate
	time.Sleep(10 * time.Millisecond)

	// Test Err after cancellation
	err = execCtx.Err()
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Test Value
	key := "test-key"
	value := "test-value"
	ctxWithValue := context.WithValue(ctx, key, value)
	execCtxWithValue := NewExecutionContext(ctxWithValue)

	result := execCtxWithValue.Value(key)
	if result != value {
		t.Errorf("Expected value %v, got %v", value, result)
	}
}

func TestFindNodeByID(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Create a tree of nodes
	child1, err := execCtx.CreateChildNode("agent", "agent1", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	child2, err := execCtx.CreateChildNode("tool", "tool1", nil)
	if err != nil {
		t.Fatalf("CreateChildNode failed: %v", err)
	}

	// Test finding root
	found := findNodeByID(execCtx.root, execCtx.root.ID)
	if found != execCtx.root {
		t.Fatal("Should find root node")
	}

	// Test finding child1
	found = findNodeByID(execCtx.root, child1.ID)
	if found != child1 {
		t.Fatal("Should find child1 node")
	}

	// Test finding child2
	found = findNodeByID(execCtx.root, child2.ID)
	if found != child2 {
		t.Fatal("Should find child2 node")
	}

	// Test finding nonexistent node
	found = findNodeByID(execCtx.root, "nonexistent-id")
	if found != nil {
		t.Fatal("Should not find nonexistent node")
	}

	// Test with nil root
	found = findNodeByID(nil, "any-id")
	if found != nil {
		t.Fatal("Should return nil for nil root")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	execCtx := NewExecutionContext(ctx)

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			_, err := execCtx.CreateChildNode("agent", "agent", id)
			if err != nil {
				t.Errorf("CreateChildNode failed: %v", err)
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
			_ = execCtx.GetCurrentNode()
			_ = execCtx.GetRootNode()
			_ = execCtx.GetError()
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

