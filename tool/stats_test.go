package tool

import (
	"fmt"
	"testing"
	"time"
)

func TestStats_Timers(t *testing.T) {
	stats := &Stats{
		startTime: time.Now(),
	}

	// Test Duration
	stats.Duration("test_timer", 100*time.Millisecond)
	duration := stats.GetTime("test_timer")
	if duration == nil {
		t.Fatal("GetTime should return a duration")
	}
	if *duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", *duration)
	}

	// Test Time
	stop := stats.Time("auto_timer")
	time.Sleep(10 * time.Millisecond)
	stop()

	duration = stats.GetTime("auto_timer")
	if duration == nil {
		t.Fatal("GetTime should return a duration after Time")
	}
	if *duration < 10*time.Millisecond {
		t.Errorf("Expected duration at least 10ms, got %v", *duration)
	}

	// Test overwriting timer
	stats.Duration("test_timer", 200*time.Millisecond)
	duration = stats.GetTime("test_timer")
	if duration == nil {
		t.Fatal("GetTime should return a duration")
	}
	if *duration != 200*time.Millisecond {
		t.Errorf("Expected duration 200ms, got %v", *duration)
	}

	// Test non-existent timer
	if stats.GetTime("nonexistent") != nil {
		t.Error("GetTime should return nil for non-existent timer")
	}
}

func TestStats_Counters(t *testing.T) {
	stats := &Stats{
		startTime: time.Now(),
	}

	// Test Incr
	stats.Incr("test_counter")
	count := stats.GetCount("test_counter")
	if count == nil {
		t.Fatal("GetCount should return a count")
	}
	if *count != 1 {
		t.Errorf("Expected count 1, got %d", *count)
	}

	// Test Add
	stats.Add("test_counter", 5)
	count = stats.GetCount("test_counter")
	if count == nil {
		t.Fatal("GetCount should return a count")
	}
	if *count != 6 {
		t.Errorf("Expected count 6, got %d", *count)
	}

	// Test Decr
	stats.Decr("test_counter")
	count = stats.GetCount("test_counter")
	if count == nil {
		t.Fatal("GetCount should return a count")
	}
	if *count != 5 {
		t.Errorf("Expected count 5, got %d", *count)
	}

	// Test Sub
	stats.Sub("test_counter", 2)
	count = stats.GetCount("test_counter")
	if count == nil {
		t.Fatal("GetCount should return a count")
	}
	if *count != 3 {
		t.Errorf("Expected count 3, got %d", *count)
	}

	// Test multiple counters with different names
	stats.Incr("test_counter2")
	count2 := stats.GetCount("test_counter2")
	if count2 == nil || *count2 != 1 {
		t.Errorf("Expected counter2 count 1, got %v", count2)
	}

	// Test non-existent counter
	if stats.GetCount("nonexistent") != nil {
		t.Error("GetCount should return nil for non-existent counter")
	}
}

func TestStats_Values(t *testing.T) {
	stats := &Stats{
		startTime: time.Now(),
	}

	// Test Set with int
	stats.Set("test_int", 42)
	value := stats.Get("test_int")
	if value == nil {
		t.Fatal("Get should return a value")
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %v", value)
	}

	// Test Set with int64
	stats.Set("test_int64", int64(100))
	value = stats.Get("test_int64")
	if value == nil {
		t.Fatal("Get should return a value")
	}
	if value != int64(100) {
		t.Errorf("Expected value 100, got %v", value)
	}

	// Test Set with float64
	stats.Set("test_float", 3.14)
	value = stats.Get("test_float")
	if value == nil {
		t.Fatal("Get should return a value")
	}
	if value != 3.14 {
		t.Errorf("Expected value 3.14, got %v", value)
	}

	// Test Set with string (generic support)
	stats.Set("test_string", "hello")
	value = stats.Get("test_string")
	if value == nil {
		t.Fatal("Get should return a value")
	}
	if value != "hello" {
		t.Errorf("Expected value 'hello', got %v", value)
	}

	// Test multiple values with different names
	stats.Set("value1", "value1")
	stats.Set("value2", "value2")
	stats.Set("value3", "value3")

	// Verify all values
	value1 := stats.Get("value1")
	if value1 == nil || value1 != "value1" {
		t.Errorf("Expected 'value1', got %v", value1)
	}

	value2 := stats.Get("value2")
	if value2 == nil || value2 != "value2" {
		t.Errorf("Expected 'value2', got %v", value2)
	}

	value3 := stats.Get("value3")
	if value3 == nil || value3 != "value3" {
		t.Errorf("Expected 'value3', got %v", value3)
	}

	// Test overwriting existing value (set operation)
	stats.Set("value1", "new-value1")
	value1 = stats.Get("value1")
	if value1 == nil || value1 != "new-value1" {
		t.Errorf("Expected 'new-value1', got %v", value1)
	}

	// Test non-existent value
	if stats.Get("nonexistent") != nil {
		t.Error("Get should return nil for non-existent value")
	}
}

func TestStats_ConcurrentAccess(t *testing.T) {
	stats := &Stats{
		startTime: time.Now(),
	}

	done := make(chan bool, 10)

	// Concurrent value access
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()
			name := fmt.Sprintf("concurrent_metric_%d", idx)
			stats.Set(name, idx)
			_ = stats.Get(name)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all values were set
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("concurrent_metric_%d", i)
		value := stats.Get(name)
		if value == nil || value != i {
			t.Errorf("Expected value %d for %s, got %v", i, name, value)
		}
	}
}
