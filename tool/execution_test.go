package tool

import (
	"testing"
	"time"

	"github.com/hlfshell/gotonomy/data/ledger"
	"github.com/hlfshell/gotonomy/utils/semver"
)

// mockTool is a simple tool implementation for testing
type mockTool struct {
	name        string
	description string
	params      []Parameter
}

func (m *mockTool) ID() string {
	return m.name
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Version() semver.SemVer {
	return semver.SemVer{}
}

func (m *mockTool) Parameters() []Parameter {
	return m.params
}

func (m *mockTool) Execute(ctx *Context, args Arguments) ResultInterface {
	return NewOK("mock result")
}

func newMockTool(name string) Tool {
	return &mockTool{
		name:        name,
		description: "mock tool for testing",
		params:      []Parameter{},
	}
}

func TestNewExecution(t *testing.T) {
	tool := newMockTool("test-tool")
	args := Arguments{"key": "value"}

	e, node := NewExecution(tool, args)

	if e == nil {
		t.Fatal("Execution should not be nil")
	}

	if node == nil {
		t.Fatal("Node should not be nil")
	}

	if node.ID() == "" {
		t.Fatal("Node should have an ID")
	}

	if node.toolName != "test-tool" {
		t.Errorf("Expected toolName 'test-tool', got %q", node.toolName)
	}

	if e.RootID() != node.ID() {
		t.Errorf("Root ID should match node ID")
	}

	if e.Root() != node {
		t.Errorf("Root() should return the root node")
	}

	// Verify all fields are initialized
	if node.data == nil {
		t.Fatal("Node data should not be nil")
	}

	if node.globalData == nil {
		t.Fatal("Node globalData should not be nil")
	}

	if node.contextData == nil {
		t.Fatal("Node contextData should not be nil")
	}

	// Stats should be initialized (check that we can access it)
	stats := node.Stats()
	if stats == nil {
		t.Fatal("Node stats should not be nil")
	}
}

func TestExecution_CreateChild(t *testing.T) {
	// Create root execution
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Create child
	childTool := newMockTool("child-tool")
	child := e.createChild(root.ID(), childTool, Arguments{"child": "data"})
	if child == nil {
		t.Fatal("Child should not be nil")
	}

	if child.parent != root.ID() {
		t.Errorf("Expected parent %s, got %s", root.ID(), child.parent)
	}

	if child.toolName != "child-tool" {
		t.Errorf("Expected toolName 'child-tool', got %q", child.toolName)
	}

	// Verify child is in parent's children list
	found := false
	for _, childID := range root.children {
		if childID == child.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Child should be in parent's children list")
	}

	// Verify child shares same data ledger
	if child.data != root.data {
		t.Fatal("Child should share same data ledger as parent")
	}

	if child.globalData != root.globalData {
		t.Fatal("Child should share same globalData as parent")
	}

}

func TestPrepareContext_WithParent(t *testing.T) {
	// Create root execution
	rootTool := newMockTool("root-tool")
	_, root := NewExecution(rootTool, Arguments{})

	// Create child using PrepareContext
	childTool := newMockTool("child-tool")
	child := PrepareContext(root, childTool, Arguments{})

	if child == nil {
		t.Fatal("Child should not be nil")
	}

	if child.parent != root.ID() {
		t.Errorf("Expected parent %s, got %s", root.ID(), child.parent)
	}

	if child.execution != root.execution {
		t.Fatal("Child should share same execution as parent")
	}
}

func TestPrepareContext_ErrorCases(t *testing.T) {
	// Test: nil context creates new execution
	rootTool := newMockTool("root-tool")
	root := PrepareContext(nil, rootTool, Arguments{})
	if root == nil {
		t.Fatal("PrepareContext should create root when context is nil")
	}

	// Test: invalid parent ID for existing execution (createChild returns nil)
	childTool := newMockTool("child-tool")
	invalidParent := &Context{
		id:        ContextID("nonexistent"),
		execution: root.execution,
	}
	child := PrepareContext(invalidParent, childTool, Arguments{})
	// Should fall back to creating new execution
	if child == nil {
		t.Fatal("PrepareContext should create new execution when parent not found")
	}
}

func TestPrepareContext_BlankContext(t *testing.T) {
	// Test: blank context gets filled
	data := ledger.NewLedger()
	blankExec := &Execution{
		data: data,
		globalData: func() *ledger.ScopedLedger {
			sl, _ := ledger.NewScoped(data, "global")
			return sl
		}(),
		ctxs: make(map[ContextID]*Context),
	}
	blankCtx := blankContext(blankExec)
	rootTool := newMockTool("root-tool")

	ctx := PrepareContext(blankCtx, rootTool, Arguments{})
	if ctx == nil {
		t.Fatal("Context should not be nil")
	}

	if ctx.ID() == "" {
		t.Fatal("Context should have an ID after PrepareContext")
	}

	if blankExec.RootID() != ctx.ID() {
		t.Errorf("Root ID should match context ID")
	}
}

func TestPrepareContext_ChildFromParent(t *testing.T) {
	// Create root execution
	rootTool := newMockTool("root-tool")
	_, root := NewExecution(rootTool, Arguments{})

	// Create child using PrepareContext with parent
	childTool := newMockTool("child-tool")
	child := PrepareContext(root, childTool, Arguments{})

	if child.parent != root.ID() {
		t.Errorf("Expected parent %s (root), got %s", root.ID(), child.parent)
	}

	if child.execution != root.execution {
		t.Fatal("Child should share same execution as parent")
	}
}

func TestExecution_Tree(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Initially, tree should have only root
	tree := e.Tree()
	if len(tree) != 1 {
		t.Errorf("Expected tree to have 1 node, got %d", len(tree))
	}

	if len(tree[root.ID()]) != 0 {
		t.Errorf("Root should have no children initially, got %d", len(tree[root.ID()]))
	}

	// Add children
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

	// Verify tree structure
	tree = e.Tree()
	if len(tree) != 3 {
		t.Errorf("Expected tree to have 3 nodes, got %d", len(tree))
	}

	if len(tree[root.ID()]) != 2 {
		t.Errorf("Root should have 2 children, got %d", len(tree[root.ID()]))
	}

	if len(tree[child1.ID()]) != 0 {
		t.Errorf("Child1 should have no children, got %d", len(tree[child1.ID()]))
	}

	if len(tree[child2.ID()]) != 0 {
		t.Errorf("Child2 should have no children, got %d", len(tree[child2.ID()]))
	}

	// Verify children are in root's list
	found1, found2 := false, false
	for _, childID := range tree[root.ID()] {
		if childID == child1.ID() {
			found1 = true
		}
		if childID == child2.ID() {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Fatal("Both children should be in root's children list")
	}
}

func TestExecution_Data(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, _ := NewExecution(rootTool, Arguments{})

	data := e.Data()
	if data == nil {
		t.Fatal("Data should not be nil")
	}

	globalData := e.GlobalData()
	if globalData == nil {
		t.Fatal("GlobalData should not be nil")
	}
}

func TestExecution_RootStats(t *testing.T) {
	rootTool := newMockTool("root-tool")
	_, root := NewExecution(rootTool, Arguments{})

	stats := root.Stats()
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
}

func TestExecution_RootID(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	rootID := e.RootID()
	if rootID != root.ID() {
		t.Errorf("RootID should match root node ID")
	}
}

func TestExecution_Root(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	retrievedRoot := e.Root()
	if retrievedRoot == nil {
		t.Fatal("Root() should not return nil")
	}

	if retrievedRoot.ID() != root.ID() {
		t.Errorf("Root() should return the root node")
	}
}

func TestExecution_ConcurrentAccess(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			childTool := newMockTool("child")
			child := e.createChild(root.ID(), childTool, Arguments{"id": id})
			if child == nil {
				t.Errorf("createChild failed")
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_ = e.Tree()
			_ = e.Data()
			_ = e.GlobalData()
			_ = e.Root()
			_ = e.RootID()
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all children were created
	tree := e.Tree()
	if len(tree[root.ID()]) != 10 {
		t.Errorf("Expected 10 children, got %d", len(tree[root.ID()]))
	}
}

func TestExecution_DeepTree(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Create a deep tree: root -> child1 -> grandchild1 -> greatgrandchild1
	child1Tool := newMockTool("child1")
	child1 := e.createChild(root.ID(), child1Tool, Arguments{})
	if child1 == nil {
		t.Fatalf("createChild failed")
	}

	grandchild1Tool := newMockTool("grandchild1")
	grandchild1 := e.createChild(child1.ID(), grandchild1Tool, Arguments{})
	if grandchild1 == nil {
		t.Fatalf("createChild failed")
	}

	greatgrandchild1Tool := newMockTool("greatgrandchild1")
	greatgrandchild1 := e.createChild(grandchild1.ID(), greatgrandchild1Tool, Arguments{})
	if greatgrandchild1 == nil {
		t.Fatalf("createChild failed")
	}

	// Verify tree structure
	tree := e.Tree()
	if len(tree) != 4 {
		t.Errorf("Expected tree to have 4 nodes, got %d", len(tree))
	}

	if len(tree[root.ID()]) != 1 {
		t.Errorf("Root should have 1 child, got %d", len(tree[root.ID()]))
	}

	if len(tree[child1.ID()]) != 1 {
		t.Errorf("Child1 should have 1 child, got %d", len(tree[child1.ID()]))
	}

	if len(tree[grandchild1.ID()]) != 1 {
		t.Errorf("Grandchild1 should have 1 child, got %d", len(tree[grandchild1.ID()]))
	}

	if len(tree[greatgrandchild1.ID()]) != 0 {
		t.Errorf("Greatgrandchild1 should have no children, got %d", len(tree[greatgrandchild1.ID()]))
	}
}

func TestExecution_MultipleChildren(t *testing.T) {
	rootTool := newMockTool("root-tool")
	e, root := NewExecution(rootTool, Arguments{})

	// Create multiple children under root
	children := make([]*Context, 5)
	for i := 0; i < 5; i++ {
		childTool := newMockTool("child")
		child := e.createChild(root.ID(), childTool, Arguments{"index": i})
		if child == nil {
			t.Fatalf("createChild failed")
		}
		children[i] = child
	}

	// Verify all children are in root's children list
	tree := e.Tree()
	if len(tree[root.ID()]) != 5 {
		t.Errorf("Root should have 5 children, got %d", len(tree[root.ID()]))
	}

	// Verify each child has correct parent
	for _, child := range children {
		if child.parent != root.ID() {
			t.Errorf("Child %s should have parent %s, got %s", child.ID(), root.ID(), child.parent)
		}
	}
}

func TestExecution_StatsTiming(t *testing.T) {
	rootTool := newMockTool("root-tool")
	_, root := NewExecution(rootTool, Arguments{})

	stats := root.Stats()
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
