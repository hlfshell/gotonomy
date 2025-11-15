// Package agent provides interfaces and implementations for building AI agents
// that can use language models to accomplish tasks.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExecutionContext tracks the entire execution chain with ledger-based state management
type ExecutionContext struct {
	// Standard context for cancellation/timeout
	ctx context.Context

	// Root execution node
	root *Node

	// Current execution node
	current *Node

	// Execution-level data ledger (shared across all children) - tracks historical changes
	executionDataLedger []DataLedgerEntry

	// Mutex for thread safety
	mu sync.RWMutex
}

// Node represents a node in the execution tree
type Node struct {
	ID        string
	ParentID  string
	Type      string // "agent", "tool", "iteration", "root"
	Name      string
	Input     json.RawMessage   // Serialized input
	Output    json.RawMessage   // Serialized output
	Data      []DataLedgerEntry // Node-specific data ledger (tracks historical changes)
	Children  []*Node
	StartTime time.Time
	EndTime   *time.Time
	Error     string // Error message (serializable)
	Metadata  map[string]json.RawMessage
}

// NewExecutionContext creates a new execution context wrapping the given context
func NewExecutionContext(ctx context.Context) *ExecutionContext {
	root := &Node{
		ID:        uuid.New().String(),
		Type:      "root",
		Name:      "root",
		Data:      []DataLedgerEntry{},
		Metadata:  make(map[string]json.RawMessage),
		Children:  []*Node{},
		StartTime: time.Now(),
	}

	return &ExecutionContext{
		ctx:                 ctx,
		root:                root,
		current:             root,
		executionDataLedger: []DataLedgerEntry{},
	}
}

// AsExecutionContext extracts *ExecutionContext from context.Context if present
func AsExecutionContext(ctx context.Context) (*ExecutionContext, bool) {
	if execCtx, ok := ctx.(*ExecutionContext); ok {
		return execCtx, true
	}
	return nil, false
}

// InitContext gets existing ExecutionContext or creates a new one
// as needed for simplicity.
func InitContext(ctx context.Context) *ExecutionContext {
	if execCtx, ok := AsExecutionContext(ctx); ok {
		return execCtx
	}
	return NewExecutionContext(ctx)
}

// CreateChildNode creates a new child node under the specified parent node.
// If parent is nil, uses the current node as parent.
// The new node is NOT automatically set as current - use SetCurrentNode() if needed.
func (ec *ExecutionContext) CreateChildNode(parent *Node, nodeType, name string, input any) (*Node, error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Use current as parent if not specified
	if parent == nil {
		parent = ec.current
	}

	// Validate parent exists in the DAG
	if parent == nil {
		return nil, fmt.Errorf("cannot create child node: no parent specified and no current node")
	}

	var inputData json.RawMessage
	var err error
	if input != nil {
		inputData, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}
	}

	child := &Node{
		ID:        uuid.New().String(),
		ParentID:  parent.ID,
		Type:      nodeType,
		Name:      name,
		Input:     inputData,
		Data:      []DataLedgerEntry{},
		Metadata:  make(map[string]json.RawMessage),
		Children:  []*Node{},
		StartTime: time.Now(),
	}

	parent.Children = append(parent.Children, child)

	return child, nil
}

// SetCurrentNode explicitly sets which node is current for operations like SetOutput, GetData, etc.
func (ec *ExecutionContext) SetCurrentNode(node *Node) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Validate node exists in the DAG
	if node != nil && !ec.nodeExistsInDAG(node) {
		return fmt.Errorf("node %s does not exist in the execution DAG", node.ID)
	}

	ec.current = node
	return nil
}

// nodeExistsInDAG checks if a node exists in the DAG by searching from root
func (ec *ExecutionContext) nodeExistsInDAG(node *Node) bool {
	if node == nil {
		return false
	}
	found := findNodeByID(ec.root, node.ID)
	return found != nil
}

// GetNodeByID finds a node in the DAG by its ID
func (ec *ExecutionContext) GetNodeByID(id string) *Node {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return findNodeByID(ec.root, id)
}

// GetParentNode returns the parent of the given node, or nil if it's the root
func (ec *ExecutionContext) GetParentNode(node *Node) *Node {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if node == nil || node.ParentID == "" {
		return nil
	}

	return findNodeByID(ec.root, node.ParentID)
}

// SetOutput sets the output for the current node
func SetOutput[T any](ec *ExecutionContext, output T) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.current == nil {
		return fmt.Errorf("cannot set output: no current node")
	}

	data, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	ec.current.Output = data
	now := time.Now()
	ec.current.EndTime = &now
	return nil
}

// GetOutput retrieves the output from the current node
func GetOutput[T any](ec *ExecutionContext) (T, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var zero T
	if ec.current == nil {
		return zero, false
	}

	if len(ec.current.Output) == 0 {
		return zero, false
	}

	var value T
	if err := json.Unmarshal(ec.current.Output, &value); err != nil {
		return zero, false
	}

	return value, true
}

// SetError sets an error for the current node
func (ec *ExecutionContext) SetError(err error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.current == nil {
		return // Silently ignore if no current node
	}

	if err != nil {
		ec.current.Error = err.Error()
	} else {
		ec.current.Error = ""
	}
	now := time.Now()
	ec.current.EndTime = &now
}

// GetError retrieves the error from the current node
func (ec *ExecutionContext) GetError() string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	if ec.current == nil {
		return ""
	}
	return ec.current.Error
}

// SetMetadata sets metadata for the current node
func SetMetadata[T any](ec *ExecutionContext, key string, value T) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.current == nil {
		return fmt.Errorf("cannot set metadata: no current node")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if ec.current.Metadata == nil {
		ec.current.Metadata = make(map[string]json.RawMessage)
	}
	ec.current.Metadata[key] = data
	return nil
}

// GetMetadata retrieves metadata from the current node
func GetMetadata[T any](ec *ExecutionContext, key string) (T, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var zero T
	if ec.current == nil {
		return zero, false
	}

	if ec.current.Metadata == nil {
		return zero, false
	}

	data, ok := ec.current.Metadata[key]
	if !ok {
		return zero, false
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, false
	}

	return value, true
}

// GetCurrentNode returns the current execution node
func (ec *ExecutionContext) GetCurrentNode() *Node {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.current
}

// GetRootNode returns the root execution node
func (ec *ExecutionContext) GetRootNode() *Node {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.root
}

// Serialize serializes the entire execution context to JSON
func (ec *ExecutionContext) Serialize() ([]byte, error) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	data := struct {
		Root                *Node             `json:"root"`
		CurrentID           string            `json:"current_id"`
		ExecutionDataLedger []DataLedgerEntry `json:"execution_data_ledger"`
	}{
		Root:                ec.root,
		CurrentID:           ec.current.ID,
		ExecutionDataLedger: ec.executionDataLedger,
	}

	return json.Marshal(data)
}

// DeserializeExecutionContext recreates an execution context from JSON
func DeserializeExecutionContext(ctx context.Context, data []byte) (*ExecutionContext, error) {
	var aux struct {
		Root                *Node             `json:"root"`
		CurrentID           string            `json:"current_id"`
		ExecutionDataLedger []DataLedgerEntry `json:"execution_data_ledger"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution context: %w", err)
	}

	// Find current node by ID
	current := findNodeByID(aux.Root, aux.CurrentID)
	if current == nil {
		current = aux.Root // Fallback to root
	}

	return &ExecutionContext{
		ctx:                 ctx,
		root:                aux.Root,
		current:             current,
		executionDataLedger: aux.ExecutionDataLedger,
	}, nil
}

// Save writes the execution context to a file
func (ec *ExecutionContext) Save(filename string) error {
	data, err := ec.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize execution context: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// Load reads an execution context from a file
func Load(ctx context.Context, filename string) (*ExecutionContext, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return DeserializeExecutionContext(ctx, data)
}

// findNodeByID recursively finds a node by ID
func findNodeByID(root *Node, id string) *Node {
	if root == nil {
		return nil
	}
	if root.ID == id {
		return root
	}
	for _, child := range root.Children {
		if found := findNodeByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

// GetExecutionChain returns all nodes in execution order
func (ec *ExecutionContext) GetExecutionChain() []*Node {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var chain []*Node
	collectNodes(ec.root, &chain)
	return chain
}

// collectNodes recursively collects all nodes in depth-first order
func collectNodes(node *Node, chain *[]*Node) {
	if node == nil {
		return
	}
	*chain = append(*chain, node)
	for _, child := range node.Children {
		collectNodes(child, chain)
	}
}

// Implement context.Context interface methods
func (ec *ExecutionContext) Deadline() (time.Time, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.ctx.Deadline()
}

func (ec *ExecutionContext) Done() <-chan struct{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.ctx.Done()
}

func (ec *ExecutionContext) Err() error {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.ctx.Err()
}

func (ec *ExecutionContext) Value(key interface{}) interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.ctx.Value(key)
}
