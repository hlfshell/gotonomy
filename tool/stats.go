package tool

import (
	"encoding/json"
	"sync"
	"time"
)

// Stats tracks execution statistics with support for various metric types.
type Stats struct {
	startTime time.Time
	endTime   time.Time

	// Metric stores: map[metricName]primitiveValue
	timers   map[string]int64 // stored as nanoseconds
	counters map[string]int64
	values   map[string]any

	mu sync.RWMutex
}

func (s *Stats) StartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

func (s *Stats) EndTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endTime
}

func (s *Stats) MarkStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startTime = time.Now()
}

// MarkFinished marks the end time for overall execution timing
func (s *Stats) MarkFinished() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endTime = time.Now()
}

// ExecutionDuration returns the overall execution duration
func (s *Stats) ExecutionDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.endTime.IsZero() {
		return time.Since(s.startTime)
	}
	return s.endTime.Sub(s.startTime)
}

// Timer functions

// Time starts a timer for the given metric name.
// Returns a function that should be called to stop the timer.
// Expected usage:
//
//	stop := stats.Time("my_timer")
//	// --- do stuff ---
//	stop()
//
// OR
//
//	defer stats.Time("my_timer")()
func (s *Stats) Time(name string) func() {
	start := time.Now()
	return func() {
		s.Duration(name, time.Since(start))
	}
}

// Duration records a timer measurement with the given name and duration
func (s *Stats) Duration(name string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timers == nil {
		s.timers = make(map[string]int64)
	}
	s.timers[name] = duration.Nanoseconds()
}

// GetTime returns the timer duration for the given name, or nil if not found
func (s *Stats) GetTime(name string) *time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.timers == nil {
		return nil
	}
	nanos, ok := s.timers[name]
	if !ok {
		return nil
	}
	duration := time.Duration(nanos)
	return &duration
}

// Counter functions

// Incr increments counter metric with the given name
func (s *Stats) Incr(name string) {
	s.Add(name, 1)
}

// Decr decrements counter metric with the given name
func (s *Stats) Decr(name string) {
	s.Add(name, -1)
}

// Add adds the given value to a counter metric with the given name
func (s *Stats) Add(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counters == nil {
		s.counters = make(map[string]int64)
	}
	s.counters[name] += value
}

// Sub subtracts the given value from a counter metric with the given name
func (s *Stats) Sub(name string, value int64) {
	s.Add(name, -value)
}

// GetCount returns the counter value for the given name, or nil if not found
func (s *Stats) GetCount(name string) *int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.counters == nil {
		return nil
	}
	count, ok := s.counters[name]
	if !ok {
		return nil
	}
	return &count
}

// Value functions

// Set sets a value metric with the given name and value.
// Value can be any type (int, int64, float64, string, etc.).
// Only set operations are supported; values cannot be
// deleted or changed once set.
func (s *Stats) Set(name string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string]any)
	}
	s.values[name] = value
}

// Get returns the value for the given name, or nil if not found
func (s *Stats) Get(name string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.values == nil {
		return nil
	}
	return s.values[name]
}

// MarshalJSON implements json.Marshaler
func (s *Stats) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return json.Marshal(struct {
		StartTime time.Time        `json:"start_time"`
		EndTime   time.Time        `json:"end_time"`
		Timers    map[string]int64 `json:"timers"`
		Counters  map[string]int64 `json:"counters"`
		Values    map[string]any   `json:"values"`
	}{
		StartTime: s.startTime,
		EndTime:   s.endTime,
		Timers:    s.timers,
		Counters:  s.counters,
		Values:    s.values,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (s *Stats) UnmarshalJSON(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var aux struct {
		StartTime time.Time        `json:"start_time"`
		EndTime   time.Time        `json:"end_time"`
		Timers    map[string]int64 `json:"timers"` // Duration stored as nanoseconds
		Counters  map[string]int64 `json:"counters"`
		Values    map[string]any   `json:"values"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.startTime = aux.StartTime
	s.endTime = aux.EndTime
	s.timers = aux.Timers
	s.counters = aux.Counters
	s.values = aux.Values

	return nil
}
