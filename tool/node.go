package tool

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hlfshell/gotonomy/data/history"
	"github.com/hlfshell/gotonomy/data/ledger"
	scopedledger "github.com/hlfshell/gotonomy/data/ledger/scoped_ledger"
)

type NodeID string

type Context struct {
	id       NodeID
	toolName string

	// Parent/children are mutated under Execution.mu, not Node.mu
	parent   NodeID
	children []NodeID

	input  Arguments
	output ResultInterface

	// TODO - hash on each node thats updated
	// across any changes for future syncing
	// diff creations

	// Nodes must always be constructed via Execution helpers (newExecutionWithRoot,
	// createChildInternal); data must never be nil.
	data       *ledger.Ledger
	globalData *scopedledger.ScopedLedger
	nodeData   *scopedledger.ScopedLedger
	scopedData map[string]*scopedledger.ScopedLedger
	history    *history.History

	stats Stats

	mu sync.RWMutex
}

func (n *Context) MarshalJSON() ([]byte, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	// Marshal stats separately to avoid copying the mutex
	statsJSON, err := json.Marshal(&n.stats)
	if err != nil {
		return nil, err
	}
	var statsData interface{}
	if err := json.Unmarshal(statsJSON, &statsData); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ID             NodeID         `json:"id"`
		Parent         NodeID         `json:"parent"`
		Children       []NodeID       `json:"children"`
		ExecutionStats interface{}    `json:"execution_stats"`
		Data           *ledger.Ledger `json:"data"`
	}{
		ID:             n.id,
		Parent:         n.parent,
		Children:       n.children,
		ExecutionStats: statsData,
		Data:           n.data,
	})
}

func (n *Context) UnmarshalJSON(data []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	var value struct {
		ID             NodeID          `json:"id"`
		Parent         NodeID          `json:"parent"`
		Children       []NodeID        `json:"children"`
		ExecutionStats json.RawMessage `json:"execution_stats"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.id = value.ID
	n.parent = value.Parent
	n.children = value.Children
	// Unmarshal stats separately to avoid copying the mutex
	if err := json.Unmarshal(value.ExecutionStats, &n.stats); err != nil {
		return err
	}
	return nil
}

func (n *Context) History() *history.History {
	return n.history
}

func (n *Context) Broadcast(key string, value any) error {
	if n.history == nil {
		return fmt.Errorf("history not initialized")
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	n.history.AddEvent(*history.NewEvent(key, valueJSON))
	return nil
}

func (n *Context) Data() *scopedledger.ScopedLedger {
	return n.nodeData
}

func (n *Context) GlobalData() *scopedledger.ScopedLedger {
	return n.globalData
}

// ScopedData returns a scoped ledger globally - the intent
// being that tools that know to look for data within a known
// scope can utilize this as a form of controlled data sharing
func (n *Context) ScopedData(scope string) *scopedledger.ScopedLedger {
	n.mu.RLock()
	sl, ok := n.scopedData[scope]
	n.mu.RUnlock()
	if ok {
		return sl
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	// Double-check after acquiring write lock
	if sl, ok := n.scopedData[scope]; ok {
		return sl
	}
	sl = scopedledger.NewScopedLedger(n.data, scope)
	n.scopedData[scope] = sl
	return sl
}

// Stats returns a pointer to the node's stats.
// Note: Stats is not independently lock-protected; access is safe when
// the node is accessed through Execution methods that hold appropriate locks.
func (n *Context) Stats() *Stats {
	return &n.stats
}

// ID returns the node's ID
func (n *Context) ID() NodeID {
	return n.id
}

// SetOutput sets the output result for this node
func (n *Context) SetOutput(output ResultInterface) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.output = output
}

// CreateChild creates a child node. This method requires access to the Execution
// to properly register the child. For new code, use PrepareExecution instead.
// This is kept for backward compatibility but may be deprecated.
func (n *Context) CreateChild(e *Execution, tool Tool, args Arguments) (*Context, error) {
	if e == nil {
		return nil, fmt.Errorf("execution required to create child node")
	}
	return e.createChildInternal(n.id, tool, args)
}
