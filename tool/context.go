package tool

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hlfshell/gotonomy/data/ledger"
)

type ContextID string

type Context struct {
	id        ContextID
	toolName  string
	execution *Execution

	// Parent/children are mutated while holding Execution.mu.
	// Context.mu is used only for per-context fields (input/output/stats/etc.).
	parent   ContextID
	children []ContextID

	input  Arguments
	output ResultInterface

	// TODO - hash on each node thats updated
	// across any changes for future syncing
	// diff creations

	data        *ledger.Ledger
	globalData  *ledger.ScopedLedger
	contextData *ledger.ScopedLedger
	scopedData  map[string]*ledger.ScopedLedger

	stats Stats

	mu sync.RWMutex
}

func (c *Context) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Marshal stats separately to avoid copying the mutex
	statsJSON, err := json.Marshal(&c.stats)
	if err != nil {
		return nil, err
	}
	var statsData interface{}
	if err := json.Unmarshal(statsJSON, &statsData); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ID             ContextID      `json:"id"`
		Parent         ContextID      `json:"parent"`
		Children       []ContextID    `json:"children"`
		ExecutionStats interface{}    `json:"execution_stats"`
		Data           *ledger.Ledger `json:"data"`
	}{
		ID:             c.id,
		Parent:         c.parent,
		Children:       c.children,
		ExecutionStats: statsData,
		Data:           c.data,
	})
}

func (c *Context) UnmarshalJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var value struct {
		ID             ContextID       `json:"id"`
		Parent         ContextID       `json:"parent"`
		Children       []ContextID     `json:"children"`
		ExecutionStats json.RawMessage `json:"execution_stats"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	c.id = value.ID
	c.parent = value.Parent
	c.children = value.Children
	// Unmarshal stats separately to avoid copying the mutex
	if err := json.Unmarshal(value.ExecutionStats, &c.stats); err != nil {
		return err
	}
	return nil
}

func (c *Context) Data() *ledger.ScopedLedger {
	return c.contextData
}

func (c *Context) GlobalData() *ledger.ScopedLedger {
	return c.globalData
}

// ScopedData returns a scoped ledger globally - the intent
// being that tools that know to look for data within a known
// scope can utilize this as a form of controlled data sharing
func (c *Context) ScopedData(scope string) (*ledger.ScopedLedger, error) {
	c.mu.RLock()
	sl, ok := c.scopedData[scope]
	c.mu.RUnlock()
	if ok {
		return sl, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring write lock
	if sl, ok := c.scopedData[scope]; ok {
		return sl, nil
	}
	var err error
	sl, err = ledger.NewScoped(c.data, scope)
	if err != nil {
		return nil, err
	}
	c.scopedData[scope] = sl
	return sl, nil
}

// Stats returns a pointer to the node's stats.
func (c *Context) Stats() *Stats {
	return &c.stats
}

// ID returns the node's ID
func (c *Context) ID() ContextID {
	return c.id
}

// SetOutput sets the output result for this node
func (c *Context) SetOutput(output ResultInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output = output
}

// isBlank returns true if the context is blank (both ID and tool name are unset)
func (c *Context) isBlank() bool {
	return c.id == "" && c.toolName == ""
}

// PrepareContext prepares a context for use. We expect this to be called immediately
// in any tool implementing Execution. There are three possible outcomes:
//  1. The context is nil - this is a root node, and we need to create a temporary
//     execution and new root node.
//  2. The context is "blank". Update it to belong to this tool's execution.
//  3. The context is not "blank". Created a child node from the current ctx, as it
//     is actually our parent node.
func PrepareContext(ctx *Context, tool Tool, args Arguments) *Context {
	var c *Context
	if ctx == nil {
		// 1. The context is nil - this is a root node, and we need to
		// create a temporary execution and new root node.
		_, c = NewExecution(tool, args)
	} else if ctx.isBlank() {
		// 2. The context is "blank". Update it to belong to this tool's execution.
		// Then we set to return that same context.
		fillBlankContext(ctx, tool, args)
		c = ctx
	} else {
		// 3. The context is not "blank". Created a child node from the current ctx, as it
		//    is actually our parent node.
		c = ctx.execution.createChild(ctx.id, tool, args)
		if c == nil {
			// Parent not found - this should not happen in normal usage
			// Fall back to creating a new execution
			_, c = NewExecution(tool, args)
		}
	}
	return c
}

func blankContext(e *Execution) *Context {
	return &Context{
		id:          "",
		toolName:    "",
		execution:   e,
		parent:      "",
		children:    []ContextID{},
		input:       Arguments{},
		output:      nil,
		data:        e.data,       // shared execution ledger
		globalData:  e.globalData, // shared global scoped ledger
		contextData: nil,          // will be set in fillBlankContext
		scopedData:  make(map[string]*ledger.ScopedLedger),
		stats:       Stats{},
		mu:          sync.RWMutex{},
	}
}

func fillBlankContext(c *Context, tool Tool, args Arguments) {
	// First, set all context fields under context lock
	c.mu.Lock()
	c.id = ContextID(uuid.New().String())
	c.toolName = tool.Name()
	c.input = args

	// Initialize per-node scope
	scope := fmt.Sprintf("%s:%s", c.toolName, c.id)
	var err error
	c.contextData, err = ledger.NewScoped(c.data, scope)
	if err != nil {
		// This should not happen in normal usage, but handle it gracefully
		panic(fmt.Sprintf("failed to create scoped ledger: %v", err))
	}

	// Initialize children slice if needed
	if c.children == nil {
		c.children = []ContextID{}
	}
	c.mu.Unlock()

	// Then register as root on the execution (separate lock to avoid deadlock)
	c.execution.mu.Lock()
	c.execution.root = c.id
	c.execution.ctxs[c.id] = c
	c.execution.mu.Unlock()
}
