package scoped_ledger

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hlfshell/gotonomy/data/ledger"
)

func TestNewScopedLedger(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "test_scope")

	if sl == nil {
		t.Fatal("NewScopedLedger should not return nil")
	}
	if sl.scope != "test_scope" {
		t.Errorf("Expected scope 'test_scope', got %q", sl.scope)
	}
	if sl.ledger != l {
		t.Error("ScopedLedger should reference the provided ledger")
	}
}

func TestScopedLedger_SetData(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test setting string value
	err := sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data was set
	entry, err := sl.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "scope1" {
		t.Errorf("Expected scope 'scope1', got %q", entry.Scope)
	}
	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entry.Key)
	}

	// Test setting different types
	err = sl.SetData("key2", 42)
	if err != nil {
		t.Fatalf("SetData failed for int: %v", err)
	}

	err = sl.SetData("key3", true)
	if err != nil {
		t.Fatalf("SetData failed for bool: %v", err)
	}

	// Test setting complex types
	type TestStruct struct {
		Name  string
		Value int
	}
	testStruct := TestStruct{Name: "test", Value: 100}
	err = sl.SetData("key4", testStruct)
	if err != nil {
		t.Fatalf("SetData failed for struct: %v", err)
	}
}

func TestScopedLedger_GetData(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test getting nonexistent key
	_, err := sl.GetData("nonexistent")
	if err == nil {
		t.Fatal("GetData should return error for nonexistent key")
	}

	// Set and get data
	err = sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	entry, err := sl.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entry.Key)
	}
	if entry.Scope != "scope1" {
		t.Errorf("Expected scope 'scope1', got %q", entry.Scope)
	}
	if entry.Operation != ledger.OperationSet {
		t.Errorf("Expected operation 'set', got %q", entry.Operation)
	}
	if entry.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}

	// Test getting deleted key
	err = sl.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	_, err = sl.GetData("key1")
	if err == nil {
		t.Fatal("GetData should return error for deleted key")
	}
}

func TestGetDataScoped(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test getting nonexistent key
	_, err := GetDataScoped[string](sl, "nonexistent")
	if err == nil {
		t.Fatal("GetDataScoped should return error for nonexistent key")
	}

	// Set and get string
	err = sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	value, err := GetDataScoped[string](sl, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}

	// Test different types
	err = sl.SetData("key2", 42)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	intValue, err := GetDataScoped[int](sl, "key2")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if intValue != 42 {
		t.Errorf("Expected value 42, got %d", intValue)
	}

	// Test complex type
	type TestStruct struct {
		Name  string
		Value int
	}
	testStruct := TestStruct{Name: "test", Value: 100}
	err = sl.SetData("key3", testStruct)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	structValue, err := GetDataScoped[TestStruct](sl, "key3")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if structValue.Name != "test" || structValue.Value != 100 {
		t.Errorf("Expected struct {Name: 'test', Value: 100}, got %+v", structValue)
	}
}

func TestScopedLedger_DeleteData(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test deleting nonexistent key (should not error)
	err := sl.DeleteData("nonexistent")
	if err != nil {
		t.Fatalf("DeleteData should not error for nonexistent key: %v", err)
	}

	// Set and delete data
	err = sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data exists
	_, err = sl.GetData("key1")
	if err != nil {
		t.Fatal("Data should exist before deletion")
	}

	// Delete data
	err = sl.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	// Verify data is deleted
	_, err = sl.GetData("key1")
	if err == nil {
		t.Fatal("GetData should return error for deleted key")
	}

	// Verify delete entry is in history
	history, err := sl.GetDataHistory("key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("Expected history length 2 (set + delete), got %d", len(history))
	}

	lastEntry := history[len(history)-1]
	if lastEntry.Operation != ledger.OperationDelete {
		t.Errorf("Expected last operation to be 'delete', got %q", lastEntry.Operation)
	}
}

func TestScopedLedger_GetDataHistory(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test getting history for nonexistent key
	_, err := sl.GetDataHistory("nonexistent")
	if err == nil {
		t.Fatal("GetDataHistory should return error for nonexistent key")
	}

	// Set data multiple times
	err = sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = sl.SetData("key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = sl.SetData("key1", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get history
	history, err := sl.GetDataHistory("key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}

	// Verify all entries have correct scope and key
	for _, entry := range history {
		if entry.Scope != "scope1" {
			t.Errorf("Expected scope 'scope1', got %q", entry.Scope)
		}
		if entry.Key != "key1" {
			t.Errorf("Expected key 'key1', got %q", entry.Key)
		}
		if entry.Operation != ledger.OperationSet {
			t.Errorf("Expected operation 'set', got %q", entry.Operation)
		}
	}

	// Verify timestamps are in order
	for i := 1; i < len(history); i++ {
		if history[i].Timestamp.Before(history[i-1].Timestamp) {
			t.Errorf("Timestamps should be in chronological order")
		}
	}
}

func TestGetDataHistoryScoped(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test getting history for nonexistent key
	_, err := GetDataHistoryScoped[string](sl, "nonexistent")
	if err == nil {
		t.Fatal("GetDataHistoryScoped should return error for nonexistent key")
	}

	// Set data multiple times
	err = sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = sl.SetData("key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = sl.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = sl.SetData("key1", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get typed history (should skip deleted entries)
	history, err := GetDataHistoryScoped[string](sl, "key1")
	if err != nil {
		t.Fatalf("GetDataHistoryScoped failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("Expected history length 3 (deleted entry should be skipped), got %d", len(history))
	}

	if history[0] != "value1" {
		t.Errorf("Expected first value 'value1', got %q", history[0])
	}
	if history[1] != "value2" {
		t.Errorf("Expected second value 'value2', got %q", history[1])
	}
	if history[2] != "value3" {
		t.Errorf("Expected third value 'value3', got %q", history[2])
	}
}

func TestSetDataFunc(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test SetDataFunc on nonexistent key
	err := SetDataFunc[int](sl, "nonexistent", func(v int) (int, error) {
		return v + 1, nil
	})
	if err == nil {
		t.Fatal("SetDataFunc should return error for nonexistent key")
	}

	// Set initial value
	err = sl.SetData("key1", 10)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Use SetDataFunc to increment
	err = SetDataFunc[int](sl, "key1", func(v int) (int, error) {
		return v + 5, nil
	})
	if err != nil {
		t.Fatalf("SetDataFunc failed: %v", err)
	}

	// Verify new value
	value, err := GetDataScoped[int](sl, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected value 15, got %d", value)
	}

	// Test with error in function
	err = SetDataFunc[int](sl, "key1", func(v int) (int, error) {
		return 0, json.Unmarshal([]byte("invalid"), &v)
	})
	if err == nil {
		t.Fatal("SetDataFunc should return error when function returns error")
	}

	// Verify value wasn't changed
	value, err = GetDataScoped[int](sl, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected value to remain 15, got %d", value)
	}
}

func TestScopedLedger_GetKeys(t *testing.T) {
	l := ledger.NewLedger()
	sl1 := NewScopedLedger(l, "scope1")
	sl2 := NewScopedLedger(l, "scope2")

	// Test empty scope
	keys := sl1.GetKeys()
	if len(keys) != 0 {
		t.Errorf("Expected empty keys, got %v", keys)
	}

	// Set data in scope1
	err := sl1.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = sl1.SetData("key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Set data in scope2
	err = sl2.SetData("key3", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get keys for scope1
	keys = sl1.GetKeys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Verify keys are correct (order may vary)
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	if !keyMap["key1"] {
		t.Error("Expected key1 to be present")
	}
	if !keyMap["key2"] {
		t.Error("Expected key2 to be present")
	}

	// Get keys for scope2
	keys = sl2.GetKeys()
	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}
	if keys[0] != "key3" {
		t.Errorf("Expected key 'key3', got %q", keys[0])
	}

	// Test that deleted keys still appear in GetKeys
	err = sl1.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	keys = sl1.GetKeys()
	keyMap = make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	if !keyMap["key1"] {
		t.Error("Deleted key should still appear in GetKeys")
	}
	if !keyMap["key2"] {
		t.Error("Expected key2 to be present")
	}
}

func TestScopedLedger_ScopeWithColon(t *testing.T) {
	l := ledger.NewLedger()
	scope := "parent:child"
	sl := NewScopedLedger(l, scope)

	// Set data using a scope that contains ":".
	err := sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = sl.SetData("key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Ensure underlying ledger reports scopes with ":" correctly.
	allScopes := l.GetScopes()
	scopeMap := make(map[string]bool)
	for _, s := range allScopes {
		scopeMap[s] = true
	}
	if !scopeMap[scope] {
		t.Fatalf("Expected scope %q to be present in underlying ledger", scope)
	}

	// ScopedLedger should still see keys correctly for its scope.
	keys := sl.GetKeys()
	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys for scoped ledger, got %d", len(keys))
	}
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	if !keyMap["key1"] || !keyMap["key2"] {
		t.Fatalf("Expected keys 'key1' and 'key2' for scoped ledger with ':' in scope, got %v", keys)
	}

	// And GetData should return entries with the original scope string.
	entry, err := sl.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != scope {
		t.Errorf("Expected entry scope %q, got %q", scope, entry.Scope)
	}
}

func TestScopedLedger_MarshalJSON(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test empty scope
	data, err := sl.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result map[string][]ledger.Entry
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}

	// Set data
	err = sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = sl.SetData("key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Marshal
	data, err = sl.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal and verify
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(result))
	}

	// Verify key1
	entries1, ok := result["key1"]
	if !ok {
		t.Fatal("Expected key1 to be present")
	}
	if len(entries1) != 1 {
		t.Errorf("Expected 1 entry for key1, got %d", len(entries1))
	}
	if entries1[0].Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entries1[0].Key)
	}
	if entries1[0].Scope != "scope1" {
		t.Errorf("Expected scope 'scope1', got %q", entries1[0].Scope)
	}

	// Verify key2
	entries2, ok := result["key2"]
	if !ok {
		t.Fatal("Expected key2 to be present")
	}
	if len(entries2) != 1 {
		t.Errorf("Expected 1 entry for key2, got %d", len(entries2))
	}
}

func TestScopedLedger_UnmarshalJSON(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test unmarshaling (should return error)
	data := []byte(`{"key1": [{"key": "key1", "value": "value1"}]}`)
	err := sl.UnmarshalJSON(data)
	if err == nil {
		t.Fatal("UnmarshalJSON should return error")
	}
}

func TestScopedLedger_ScopeIsolation(t *testing.T) {
	l := ledger.NewLedger()
	sl1 := NewScopedLedger(l, "scope1")
	sl2 := NewScopedLedger(l, "scope2")

	// Set data in scope1
	err := sl1.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify scope2 cannot see scope1's data
	_, err = sl2.GetData("key1")
	if err == nil {
		t.Fatal("Scope2 should not be able to access scope1's data")
	}

	// Set same key in scope2
	err = sl2.SetData("key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify both scopes have their own data
	value1, err := GetDataScoped[string](sl1, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Expected scope1 value 'value1', got %q", value1)
	}

	value2, err := GetDataScoped[string](sl2, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Expected scope2 value 'value2', got %q", value2)
	}

	// Verify GetKeys returns only scope-specific keys
	keys1 := sl1.GetKeys()
	if len(keys1) != 1 || keys1[0] != "key1" {
		t.Errorf("Expected scope1 to have only key1, got %v", keys1)
	}

	keys2 := sl2.GetKeys()
	if len(keys2) != 1 || keys2[0] != "key1" {
		t.Errorf("Expected scope2 to have only key1, got %v", keys2)
	}
}

func TestScopedLedger_ComplexTypes(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test with struct
	type TestStruct struct {
		Name  string
		Value int
		Tags  []string
	}
	testStruct := TestStruct{Name: "test", Value: 42, Tags: []string{"tag1", "tag2"}}
	err := sl.SetData("struct_key", testStruct)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	value, err := GetDataScoped[TestStruct](sl, "struct_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value.Name != "test" || value.Value != 42 || len(value.Tags) != 2 {
		t.Errorf("Expected struct {Name: 'test', Value: 42, Tags: ['tag1', 'tag2']}, got %+v", value)
	}

	// Test with slice
	slice := []int{1, 2, 3, 4, 5}
	err = sl.SetData("slice_key", slice)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	valueSlice, err := GetDataScoped[[]int](sl, "slice_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if len(valueSlice) != 5 {
		t.Errorf("Expected slice length 5, got %d", len(valueSlice))
	}

	// Test with map
	testMap := map[string]interface{}{"key1": "value1", "key2": 42, "key3": true}
	err = sl.SetData("map_key", testMap)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	valueMap, err := GetDataScoped[map[string]interface{}](sl, "map_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if valueMap["key1"] != "value1" {
		t.Errorf("Expected map value 'value1', got %v", valueMap["key1"])
	}
	if valueMap["key2"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("Expected map value 42, got %v", valueMap["key2"])
	}
}

func TestScopedLedger_ConcurrentAccess(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			err := sl.SetData("concurrent_key", id)
			if err != nil {
				t.Errorf("SetData failed: %v", err)
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
			_, _ = GetDataScoped[int](sl, "concurrent_key")
			_, _ = sl.GetDataHistory("concurrent_key")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	history, err := sl.GetDataHistory("concurrent_key")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 10 {
		t.Errorf("Expected 10 entries in history, got %d", len(history))
	}
}

func TestScopedLedger_EdgeCases(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Test empty key
	err := sl.SetData("", "value")
	if err != nil {
		t.Fatalf("SetData should accept empty key: %v", err)
	}

	value, err := GetDataScoped[string](sl, "")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got %q", value)
	}

	// Test empty value
	err = sl.SetData("empty_key", "")
	if err != nil {
		t.Fatalf("SetData should accept empty value: %v", err)
	}

	value, err = GetDataScoped[string](sl, "empty_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty value, got %q", value)
	}

	// Test nil value
	err = sl.SetData("nil_key", nil)
	if err != nil {
		t.Fatalf("SetData should accept nil value: %v", err)
	}

	// Test very long key
	longKey := string(make([]byte, 1000))
	err = sl.SetData(longKey, "value")
	if err != nil {
		t.Fatalf("SetData should accept long key: %v", err)
	}

	// Test special characters in key
	err = sl.SetData("key:with:colons", "value")
	if err != nil {
		t.Fatalf("SetData should accept key with colons: %v", err)
	}

	value, err = GetDataScoped[string](sl, "key:with:colons")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got %q", value)
	}
}

func TestScopedLedger_DeleteAndRecreate(t *testing.T) {
	l := ledger.NewLedger()
	sl := NewScopedLedger(l, "scope1")

	// Set, delete, and recreate
	err := sl.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = sl.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	err = sl.SetData("key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify we can get the new value
	value, err := GetDataScoped[string](sl, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != "value2" {
		t.Errorf("Expected value 'value2', got %q", value)
	}

	// Verify history contains all operations
	history, err := sl.GetDataHistory("key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}
	if history[0].Operation != ledger.OperationSet {
		t.Errorf("Expected first operation 'set', got %q", history[0].Operation)
	}
	if history[1].Operation != ledger.OperationDelete {
		t.Errorf("Expected second operation 'delete', got %q", history[1].Operation)
	}
	if history[2].Operation != ledger.OperationSet {
		t.Errorf("Expected third operation 'set', got %q", history[2].Operation)
	}
}

func TestScopedLedger_Subscoped_Basic(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	child := parent.Subscoped("child")

	// Verify child has correct scope
	if child.scope != "parent:child" {
		t.Errorf("Expected scope 'parent:child', got %q", child.scope)
	}

	// Verify child uses same underlying ledger
	if child.ledger != l {
		t.Error("Subscoped ledger should reference the same underlying ledger")
	}

	// Set data in child scope
	err := child.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data is stored with correct scope
	entry, err := child.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "parent:child" {
		t.Errorf("Expected entry scope 'parent:child', got %q", entry.Scope)
	}

	// Verify parent cannot see child's data
	_, err = parent.GetData("key1")
	if err == nil {
		t.Fatal("Parent scope should not be able to access child scope's data")
	}

	// Verify child can see its own data
	value, err := GetDataScoped[string](child, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}
}

func TestScopedLedger_Subscoped_Nested(t *testing.T) {
	l := ledger.NewLedger()
	root := NewScopedLedger(l, "root")
	level1 := root.Subscoped("level1")
	level2 := level1.Subscoped("level2")
	level3 := level2.Subscoped("level3")

	// Verify nested scopes
	if level1.scope != "root:level1" {
		t.Errorf("Expected scope 'root:level1', got %q", level1.scope)
	}
	if level2.scope != "root:level1:level2" {
		t.Errorf("Expected scope 'root:level1:level2', got %q", level2.scope)
	}
	if level3.scope != "root:level1:level2:level3" {
		t.Errorf("Expected scope 'root:level1:level2:level3', got %q", level3.scope)
	}

	// Set data at each level
	err := root.SetData("root_key", "root_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = level1.SetData("level1_key", "level1_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = level2.SetData("level2_key", "level2_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = level3.SetData("level3_key", "level3_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify each level can only see its own data
	rootValue, err := GetDataScoped[string](root, "root_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if rootValue != "root_value" {
		t.Errorf("Expected 'root_value', got %q", rootValue)
	}

	level1Value, err := GetDataScoped[string](level1, "level1_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if level1Value != "level1_value" {
		t.Errorf("Expected 'level1_value', got %q", level1Value)
	}

	level2Value, err := GetDataScoped[string](level2, "level2_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if level2Value != "level2_value" {
		t.Errorf("Expected 'level2_value', got %q", level2Value)
	}

	level3Value, err := GetDataScoped[string](level3, "level3_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if level3Value != "level3_value" {
		t.Errorf("Expected 'level3_value', got %q", level3Value)
	}

	// Verify levels cannot see each other's data
	_, err = root.GetData("level1_key")
	if err == nil {
		t.Fatal("Root should not see level1's data")
	}

	_, err = level1.GetData("level2_key")
	if err == nil {
		t.Fatal("Level1 should not see level2's data")
	}

	_, err = level2.GetData("level3_key")
	if err == nil {
		t.Fatal("Level2 should not see level3's data")
	}
}

func TestScopedLedger_Subscoped_Isolation(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	child1 := parent.Subscoped("child1")
	child2 := parent.Subscoped("child2")

	// Set data in each child with same key
	err := child1.SetData("shared_key", "child1_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = child2.SetData("shared_key", "child2_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify each child has its own value
	value1, err := GetDataScoped[string](child1, "shared_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value1 != "child1_value" {
		t.Errorf("Expected 'child1_value', got %q", value1)
	}

	value2, err := GetDataScoped[string](child2, "shared_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value2 != "child2_value" {
		t.Errorf("Expected 'child2_value', got %q", value2)
	}

	// Verify parent cannot see either child's data
	_, err = parent.GetData("shared_key")
	if err == nil {
		t.Fatal("Parent should not see child data")
	}

	// Verify children cannot see each other's data
	_, err = child1.GetData("shared_key")
	if err != nil {
		t.Fatalf("Child1 should see its own data: %v", err)
	}
	// child1's GetData will work, but GetDataScoped from child1 won't see child2's data
	// because they're in different scopes. The underlying ledger has both entries,
	// but each scoped ledger only sees its own scope.
}

func TestScopedLedger_Subscoped_GetKeys(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	child := parent.Subscoped("child")

	// Set data in parent
	err := parent.SetData("parent_key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = parent.SetData("parent_key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Set data in child
	err = child.SetData("child_key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = child.SetData("child_key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify GetKeys returns only scope-specific keys
	parentKeys := parent.GetKeys()
	if len(parentKeys) != 2 {
		t.Errorf("Expected 2 keys in parent, got %d", len(parentKeys))
	}
	parentKeyMap := make(map[string]bool)
	for _, k := range parentKeys {
		parentKeyMap[k] = true
	}
	if !parentKeyMap["parent_key1"] || !parentKeyMap["parent_key2"] {
		t.Errorf("Parent keys should only contain parent_key1 and parent_key2, got %v", parentKeys)
	}

	childKeys := child.GetKeys()
	if len(childKeys) != 2 {
		t.Errorf("Expected 2 keys in child, got %d", len(childKeys))
	}
	childKeyMap := make(map[string]bool)
	for _, k := range childKeys {
		childKeyMap[k] = true
	}
	if !childKeyMap["child_key1"] || !childKeyMap["child_key2"] {
		t.Errorf("Child keys should only contain child_key1 and child_key2, got %v", childKeys)
	}
}

func TestScopedLedger_Subscoped_WithPrefix(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	
	// Test that if subScope already contains the parent scope prefix,
	// it uses the subScope as-is (avoids double prefix)
	child := parent.Subscoped("parent:child")
	
	if child.scope != "parent:child" {
		t.Errorf("Expected scope 'parent:child' (no double prefix), got %q", child.scope)
	}

	// Set data and verify it works
	err := child.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	entry, err := child.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "parent:child" {
		t.Errorf("Expected entry scope 'parent:child', got %q", entry.Scope)
	}
}

func TestScopedLedger_Subscoped_EmptySubScope(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	
	// Test with empty subscope
	child := parent.Subscoped("")
	
	// Empty subscope should create "parent:" scope
	if child.scope != "parent:" {
		t.Errorf("Expected scope 'parent:' for empty subscope, got %q", child.scope)
	}

	// Set data and verify it works
	err := child.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	entry, err := child.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "parent:" {
		t.Errorf("Expected entry scope 'parent:', got %q", entry.Scope)
	}
}

func TestScopedLedger_Subscoped_DataHistory(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	child := parent.Subscoped("child")

	// Set data multiple times in child
	err := child.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = child.SetData("key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get history from child
	history, err := child.GetDataHistory("key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("Expected 2 entries in history, got %d", len(history))
	}

	// Verify all entries have correct scope
	for _, entry := range history {
		if entry.Scope != "parent:child" {
			t.Errorf("Expected entry scope 'parent:child', got %q", entry.Scope)
		}
	}

	// Get typed history
	typedHistory, err := GetDataHistoryScoped[string](child, "key1")
	if err != nil {
		t.Fatalf("GetDataHistoryScoped failed: %v", err)
	}
	if len(typedHistory) != 2 {
		t.Errorf("Expected 2 entries in typed history, got %d", len(typedHistory))
	}
	if typedHistory[0] != "value1" || typedHistory[1] != "value2" {
		t.Errorf("Expected history ['value1', 'value2'], got %v", typedHistory)
	}
}

func TestScopedLedger_Subscoped_DeleteData(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	child := parent.Subscoped("child")

	// Set and delete data in child
	err := child.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = child.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	// Verify deletion
	_, err = child.GetData("key1")
	if err == nil {
		t.Fatal("GetData should return error for deleted key")
	}

	// Verify history includes delete operation
	history, err := child.GetDataHistory("key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("Expected 2 entries (set + delete), got %d", len(history))
	}
	if history[1].Operation != ledger.OperationDelete {
		t.Errorf("Expected last operation 'delete', got %q", history[1].Operation)
	}
	if history[1].Scope != "parent:child" {
		t.Errorf("Expected delete entry scope 'parent:child', got %q", history[1].Scope)
	}
}

func TestScopedLedger_Subscoped_SetDataFunc(t *testing.T) {
	l := ledger.NewLedger()
	parent := NewScopedLedger(l, "parent")
	child := parent.Subscoped("child")

	// Set initial value in child
	err := child.SetData("key1", 10)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Use SetDataFunc to modify value
	err = SetDataFunc[int](child, "key1", func(v int) (int, error) {
		return v + 5, nil
	})
	if err != nil {
		t.Fatalf("SetDataFunc failed: %v", err)
	}

	// Verify new value
	value, err := GetDataScoped[int](child, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected value 15, got %d", value)
	}

	// Verify entry has correct scope
	entry, err := child.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "parent:child" {
		t.Errorf("Expected entry scope 'parent:child', got %q", entry.Scope)
	}
}

func TestScopedLedger_Subscoped_ChainedPrefixDetection(t *testing.T) {
	l := ledger.NewLedger()
	root := NewScopedLedger(l, "root")
	level1 := root.Subscoped("level1")
	
	// Test 1: Creating a subscope from a subscoped ledger with full prefix
	// This should detect the prefix and not double it
	level2a := level1.Subscoped("root:level1:level2")
	if level2a.scope != "root:level1:level2" {
		t.Errorf("Expected scope 'root:level1:level2' (no double prefix), got %q", level2a.scope)
	}
	
	// Test 2: Creating a subscope from a subscoped ledger with just the subscope name
	// This should append normally
	level2b := level1.Subscoped("level2")
	if level2b.scope != "root:level1:level2" {
		t.Errorf("Expected scope 'root:level1:level2', got %q", level2b.scope)
	}
	
	// Test 3: Creating a subscope with partial prefix (root: but not root:level1:)
	// This should still append, creating root:level1:root:something
	level2c := level1.Subscoped("root:something")
	if level2c.scope != "root:level1:root:something" {
		t.Errorf("Expected scope 'root:level1:root:something', got %q", level2c.scope)
	}
	
	// Test 4: Deep nesting with prefix detection
	level3 := level2a.Subscoped("level3")
	if level3.scope != "root:level1:level2:level3" {
		t.Errorf("Expected scope 'root:level1:level2:level3', got %q", level3.scope)
	}
	
	// Test 5: Deep nesting with full prefix
	level4 := level3.Subscoped("root:level1:level2:level3:level4")
	if level4.scope != "root:level1:level2:level3:level4" {
		t.Errorf("Expected scope 'root:level1:level2:level3:level4' (no double prefix), got %q", level4.scope)
	}
}

func TestScopedLedger_Subscoped_ChainedDataIsolation(t *testing.T) {
	l := ledger.NewLedger()
	root := NewScopedLedger(l, "root")
	level1 := root.Subscoped("level1")
	level2 := level1.Subscoped("level2")
	level3 := level2.Subscoped("level3")
	
	// Set data at each level with the same key name
	err := root.SetData("shared_key", "root_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	err = level1.SetData("shared_key", "level1_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	err = level2.SetData("shared_key", "level2_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	err = level3.SetData("shared_key", "level3_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	// Verify each level sees only its own value
	rootValue, err := GetDataScoped[string](root, "shared_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if rootValue != "root_value" {
		t.Errorf("Expected root 'root_value', got %q", rootValue)
	}
	
	level1Value, err := GetDataScoped[string](level1, "shared_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if level1Value != "level1_value" {
		t.Errorf("Expected level1 'level1_value', got %q", level1Value)
	}
	
	level2Value, err := GetDataScoped[string](level2, "shared_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if level2Value != "level2_value" {
		t.Errorf("Expected level2 'level2_value', got %q", level2Value)
	}
	
	level3Value, err := GetDataScoped[string](level3, "shared_key")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if level3Value != "level3_value" {
		t.Errorf("Expected level3 'level3_value', got %q", level3Value)
	}
	
	// Verify GetKeys returns correct keys for each level
	rootKeys := root.GetKeys()
	if len(rootKeys) != 1 || rootKeys[0] != "shared_key" {
		t.Errorf("Expected root to have only 'shared_key', got %v", rootKeys)
	}
	
	level1Keys := level1.GetKeys()
	if len(level1Keys) != 1 || level1Keys[0] != "shared_key" {
		t.Errorf("Expected level1 to have only 'shared_key', got %v", level1Keys)
	}
	
	level2Keys := level2.GetKeys()
	if len(level2Keys) != 1 || level2Keys[0] != "shared_key" {
		t.Errorf("Expected level2 to have only 'shared_key', got %v", level2Keys)
	}
	
	level3Keys := level3.GetKeys()
	if len(level3Keys) != 1 || level3Keys[0] != "shared_key" {
		t.Errorf("Expected level3 to have only 'shared_key', got %v", level3Keys)
	}
}

func TestScopedLedger_Subscoped_ChainedOperations(t *testing.T) {
	l := ledger.NewLedger()
	root := NewScopedLedger(l, "root")
	level1 := root.Subscoped("level1")
	level2 := level1.Subscoped("level2")
	
	// Test SetData in deeply nested scope
	err := level2.SetData("key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	// Verify entry has correct scope
	entry, err := level2.GetData("key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "root:level1:level2" {
		t.Errorf("Expected entry scope 'root:level1:level2', got %q", entry.Scope)
	}
	
	// Test DeleteData in deeply nested scope
	err = level2.DeleteData("key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}
	
	// Verify deletion
	_, err = level2.GetData("key1")
	if err == nil {
		t.Fatal("GetData should return error for deleted key")
	}
	
	// Test SetDataFunc in deeply nested scope
	err = level2.SetData("key2", 10)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	err = SetDataFunc[int](level2, "key2", func(v int) (int, error) {
		return v + 5, nil
	})
	if err != nil {
		t.Fatalf("SetDataFunc failed: %v", err)
	}
	
	value, err := GetDataScoped[int](level2, "key2")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected value 15, got %d", value)
	}
	
	// Test GetDataHistory in deeply nested scope
	history, err := level2.GetDataHistory("key2")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("Expected 2 entries in history, got %d", len(history))
	}
	
	// Verify all entries have correct scope
	for _, e := range history {
		if e.Scope != "root:level1:level2" {
			t.Errorf("Expected entry scope 'root:level1:level2', got %q", e.Scope)
		}
	}
}

func TestScopedLedger_Subscoped_ChainedSiblingIsolation(t *testing.T) {
	l := ledger.NewLedger()
	root := NewScopedLedger(l, "root")
	
	// Create two separate branches from root
	branch1_level1 := root.Subscoped("branch1")
	branch1_level2 := branch1_level1.Subscoped("level2")
	
	branch2_level1 := root.Subscoped("branch2")
	branch2_level2 := branch2_level1.Subscoped("level2")
	
	// Set data in each branch with same key
	err := branch1_level2.SetData("key1", "branch1_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	err = branch2_level2.SetData("key1", "branch2_value")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	
	// Verify each branch sees only its own data
	value1, err := GetDataScoped[string](branch1_level2, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value1 != "branch1_value" {
		t.Errorf("Expected 'branch1_value', got %q", value1)
	}
	
	value2, err := GetDataScoped[string](branch2_level2, "key1")
	if err != nil {
		t.Fatalf("GetDataScoped failed: %v", err)
	}
	if value2 != "branch2_value" {
		t.Errorf("Expected 'branch2_value', got %q", value2)
	}
	
	// Verify branches cannot see each other's data
	_, err = branch1_level2.GetData("key1")
	if err != nil {
		t.Fatalf("branch1_level2 should see its own data: %v", err)
	}
	
	// Verify scopes are different
	if branch1_level2.scope == branch2_level2.scope {
		t.Error("Branch scopes should be different")
	}
	if branch1_level2.scope != "root:branch1:level2" {
		t.Errorf("Expected branch1 scope 'root:branch1:level2', got %q", branch1_level2.scope)
	}
	if branch2_level2.scope != "root:branch2:level2" {
		t.Errorf("Expected branch2 scope 'root:branch2:level2', got %q", branch2_level2.scope)
	}
}

