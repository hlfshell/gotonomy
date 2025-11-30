package history

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/hlfshell/gotonomy/context"
)

type Event struct {
	source    context.NodeID
	key       string
	value     json.RawMessage
	timestamp time.Time
}

func NewEvent(key string, value json.RawMessage) *Event {
	return &Event{
		key:       key,
		value:     value,
		timestamp: time.Now(),
	}
}
func (e *Event) Key() string {
	return e.key
}
func (e *Event) Value() json.RawMessage {
	return e.value
}
func (e *Event) Timestamp() time.Time {
	return e.timestamp
}

func (e *Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Source    context.NodeID  `json:"source"`
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		Timestamp time.Time       `json:"timestamp"`
	}{
		Source:    e.source,
		Key:       e.key,
		Value:     e.value,
		Timestamp: e.timestamp,
	})
}

func (e *Event) UnmarshalJSON(data []byte) error {
	var aux struct {
		Source    context.NodeID  `json:"source"`
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		Timestamp time.Time       `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.source = aux.Source
	e.key = aux.Key
	e.value = aux.Value
	e.timestamp = aux.Timestamp
	return nil
}

type History struct {
	events []Event
	mu     sync.RWMutex
}

func NewHistory() *History {
	return &History{
		events: []Event{},
	}
}

func (h *History) AddEvent(event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

func (h *History) GetEvents() []Event {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.events
}

func (h *History) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Events []Event `json:"events"`
	}{
		Events: h.events,
	})
}
func (h *History) UnmarshalJSON(data []byte) error {
	var aux struct {
		Events []Event `json:"events"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	h.events = aux.Events
	return nil
}
