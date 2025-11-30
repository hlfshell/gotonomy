package history

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hlfshell/gotonomy/context"
)

func TestNewEvent(t *testing.T) {
	key := "test_key"
	value := json.RawMessage(`"test_value"`)

	event := NewEvent(key, value)

	if event == nil {
		t.Fatal("NewEvent should not return nil")
	}
	if event.Key() != key {
		t.Errorf("Expected key %q, got %q", key, event.Key())
	}
	if string(event.Value()) != string(value) {
		t.Errorf("Expected value %q, got %q", string(value), string(event.Value()))
	}
	if event.Timestamp().IsZero() {
		t.Fatal("Timestamp should not be zero")
	}
	if !event.Timestamp().Before(time.Now().Add(time.Second)) {
		t.Fatal("Timestamp should be recent")
	}
}

func TestEvent_Key(t *testing.T) {
	event := NewEvent("test_key", json.RawMessage(`"value"`))
	if event.Key() != "test_key" {
		t.Errorf("Expected key 'test_key', got %q", event.Key())
	}
}

func TestEvent_Value(t *testing.T) {
	value := json.RawMessage(`"test_value"`)
	event := NewEvent("test_key", value)
	if string(event.Value()) != string(value) {
		t.Errorf("Expected value %q, got %q", string(value), string(event.Value()))
	}
}

func TestEvent_Timestamp(t *testing.T) {
	before := time.Now()
	event := NewEvent("test_key", json.RawMessage(`"value"`))
	after := time.Now()

	timestamp := event.Timestamp()
	if timestamp.Before(before) || timestamp.After(after) {
		t.Errorf("Timestamp %v should be between %v and %v", timestamp, before, after)
	}
}

func TestEvent_MarshalJSON(t *testing.T) {
	event := NewEvent("test_key", json.RawMessage(`"test_value"`))

	data, err := event.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result struct {
		Source    context.NodeID  `json:"source"`
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		Timestamp time.Time       `json:"timestamp"`
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Key != "test_key" {
		t.Errorf("Expected key 'test_key', got %q", result.Key)
	}
	if string(result.Value) != `"test_value"` {
		t.Errorf("Expected value '\"test_value\"', got %q", string(result.Value))
	}
	if result.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}
	// Source should be empty string (zero value) since it's never set
	if result.Source != "" {
		t.Errorf("Expected empty source, got %q", result.Source)
	}
}

func TestEvent_UnmarshalJSON(t *testing.T) {
	// Create JSON data
	jsonData := `{
		"source": "node123",
		"key": "test_key",
		"value": "\"test_value\"",
		"timestamp": "2023-01-01T00:00:00Z"
	}`

	event := &Event{}
	err := event.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if event.Key() != "test_key" {
		t.Errorf("Expected key 'test_key', got %q", event.Key())
	}
	if string(event.Value()) != `"test_value"` {
		t.Errorf("Expected value '\"test_value\"', got %q", string(event.Value()))
	}

	// Verify timestamp was unmarshaled
	expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	if !event.Timestamp().Equal(expectedTime) {
		t.Errorf("Expected timestamp %v, got %v", expectedTime, event.Timestamp())
	}
}

func TestNewHistory(t *testing.T) {
	h := NewHistory()

	if h == nil {
		t.Fatal("NewHistory should not return nil")
	}

	events := h.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected empty events, got %d", len(events))
	}
}

func TestHistory_AddEvent(t *testing.T) {
	h := NewHistory()

	event1 := Event{
		key:       "key1",
		value:     json.RawMessage(`"value1"`),
		timestamp: time.Now(),
	}

	event2 := Event{
		key:       "key2",
		value:     json.RawMessage(`"value2"`),
		timestamp: time.Now(),
	}

	h.AddEvent(event1)
	h.AddEvent(event2)

	events := h.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].Key() != "key1" {
		t.Errorf("Expected first event key 'key1', got %q", events[0].Key())
	}
	if string(events[0].Value()) != `"value1"` {
		t.Errorf("Expected first event value '\"value1\"', got %q", string(events[0].Value()))
	}

	if events[1].Key() != "key2" {
		t.Errorf("Expected second event key 'key2', got %q", events[1].Key())
	}
	if string(events[1].Value()) != `"value2"` {
		t.Errorf("Expected second event value '\"value2\"', got %q", string(events[1].Value()))
	}
}

func TestHistory_Mark(t *testing.T) {
	h := NewHistory()

	// Mark with string value
	err := h.Mark("key1", "value1")
	if err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	events := h.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].Key() != "key1" {
		t.Errorf("Expected key 'key1', got %q", events[0].Key())
	}

	// Verify value can be unmarshaled
	var value string
	err = json.Unmarshal(events[0].Value(), &value)
	if err != nil {
		t.Fatalf("Failed to unmarshal value: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}

	// Mark with different types
	err = h.Mark("key2", 42)
	if err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	err = h.Mark("key3", true)
	if err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	events = h.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestHistory_Mark_ComplexTypes(t *testing.T) {
	h := NewHistory()

	// Test with struct
	type TestStruct struct {
		Name  string
		Value int
	}
	testStruct := TestStruct{Name: "test", Value: 42}
	err := h.Mark("struct_key", testStruct)
	if err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	events := h.GetEvents()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	var value TestStruct
	err = json.Unmarshal(events[0].Value(), &value)
	if err != nil {
		t.Fatalf("Failed to unmarshal value: %v", err)
	}
	if value.Name != "test" || value.Value != 42 {
		t.Errorf("Expected struct {Name: 'test', Value: 42}, got %+v", value)
	}

	// Test with slice
	slice := []int{1, 2, 3}
	err = h.Mark("slice_key", slice)
	if err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	events = h.GetEvents()
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	var sliceValue []int
	err = json.Unmarshal(events[1].Value(), &sliceValue)
	if err != nil {
		t.Fatalf("Failed to unmarshal value: %v", err)
	}
	if len(sliceValue) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(sliceValue))
	}

	// Test with map
	testMap := map[string]interface{}{"key1": "value1", "key2": 42}
	err = h.Mark("map_key", testMap)
	if err != nil {
		t.Fatalf("Mark failed: %v", err)
	}

	events = h.GetEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}
}

func TestHistory_GetEvents(t *testing.T) {
	h := NewHistory()

	// Test empty history
	events := h.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected empty events, got %d", len(events))
	}

	// Add events
	h.Mark("key1", "value1")
	h.Mark("key2", "value2")
	h.Mark("key3", "value3")

	events = h.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Verify order
	if events[0].Key() != "key1" {
		t.Errorf("Expected first event key 'key1', got %q", events[0].Key())
	}
	if events[1].Key() != "key2" {
		t.Errorf("Expected second event key 'key2', got %q", events[1].Key())
	}
	if events[2].Key() != "key3" {
		t.Errorf("Expected third event key 'key3', got %q", events[2].Key())
	}
}

func TestHistory_GetEvents_ReturnsCopy(t *testing.T) {
	h := NewHistory()

	h.Mark("key1", "value1")
	events1 := h.GetEvents()

	// Modify the returned slice (should not affect internal state)
	events1 = append(events1, Event{key: "fake"})

	events2 := h.GetEvents()
	if len(events2) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events2))
	}
}

func TestHistory_MarshalJSON(t *testing.T) {
	h := NewHistory()

	// Test empty history
	data, err := h.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result struct {
		Events []Event `json:"events"`
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result.Events) != 0 {
		t.Errorf("Expected empty events, got %d", len(result.Events))
	}

	// Add events
	h.Mark("key1", "value1")
	h.Mark("key2", "value2")

	// Marshal
	data, err = h.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal and verify
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(result.Events))
	}

	if result.Events[0].Key() != "key1" {
		t.Errorf("Expected first event key 'key1', got %q", result.Events[0].Key())
	}
	if result.Events[1].Key() != "key2" {
		t.Errorf("Expected second event key 'key2', got %q", result.Events[1].Key())
	}
}

func TestHistory_UnmarshalJSON(t *testing.T) {
	h := NewHistory()

	// Create JSON data by marshaling actual events
	event1 := NewEvent("key1", json.RawMessage(`"value1"`))
	event2 := NewEvent("key2", json.RawMessage(`"value2"`))

	jsonData := struct {
		Events []Event `json:"events"`
	}{
		Events: []Event{*event1, *event2},
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = h.UnmarshalJSON(jsonBytes)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	events := h.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].Key() != "key1" {
		t.Errorf("Expected first event key 'key1', got %q", events[0].Key())
	}
	if events[1].Key() != "key2" {
		t.Errorf("Expected second event key 'key2', got %q", events[1].Key())
	}
}

func TestHistory_MarshalUnmarshalRoundTrip(t *testing.T) {
	h1 := NewHistory()

	// Add events
	h1.Mark("key1", "value1")
	h1.Mark("key2", 42)
	h1.Mark("key3", true)

	// Marshal
	data, err := h1.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal into new history
	h2 := NewHistory()
	err = h2.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Verify events match
	events1 := h1.GetEvents()
	events2 := h2.GetEvents()

	if len(events1) != len(events2) {
		t.Errorf("Expected %d events, got %d", len(events1), len(events2))
	}

	for i := range events1 {
		if events1[i].Key() != events2[i].Key() {
			t.Errorf("Event %d: Expected key %q, got %q", i, events1[i].Key(), events2[i].Key())
		}
		if string(events1[i].Value()) != string(events2[i].Value()) {
			t.Errorf("Event %d: Expected value %q, got %q", i, string(events1[i].Value()), string(events2[i].Value()))
		}
	}
}

func TestHistory_ConcurrentAccess(t *testing.T) {
	h := NewHistory()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			err := h.Mark("concurrent_key", id)
			if err != nil {
				t.Errorf("Mark failed: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_ = h.GetEvents()
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	events := h.GetEvents()
	if len(events) != 10 {
		t.Errorf("Expected 10 events, got %d", len(events))
	}
}

func TestHistory_ConcurrentAddEvent(t *testing.T) {
	h := NewHistory()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			event := Event{
				key:       "key",
				value:     json.RawMessage(`"value"`),
				timestamp: time.Now(),
			}
			h.AddEvent(event)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	events := h.GetEvents()
	if len(events) != 10 {
		t.Errorf("Expected 10 events, got %d", len(events))
	}
}

func TestHistory_EdgeCases(t *testing.T) {
	h := NewHistory()

	// Test empty key
	err := h.Mark("", "value")
	if err != nil {
		t.Fatalf("Mark should accept empty key: %v", err)
	}

	events := h.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
	if events[0].Key() != "" {
		t.Errorf("Expected empty key, got %q", events[0].Key())
	}

	// Test empty value
	err = h.Mark("empty_value", "")
	if err != nil {
		t.Fatalf("Mark should accept empty value: %v", err)
	}

	events = h.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	// Test nil value
	err = h.Mark("nil_value", nil)
	if err != nil {
		t.Fatalf("Mark should accept nil value: %v", err)
	}

	events = h.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestHistory_MultipleMarksSameKey(t *testing.T) {
	h := NewHistory()

	// Mark same key multiple times
	h.Mark("key1", "value1")
	time.Sleep(10 * time.Millisecond)
	h.Mark("key1", "value2")
	time.Sleep(10 * time.Millisecond)
	h.Mark("key1", "value3")

	events := h.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Verify all events have same key
	for i, event := range events {
		if event.Key() != "key1" {
			t.Errorf("Event %d: Expected key 'key1', got %q", i, event.Key())
		}
	}

	// Verify timestamps are in order
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp().Before(events[i-1].Timestamp()) {
			t.Errorf("Timestamps should be in chronological order")
		}
	}
}

func TestHistory_EventOrdering(t *testing.T) {
	h := NewHistory()

	// Add events with delays to ensure different timestamps
	h.Mark("key1", "value1")
	time.Sleep(10 * time.Millisecond)
	h.Mark("key2", "value2")
	time.Sleep(10 * time.Millisecond)
	h.Mark("key3", "value3")

	events := h.GetEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Verify timestamps are in chronological order
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp().Before(events[i-1].Timestamp()) {
			t.Errorf("Timestamps should be in chronological order")
		}
	}
}

func TestHistory_AddEventAndMark(t *testing.T) {
	h := NewHistory()

	// Add event directly
	event := Event{
		key:       "key1",
		value:     json.RawMessage(`"value1"`),
		timestamp: time.Now(),
	}
	h.AddEvent(event)

	// Mark another event
	h.Mark("key2", "value2")

	events := h.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].Key() != "key1" {
		t.Errorf("Expected first event key 'key1', got %q", events[0].Key())
	}
	if events[1].Key() != "key2" {
		t.Errorf("Expected second event key 'key2', got %q", events[1].Key())
	}
}

func TestHistory_UnmarshalJSON_InvalidData(t *testing.T) {
	h := NewHistory()

	// Test with invalid JSON
	err := h.UnmarshalJSON([]byte("invalid json"))
	if err == nil {
		t.Fatal("UnmarshalJSON should return error for invalid JSON")
	}

	// Test with empty JSON
	err = h.UnmarshalJSON([]byte("{}"))
	if err != nil {
		t.Fatalf("UnmarshalJSON should handle empty JSON: %v", err)
	}

	events := h.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected empty events after unmarshaling empty JSON, got %d", len(events))
	}
}

func TestEvent_MarshalJSON_WithSource(t *testing.T) {
	// Create event with source (though NewEvent doesn't set it)
	event := &Event{
		source:    context.NodeID("node123"),
		key:       "test_key",
		value:     json.RawMessage(`"test_value"`),
		timestamp: time.Now(),
	}

	data, err := event.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result struct {
		Source    context.NodeID  `json:"source"`
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		Timestamp time.Time       `json:"timestamp"`
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Source != "node123" {
		t.Errorf("Expected source 'node123', got %q", result.Source)
	}
	if result.Key != "test_key" {
		t.Errorf("Expected key 'test_key', got %q", result.Key)
	}
}
