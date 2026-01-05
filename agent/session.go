package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hlfshell/gotonomy/data/ledger"
	"github.com/hlfshell/gotonomy/model"
)

type Step struct {
	// input is the message slice that was sent to the model for this step.
	// It should be treated as immutable once the step is created.
	input []model.Message

	// appended are messages added after the model response for this step,
	// e.g. tool outputs and extractor feedback to influence the next iteration.
	appended []model.Message

	response Response
	stats    StepStats
}

func NewStep(input []model.Message) *Step {
	// Copy input so callers can't mutate it after construction.
	inputCopy := make([]model.Message, len(input))
	copy(inputCopy, input)
	return &Step{
		input: inputCopy,
		stats: StepStats{
			SentAt: time.Now(),
		},
	}
}

func (s *Step) GetInput() []model.Message {
	if len(s.input) == 0 {
		return nil
	}
	out := make([]model.Message, len(s.input))
	copy(out, s.input)
	return out
}

func (s *Step) GetAppended() []model.Message {
	if len(s.appended) == 0 {
		return nil
	}
	out := make([]model.Message, len(s.appended))
	copy(out, s.appended)
	return out
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
	s.appended = append(s.appended, msg)
}

func (s *Step) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Input    []model.Message `json:"input"`
		Appended []model.Message `json:"appended,omitempty"`
		Response Response        `json:"response"`
		Stats    StepStats       `json:"stats"`
		Duration string          `json:"duration"`
	}{
		Input:    s.input,
		Appended: s.appended,
		Response: s.response,
		Stats:    s.stats,
		Duration: s.stats.Duration().String(),
	})
}

func (s *Step) UnmarshalJSON(data []byte) error {
	var aux struct {
		Input    []model.Message `json:"input"`
		Appended []model.Message `json:"appended"`
		Response Response        `json:"response"`
		Stats    StepStats       `json:"stats"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.input = aux.Input
	s.appended = aux.Appended

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
	ledger *ledger.ScopedLedger
	steps  []*Step
}

func (s *Session) Iterations() int {
	return len(s.steps)
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

// NewSession creates a new session. If a ledger is provided, it attempts to
// load an existing session from the ledger. Otherwise, it creates a fresh session.
func NewSession(sessionLedger ...*ledger.ScopedLedger) *Session {
	var sl *ledger.ScopedLedger
	if len(sessionLedger) > 0 && sessionLedger[0] != nil {
		sl = sessionLedger[0]
		// Sessions are objects comprised of multiple ledger entries
		// each of which is a step. Each step is stored with its
		// index as the key.
		keys := sl.GetKeys()
		stepIdxs := []int{}
		for _, key := range keys {
			if strings.HasPrefix(key, "step:") {
				stepIdx, err := strconv.Atoi(strings.TrimPrefix(key, "step:"))
				if err != nil {
					continue
				}
				stepIdxs = append(stepIdxs, stepIdx)
			}
		}

		steps := make([]*Step, 0, len(stepIdxs))
		for _, stepIdx := range stepIdxs {
			step, err := ledger.GetDataScoped[*Step](sl, fmt.Sprintf("step:%d", stepIdx))
			if err != nil {
				continue
			}
			if stepIdx < len(steps) {
				steps[stepIdx] = step
			} else {
				// Grow slice if needed
				for len(steps) <= stepIdx {
					steps = append(steps, nil)
				}
				steps[stepIdx] = step
			}
		}

		// Sort the step indexes
		sort.Ints(stepIdxs)

		return &Session{
			ledger: sl,
			steps:  steps,
		}
	}

	// Create a new empty session
	return &Session{
		ledger: nil,
		steps:  []*Step{},
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

// AppendSystemMessage appends a system-role message to the most recent step.
// This is primarily used by extractors to inject guidance without modifying
// the original user prompt.
func (s *Session) AppendSystemMessage(content string) {
	msg := model.Message{
		Role:    model.RoleSystem,
		Content: content,
	}
	s.AppendToolMessage(msg)
}

// AppendUserMessage appends a user-role message to the most recent step.
// This can be useful for simulating follow-up user questions generated
// by higher-level orchestration logic.
func (s *Session) AppendUserMessage(content string) {
	msg := model.Message{
		Role:    model.RoleUser,
		Content: content,
	}
	s.AppendToolMessage(msg)
}

// Conversation flattens the session into a sequence of model messages in the
// order they were sent/received. For steps with tool calls, the order is:
// 1. Original input messages (user/system messages)
// 2. Assistant message with tool_calls (if one was returned)
// 3. Tool result messages
func (s *Session) Conversation() []model.Message {
	var msgs []model.Message
	for _, step := range s.steps {
		// Original input messages
		msgs = append(msgs, step.input...)

		// Assistant output (may include tool_calls at provider level)
		if step.response.Output.Role != "" {
			msgs = append(msgs, step.response.Output)
		}

		// Tool outputs + extractor feedback appended after the response
		msgs = append(msgs, step.appended...)
	}
	return msgs
}

func (s *Session) Finished() bool {
	if len(s.steps) == 0 {
		return false
	}
	lastStep := s.steps[len(s.steps)-1]
	// Finished if no tool calls in the last step
	return len(lastStep.GetResponse().ToolCalls) == 0
}
