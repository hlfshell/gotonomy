package agent

import (
	"encoding/json"
	"time"

	"github.com/hlfshell/gotonomy/model"
)

type Step struct {
	input    []model.Message
	response Response
	stats    StepStats
}

func NewStep(input []model.Message) *Step {
	return &Step{
		input: input,
		stats: StepStats{
			SentAt: time.Now(),
		},
	}
}

func (s *Step) GetInput() []model.Message {
	return s.input
}

func (s *Step) GetResponse() Response {
	return s.response
}

func (s *Step) GetStats() StepStats {
	return s.stats
}

func (s *Step) SetResponse(response Response) {
	s.response = response
	s.stats.ReceivedAt = time.Now()
}

// AppendToolMessage appends a tool message to the step's input so that tool
// outputs can be replayed on subsequent LLM calls.
func (s *Step) AppendToolMessage(msg model.Message) {
	s.input = append(s.input, msg)
}

func (s *Step) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Input    []model.Message `json:"input"`
		Response Response        `json:"response"`
		Stats    StepStats       `json:"stats"`
		Duration string          `json:"duration"`
	}{
		Input:    s.input,
		Response: s.response,
		Stats:    s.stats,
		Duration: s.stats.Duration().String(),
	})
}

func (s *Step) UnmarshalJSON(data []byte) error {
	var aux struct {
		Input    []model.Message `json:"input"`
		Response Response        `json:"response"`
		Stats    StepStats       `json:"stats"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.input = aux.Input
	s.response = aux.Response
	s.stats = aux.Stats
	return nil
}

type Response struct {
	// If Error is populated, Output may be blank.
	Output    model.Message    `json:"output"`
	ToolCalls []model.ToolCall `json:"tool_calls"`
	Error     string           `json:"error,omitempty"`
}

type StepStats struct {
	SentAt     time.Time `json:"sent_at"`
	ReceivedAt time.Time `json:"received_at"`
}

func (ss *StepStats) Duration() time.Duration {
	return ss.ReceivedAt.Sub(ss.SentAt)
}

type Session struct {
	steps []*Step
}

func NewSession() *Session {
	return &Session{
		steps: []*Step{},
	}
}

func (s *Session) AddStep(step *Step) {
	s.steps = append(s.steps, step)
}

// Steps returns a copy of the steps slice for read-only access.
func (s *Session) Steps() []*Step {
	if len(s.steps) == 0 {
		return nil
	}
	out := make([]*Step, len(s.steps))
	copy(out, s.steps)
	return out
}

// LastStep returns the most recent step in the session, or nil if none exist.
func (s *Session) LastStep() *Step {
	if len(s.steps) == 0 {
		return nil
	}
	return s.steps[len(s.steps)-1]
}

// AppendToolMessage appends a tool message to the most recent step, if any.
func (s *Session) AppendToolMessage(msg model.Message) {
	last := s.LastStep()
	if last == nil {
		return
	}
	last.AppendToolMessage(msg)
}

// Conversation flattens the session into a sequence of model messages in the
// order they were sent/received.
func (s *Session) Conversation() []model.Message {
	var msgs []model.Message
	for _, step := range s.steps {
		if len(step.input) > 0 {
			msgs = append(msgs, step.input...)
		}
		// Always include the assistant output; for error steps this may be blank.
		msgs = append(msgs, step.response.Output)
	}
	return msgs
}

func (s *Session) Duration() time.Duration {
	if len(s.steps) == 0 {
		return 0
	}
	startTime := s.steps[0].GetStats().SentAt
	endTime := s.steps[len(s.steps)-1].GetStats().ReceivedAt
	if endTime.IsZero() {
		endTime = time.Now()
	}
	return endTime.Sub(startTime)
}

func (s *Session) Finished() bool {
	if len(s.steps) == 0 {
		return false
	}
	lastStep := s.steps[len(s.steps)-1]
	// Finished if no tool calls in the last step
	return len(lastStep.GetResponse().ToolCalls) == 0
}

func (s *Session) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Steps    []*Step `json:"steps"`
		Finished bool    `json:"finished"`
		Duration string  `json:"duration"`
	}{
		Steps:    s.steps,
		Finished: s.Finished(),
		Duration: s.Duration().String(),
	})
}

func (s *Session) UnmarshalJSON(data []byte) error {
	var aux struct {
		Steps []*Step `json:"steps"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.steps = aux.Steps
	return nil
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
