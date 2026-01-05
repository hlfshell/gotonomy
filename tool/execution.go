package tool

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hlfshell/gotonomy/data/ledger"
)

// Execution tracks the entire execution chain of all tools
// w/ a built in ledger data store for communicating data
// across tools during execution
type Execution struct {
	root ContextID

	ctxs map[ContextID]*Context

	// Execution-level data ledger (shared across all children)
	data       *ledger.Ledger
	globalData *ledger.ScopedLedger

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewExecution creates a fresh execution and a fully initialized root context.
func NewExecution(tool Tool, args Arguments) (*Execution, *Context) {
	data := ledger.NewLedger()

	globalData, _ := ledger.NewScoped(data, "global")

	e := &Execution{
		root:       "",
		ctxs:       make(map[ContextID]*Context),
		data:       data,
		globalData: globalData,
		mu:         sync.RWMutex{},
	}

	root := blankContext(e)
	fillBlankContext(root, tool, args) // sets id, toolName, contextData, root id, etc.
	return e, root
}

// Tree returns an adjacency list mapping each context ID to the IDs of its direct children (if any).
// Each context in the execution will have an entry in the returned map, pointing to all direct descendants.
// Nodes with no children will have an empty slice.
//
// Example output (ContextID shown as simple strings for illustration):
//
//	{
//	  "root":    {"child1", "child2"},
//	  "child1":  {"grandchild1"},
//	  "child2":  {},
//	  "grandchild1": {},
//	}
//
// In this example, "root" has two children, "child1" and "child2". "child1" has one child, "grandchild1".
// Both "child2" and "grandchild1" have no children.
//
// The returned map covers all node IDs tracked by the Execution.
func (e *Execution) Tree() map[ContextID][]ContextID {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Initialize with all node IDs as keys to ensure complete adjacency map
	children := make(map[ContextID][]ContextID, len(e.ctxs))
	for id := range e.ctxs {
		children[id] = []ContextID{}
	}

	// Fill in child lists
	for _, n := range e.ctxs {
		if n.parent != "" {
			children[n.parent] = append(children[n.parent], n.id)
		}
	}
	return children
}

func (e *Execution) Data() *ledger.Ledger {
	return e.data
}

func (e *Execution) GlobalData() *ledger.ScopedLedger {
	return e.globalData
}

func (e *Execution) StartAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.root == "" {
		return time.Time{}
	}
	return e.ctxs[e.root].Stats().StartTime()
}

func (e *Execution) EndAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.root == "" {
		return time.Time{}
	}
	return e.ctxs[e.root].Stats().EndTime()
}

func (e *Execution) Duration() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.root == "" {
		return 0
	}
	return e.ctxs[e.root].Stats().ExecutionDuration()
}

// RootID returns the ID of the root node
func (e *Execution) RootID() ContextID {
	return e.root
}

// Root returns the root node, or nil if not found
func (e *Execution) Root() *Context {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ctxs[e.root]
}

// Context returns the context with the given ID, or nil if not found
func (e *Execution) Context(id ContextID) *Context {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ctxs[id]
}

// RootContext returns the root context, or nil if not found
// This is an alias for Root() for better ergonomics
func (e *Execution) RootContext() *Context {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ctxs[e.root]
}

func (e *Execution) createChild(parentID ContextID, tool Tool, args Arguments) *Context {
	e.mu.Lock()
	defer e.mu.Unlock()

	parent, ok := e.ctxs[parentID]
	if !ok {
		// Return nil if parent not found - caller should handle error
		return nil
	}

	id := ContextID(uuid.New().String())
	scope := fmt.Sprintf("%s:%s", tool.Name(), id)

	contextData, err := ledger.NewScoped(e.data, scope)
	if err != nil {
		// This should not happen in normal usage, but handle it gracefully
		return nil
	}

	child := &Context{
		id:          id,
		toolName:    tool.Name(),
		parent:      parentID,
		children:    []ContextID{},
		data:        e.data,
		globalData:  e.globalData,
		contextData: contextData,
		scopedData:  make(map[string]*ledger.ScopedLedger),
		execution:   e,
		stats:       Stats{},
		input:       args,
		output:      nil,
		mu:          sync.RWMutex{},
	}

	e.ctxs[id] = child

	// Update the children list of the parent
	// Note: We hold Execution.mu, so we can safely mutate parent.children
	parent.children = append(parent.children, id)

	return child
}
