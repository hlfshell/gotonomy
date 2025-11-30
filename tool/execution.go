package tool

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hlfshell/gotonomy/data/history"
	"github.com/hlfshell/gotonomy/data/ledger"
	scopedledger "github.com/hlfshell/gotonomy/data/ledger/scoped_ledger"
)

// Execution tracks the entire execution chain of all tools
// w/ a built in ledger data store for communicating data
// across tools during execution
type Execution struct {
	root NodeID

	nodes map[NodeID]*Context

	// Execution-level data ledger (shared across all children)
	data       *ledger.Ledger
	globalData *scopedledger.ScopedLedger

	startedAt time.Time
	endedAt   time.Time

	// Mutex for thread safety
	mu sync.RWMutex
}

// isBlank returns true if the execution is nil or has no root and no nodes.
// A blank execution is one that hasn't actually started yet.
func (e *Execution) isBlank() bool {
	if e == nil {
		return true
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.root == "" && len(e.nodes) == 0
}

// newExecutionWithRoot creates:
// - the shared ledger
// - global scoped ledger
// - a fully wired root node for tool t + args
// If e is nil or blank, it will be allocated/initialized. If e is already initialized,
// it will be reused and the root node will be added to it.
func newExecutionWithRoot(
	e *Execution, // may be nil or blank
	t Tool,
	args Arguments,
) (*Execution, *Context, error) {
	// Allocate execution if nil
	if e == nil {
		e = &Execution{}
	}

	// Initialize core fields if needed
	if e.data == nil {
		e.data = ledger.NewLedger()
		e.globalData = scopedledger.NewScopedLedger(e.data, "global")
		e.nodes = make(map[NodeID]*Context)
		e.stats = Stats{}
	}

	// Create the root node
	hist := history.NewHistory()
	rootID := NodeID(uuid.New().String())
	rootScope := fmt.Sprintf("%s:%s", t.Name(), rootID)

	root := &Context{
		id:         rootID,
		toolName:   t.Name(),
		parent:     "",
		children:   []NodeID{},
		input:      args,
		output:     nil,
		data:       e.data,
		globalData: e.globalData,
		nodeData:   scopedledger.NewScopedLedger(e.data, rootScope),
		scopedData: map[string]*scopedledger.ScopedLedger{},
		history:    hist,
		stats: Stats{
			startTime: time.Now(),
			endTime:   time.Time{},
		},
	}

	// Assign root into execution
	e.root = rootID
	e.nodes[rootID] = root

	return e, root, nil
}

// createChildInternal creates a child node under a given parent.
// This is the unified source of truth for child node creation.
func (e *Execution) createChildInternal(
	parentID NodeID,
	t Tool,
	args Arguments,
) (*Context, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	parent, ok := e.nodes[parentID]
	if !ok {
		return nil, fmt.Errorf("parent node %s not found", parentID)
	}

	id := NodeID(uuid.New().String())
	scope := fmt.Sprintf("%s:%s", t.Name(), id)

	child := &Context{
		id:         id,
		toolName:   t.Name(),
		parent:     parentID,
		children:   []NodeID{},
		input:      args,
		output:     nil,
		data:       e.data,
		globalData: e.globalData,
		nodeData:   scopedledger.NewScopedLedger(e.data, scope),
		scopedData: map[string]*scopedledger.ScopedLedger{},
		history:    parent.history,
		stats: Stats{
			startTime: time.Now(),
			endTime:   time.Time{},
		},
	}

	parent.children = append(parent.children, id)
	e.nodes[id] = child

	return child, nil
}

// PrepareExecution is the single entrypoint for preparing an execution.
// It handles:
// - e == nil or blank: creates a new execution + fully wired root node
// - e != nil and not blank: creates a child node under the given parent
//
// For the first tool call, callers can either:
//   - pass nil as the *Execution, or
//   - pass a blank &Execution{} if they want to keep a reference early
//
// For subsequent calls:
//   - reuse the returned exec
//   - set parent to "" to default to the root, or
//   - set parent to a specific NodeID to attach under that node
//
// Returns both the execution and the node representing this tool call.
// Always returns the execution (even on error) so the caller can keep their reference.
func PrepareExecution(
	e *Execution,
	parent NodeID,
	t Tool,
	args Arguments,
) (*Execution, *Context, error) {
	// Treat nil or blank as a first call
	if e == nil || e.isBlank() {
		if parent != "" {
			return e, nil, fmt.Errorf("cannot specify parent for first tool call")
		}
		return newExecutionWithRoot(e, t, args)
	}

	// For a non-blank execution, allow parent == "" to default to root
	if parent == "" {
		parent = e.root
	}

	child, err := e.createChildInternal(parent, t, args)
	if err != nil {
		return e, nil, err
	}

	return e, child, nil
}

func (e *Execution) Tree() map[NodeID][]NodeID {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Initialize with all node IDs as keys to ensure complete adjacency map
	children := make(map[NodeID][]NodeID, len(e.nodes))
	for id := range e.nodes {
		children[id] = []NodeID{}
	}

	// Fill in child lists
	for _, n := range e.nodes {
		if n.parent != "" {
			children[n.parent] = append(children[n.parent], n.id)
		}
	}
	return children
}

func (e *Execution) Data() *ledger.Ledger {
	return e.data
}

func (e *Execution) GlobalData() *scopedledger.ScopedLedger {
	return e.globalData
}

func (e *Execution) Stats() *Stats {
	return &e.stats
}

// RootID returns the ID of the root node
func (e *Execution) RootID() NodeID {
	return e.root
}

// Root returns the root node, or nil if not found
func (e *Execution) Root() *Context {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.nodes[e.root]
}
