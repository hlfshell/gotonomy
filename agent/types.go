package agent

import (
	"encoding/json"
	"time"

	"github.com/hlfshell/gogentic/model"
)

type Call struct {
	Input    []Message `json:"input"`
	Response Response  `json:"response"`
	Stats    CallStats `json:"stats"`
}

func NewCall(input []Message) *Call {
	return &Call{
		Input: input,
		Stats: CallStats{
			SentAt: time.Now(),
		},
	}
}

func (c *Call) AddResponse(response Response) {
	c.Response = response
	c.Stats.ReceivedAt = time.Now()
}

// MarshalJSON implements json.Marshaler to include duration in JSON output.
func (c *Call) MarshalJSON() ([]byte, error) {
	type Alias Call // Avoid recursion
	return json.Marshal(&struct {
		*Alias
		Duration string `json:"duration"`
	}{
		Alias:    (*Alias)(c),
		Duration: c.Stats.Duration().String(),
	})
}

type Response struct {
	Output    Message          `json:"output"`
	ToolCalls []model.ToolCall `json:"tool_calls"`
}

type CallStats struct {
	SentAt     time.Time `json:"sent_at"`
	ReceivedAt time.Time `json:"received_at"`
}

func (c *CallStats) Duration() time.Duration {
	return c.ReceivedAt.Sub(c.SentAt)
}

// Message represents a message in a conversation with an agent.
type Message struct {
	// Role is the role of the message sender (e.g., "system", "user", "assistant", "tool").
	Role string
	// Content is the text content of the message.
	Content string
}

type Execution struct {
	Calls []Call `json:"calls"`
}

func NewCallHistory() *Execution {
	return &Execution{
		Calls: []Call{},
	}
}

func (c *Execution) AddCall(call Call) {
	c.Calls = append(c.Calls, call)
}

func (c *Execution) Duration() time.Duration {
	if len(c.Calls) == 0 {
		return 0
	}
	startTime := c.Calls[0].Stats.SentAt
	endTime := c.Calls[len(c.Calls)-1].Stats.ReceivedAt
	if endTime.IsZero() {
		endTime = time.Now()
	}
	return endTime.Sub(startTime)
}

func (c *Execution) Finished() bool {
	TODO
}

// Aliasing in duration
func (c *Execution) MarshalJSON() ([]byte, error) {
	type Alias Execution
	return json.Marshal(&struct {
		*Alias
		Finished bool   `json:"finished"`
		Duration string `json:"duration"`
	}{
		Alias:    (*Alias)(c),
		Finished: c.Finished(),
		Duration: c.Duration().String(),
	})
}

// ExecutionStats contains statistics about an agent execution.
type ExecutionStats struct {
	// StartTime is when execution started
	StartTime time.Time

	// EndTime is when execution completed
	EndTime time.Time

	// ToolCalls is the number of tool calls made
	ToolCalls int //TODO - swap to historical log of otol calls w/
	//timing info, etc

	// Iterations is the number of reasoning iterations
	Iterations int
}
