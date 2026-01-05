package ledger

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewLedger(t *testing.T) {
	l := NewLedger()
	if l == nil {
		t.Fatal("NewLedger should not return nil")
	}
	if l.data == nil {
		t.Fatal("Ledger data map should be initialized")
	}
}

func TestLedger_SetData(t *testing.T) {
	l := NewLedger()

	// Test setting string value
	err := l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data was set
	entry, err := l.GetData("scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Scope != "scope1" {
		t.Errorf("Expected scope 'scope1', got %q", entry.Scope)
	}
	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entry.Key)
	}
	if entry.Operation != OperationSet {
		t.Errorf("Expected operation 'set', got %q", entry.Operation)
	}
	if entry.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}

	// Test setting different types
	err = l.SetData("scope1", "key2", 42)
	if err != nil {
		t.Fatalf("SetData failed for int: %v", err)
	}

	err = l.SetData("scope1", "key3", true)
	if err != nil {
		t.Fatalf("SetData failed for bool: %v", err)
	}

	// Test setting complex types
	type TestStruct struct {
		Name  string
		Value int
	}
	testStruct := TestStruct{Name: "test", Value: 100}
	err = l.SetData("scope1", "key4", testStruct)
	if err != nil {
		t.Fatalf("SetData failed for struct: %v", err)
	}
}

func TestLedger_GetData(t *testing.T) {
	l := NewLedger()

	// Test getting nonexistent key
	_, err := l.GetData("scope1", "nonexistent")
	if err == nil {
		t.Fatal("GetData should return error for nonexistent key")
	}

	// Set and get data
	err = l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	entry, err := l.GetData("scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entry.Key)
	}
	if entry.Scope != "scope1" {
		t.Errorf("Expected scope 'scope1', got %q", entry.Scope)
	}

	// Test getting deleted key
	err = l.DeleteData("scope1", "key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	_, err = l.GetData("scope1", "key1")
	if err == nil {
		t.Fatal("GetData should return error for deleted key")
	}
}

func TestGetData(t *testing.T) {
	l := NewLedger()

	// Test getting nonexistent key
	_, err := GetData[string](l, "scope1", "nonexistent")
	if err == nil {
		t.Fatal("GetData should return error for nonexistent key")
	}

	// Set and get string
	err = l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	value, err := GetData[string](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}

	// Test different types
	err = l.SetData("scope1", "key2", 42)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	intValue, err := GetData[int](l, "scope1", "key2")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
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
	err = l.SetData("scope1", "key3", testStruct)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	structValue, err := GetData[TestStruct](l, "scope1", "key3")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if structValue.Name != "test" || structValue.Value != 100 {
		t.Errorf("Expected struct {Name: 'test', Value: 100}, got %+v", structValue)
	}
}

func TestLedger_DeleteData(t *testing.T) {
	l := NewLedger()

	// Test deleting nonexistent key (should not error)
	err := l.DeleteData("scope1", "nonexistent")
	if err != nil {
		t.Fatalf("DeleteData should not error for nonexistent key: %v", err)
	}

	// Set and delete data
	err = l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify data exists
	_, err = l.GetData("scope1", "key1")
	if err != nil {
		t.Fatal("Data should exist before deletion")
	}

	// Delete data
	err = l.DeleteData("scope1", "key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	// Verify data is deleted
	_, err = l.GetData("scope1", "key1")
	if err == nil {
		t.Fatal("GetData should return error for deleted key")
	}

	// Verify delete entry is in history
	history, err := l.GetDataHistory("scope1", "key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("Expected history length 2 (set + delete), got %d", len(history))
	}

	lastEntry := history[len(history)-1]
	if lastEntry.Operation != OperationDelete {
		t.Errorf("Expected last operation to be 'delete', got %q", lastEntry.Operation)
	}
}

func TestLedger_GetDataHistory(t *testing.T) {
	l := NewLedger()

	// Test getting history for nonexistent key
	_, err := l.GetDataHistory("scope1", "nonexistent")
	if err == nil {
		t.Fatal("GetDataHistory should return error for nonexistent key")
	}

	// Set data multiple times
	err = l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = l.SetData("scope1", "key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = l.SetData("scope1", "key1", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get history
	history, err := l.GetDataHistory("scope1", "key1")
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
		if entry.Operation != OperationSet {
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

func TestGetDataHistory(t *testing.T) {
	l := NewLedger()

	// Test getting history for nonexistent key
	_, err := GetDataHistory[string](l, "scope1", "nonexistent")
	if err == nil {
		t.Fatal("GetDataHistory should return error for nonexistent key")
	}

	// Set data multiple times
	err = l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = l.SetData("scope1", "key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = l.DeleteData("scope1", "key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = l.SetData("scope1", "key1", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get typed history (should skip deleted entries)
	history, err := GetDataHistory[string](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
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

func TestLedger_SetDataFunc(t *testing.T) {
	l := NewLedger()

	// Test SetDataFunc on nonexistent key
	err := SetDataFunc[int](l, "scope1", "nonexistent", func(v int) (int, error) {
		return v + 1, nil
	})
	if err == nil {
		t.Fatal("SetDataFunc should return error for nonexistent key")
	}

	// Set initial value
	err = l.SetData("scope1", "key1", 10)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Use SetDataFunc to increment
	err = SetDataFunc[int](l, "scope1", "key1", func(v int) (int, error) {
		return v + 5, nil
	})
	if err != nil {
		t.Fatalf("SetDataFunc failed: %v", err)
	}

	// Verify new value
	value, err := GetData[int](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected value 15, got %d", value)
	}

	// Test with error in function
	err = SetDataFunc[int](l, "scope1", "key1", func(v int) (int, error) {
		return 0, json.Unmarshal([]byte("invalid"), &v)
	})
	if err == nil {
		t.Fatal("SetDataFunc should return error when function returns error")
	}

	// Verify value wasn't changed
	value, err = GetData[int](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected value to remain 15, got %d", value)
	}
}

func TestLedger_GetScopes(t *testing.T) {
	l := NewLedger()

	// Test empty ledger
	scopes := l.GetScopes()
	if len(scopes) != 0 {
		t.Errorf("Expected empty scopes, got %v", scopes)
	}

	// Set data in multiple scopes
	err := l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope1", "key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope2", "key3", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope3", "key4", "value4")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get scopes
	scopes = l.GetScopes()
	if len(scopes) != 3 {
		t.Errorf("Expected 3 scopes, got %d", len(scopes))
	}

	// Verify scopes are correct (order may vary)
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[scope] = true
	}
	if !scopeMap["scope1"] {
		t.Error("Expected scope1 to be present")
	}
	if !scopeMap["scope2"] {
		t.Error("Expected scope2 to be present")
	}
	if !scopeMap["scope3"] {
		t.Error("Expected scope3 to be present")
	}
}

func TestLedger_GetScopes_WithColonInScope(t *testing.T) {
	l := NewLedger()

	// Scopes that themselves contain ":" should be handled correctly.
	err := l.SetData("parent:child", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("parent:child:grandchild", "key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	scopes := l.GetScopes()
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	if !scopeMap["parent:child"] {
		t.Error("Expected scope 'parent:child' to be present")
	}
	if !scopeMap["parent:child:grandchild"] {
		t.Error("Expected scope 'parent:child:grandchild' to be present")
	}
}

func TestLedger_GetKeys(t *testing.T) {
	l := NewLedger()

	// Test empty ledger
	keys := l.GetKeys()
	if len(keys) != 0 {
		t.Errorf("Expected empty keys, got %v", keys)
	}

	// Set data in multiple scopes
	err := l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope1", "key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope2", "key3", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope2", "key4", "value4")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope2", "key5", "value5")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Get keys
	keys = l.GetKeys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(keys))
	}

	// Verify scope1 keys
	scope1KeysRaw, ok := keys["scope1"]
	if !ok {
		t.Fatal("Expected scope1 to be present")
	}
	scope1Keys, ok := scope1KeysRaw.([]string)
	if !ok {
		t.Fatalf("Expected scope1 keys to be []string, got %T", scope1KeysRaw)
	}
	if len(scope1Keys) != 2 {
		t.Errorf("Expected 2 keys for scope1, got %d", len(scope1Keys))
	}
	keyMap := make(map[string]bool)
	for _, key := range scope1Keys {
		keyMap[key] = true
	}
	if !keyMap["key1"] {
		t.Error("Expected key1 to be present in scope1")
	}
	if !keyMap["key2"] {
		t.Error("Expected key2 to be present in scope1")
	}

	// Verify scope2 keys
	scope2KeysRaw, ok := keys["scope2"]
	if !ok {
		t.Fatal("Expected scope2 to be present")
	}
	scope2Keys, ok := scope2KeysRaw.([]string)
	if !ok {
		t.Fatalf("Expected scope2 keys to be []string, got %T", scope2KeysRaw)
	}
	if len(scope2Keys) != 3 {
		t.Errorf("Expected 3 keys for scope2, got %d", len(scope2Keys))
	}
	keyMap = make(map[string]bool)
	for _, key := range scope2Keys {
		keyMap[key] = true
	}
	if !keyMap["key3"] {
		t.Error("Expected key3 to be present in scope2")
	}
	if !keyMap["key4"] {
		t.Error("Expected key4 to be present in scope2")
	}
	if !keyMap["key5"] {
		t.Error("Expected key5 to be present in scope2")
	}

	// Test that deleted keys still appear in GetKeys
	err = l.DeleteData("scope1", "key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	keys = l.GetKeys()
	scope1KeysRaw, ok = keys["scope1"]
	if !ok {
		t.Fatal("Expected scope1 to be present")
	}
	scope1Keys, ok = scope1KeysRaw.([]string)
	if !ok {
		t.Fatalf("Expected scope1 keys to be []string, got %T", scope1KeysRaw)
	}
	keyMap = make(map[string]bool)
	for _, key := range scope1Keys {
		keyMap[key] = true
	}
	if !keyMap["key1"] {
		t.Error("Deleted key should still appear in GetKeys")
	}
	if !keyMap["key2"] {
		t.Error("Expected key2 to be present")
	}
}

func TestLedger_GetKeys_WithColonInScope(t *testing.T) {
	l := NewLedger()

	// Use a scope that contains ":" and ensure keys are grouped correctly.
	scope := "parent:child"
	err := l.SetData(scope, "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData(scope, "key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	keys := l.GetKeys()
	scopeKeysRaw, ok := keys[scope]
	if !ok {
		t.Fatalf("Expected scope %q to be present", scope)
	}

	scopeKeys, ok := scopeKeysRaw.([]string)
	if !ok {
		t.Fatalf("Expected scope keys to be []string, got %T", scopeKeysRaw)
	}

	if len(scopeKeys) != 2 {
		t.Errorf("Expected 2 keys for scope %s, got %d", scope, len(scopeKeys))
	}

	keyMap := make(map[string]bool)
	for _, key := range scopeKeys {
		keyMap[key] = true
	}
	if !keyMap["key1"] {
		t.Error("Expected key1 to be present for scope with ':'")
	}
	if !keyMap["key2"] {
		t.Error("Expected key2 to be present for scope with ':'")
	}
}

func TestLedger_ScopeIsolation(t *testing.T) {
	l := NewLedger()

	// Set data in scope1
	err := l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify scope2 cannot see scope1's data
	_, err = l.GetData("scope2", "key1")
	if err == nil {
		t.Fatal("Scope2 should not be able to access scope1's data")
	}

	// Set same key in scope2
	err = l.SetData("scope2", "key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify both scopes have their own data
	value1, err := GetData[string](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Expected scope1 value 'value1', got %q", value1)
	}

	value2, err := GetData[string](l, "scope2", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Expected scope2 value 'value2', got %q", value2)
	}
}

func TestLedger_ComplexTypes(t *testing.T) {
	l := NewLedger()

	// Test with struct
	type TestStruct struct {
		Name  string
		Value int
		Tags  []string
	}
	testStruct := TestStruct{Name: "test", Value: 42, Tags: []string{"tag1", "tag2"}}
	err := l.SetData("scope1", "struct_key", testStruct)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	value, err := GetData[TestStruct](l, "scope1", "struct_key")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value.Name != "test" || value.Value != 42 || len(value.Tags) != 2 {
		t.Errorf("Expected struct {Name: 'test', Value: 42, Tags: ['tag1', 'tag2']}, got %+v", value)
	}

	// Test with slice
	slice := []int{1, 2, 3, 4, 5}
	err = l.SetData("scope1", "slice_key", slice)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	valueSlice, err := GetData[[]int](l, "scope1", "slice_key")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if len(valueSlice) != 5 {
		t.Errorf("Expected slice length 5, got %d", len(valueSlice))
	}

	// Test with map
	testMap := map[string]interface{}{"key1": "value1", "key2": 42, "key3": true}
	err = l.SetData("scope1", "map_key", testMap)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	valueMap, err := GetData[map[string]interface{}](l, "scope1", "map_key")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if valueMap["key1"] != "value1" {
		t.Errorf("Expected map value 'value1', got %v", valueMap["key1"])
	}
	if valueMap["key2"] != float64(42) { // JSON unmarshals numbers as float64
		t.Errorf("Expected map value 42, got %v", valueMap["key2"])
	}
}

func TestLedger_ConcurrentAccess(t *testing.T) {
	l := NewLedger()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			err := l.SetData("scope1", "concurrent_key", id)
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
			_, _ = GetData[int](l, "scope1", "concurrent_key")
			_, _ = l.GetDataHistory("scope1", "concurrent_key")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	history, err := l.GetDataHistory("scope1", "concurrent_key")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 10 {
		t.Errorf("Expected 10 entries in history, got %d", len(history))
	}
}

func TestLedger_EdgeCases(t *testing.T) {
	l := NewLedger()

	// Test empty scope
	err := l.SetData("", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData should accept empty scope: %v", err)
	}

	value, err := GetData[string](l, "", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}

	// Test empty key
	err = l.SetData("scope1", "", "value")
	if err != nil {
		t.Fatalf("SetData should accept empty key: %v", err)
	}

	value, err = GetData[string](l, "scope1", "")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got %q", value)
	}

	// Test empty value
	err = l.SetData("scope1", "empty_key", "")
	if err != nil {
		t.Fatalf("SetData should accept empty value: %v", err)
	}

	value, err = GetData[string](l, "scope1", "empty_key")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty value, got %q", value)
	}

	// Test nil value
	err = l.SetData("scope1", "nil_key", nil)
	if err != nil {
		t.Fatalf("SetData should accept nil value: %v", err)
	}

	// Test very long scope and key
	longScope := string(make([]byte, 1000))
	longKey := string(make([]byte, 1000))
	err = l.SetData(longScope, longKey, "value")
	if err != nil {
		t.Fatalf("SetData should accept long scope and key: %v", err)
	}

	// Test special characters in scope and key
	err = l.SetData("scope:with:colons", "key:with:colons", "value")
	if err != nil {
		t.Fatalf("SetData should accept scope and key with colons: %v", err)
	}

	value, err = GetData[string](l, "scope:with:colons", "key:with:colons")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != "value" {
		t.Errorf("Expected value 'value', got %q", value)
	}
}

func TestLedger_DeleteAndRecreate(t *testing.T) {
	l := NewLedger()

	// Set, delete, and recreate
	err := l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	err = l.DeleteData("scope1", "key1")
	if err != nil {
		t.Fatalf("DeleteData failed: %v", err)
	}

	err = l.SetData("scope1", "key1", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Verify we can get the new value
	value, err := GetData[string](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value != "value2" {
		t.Errorf("Expected value 'value2', got %q", value)
	}

	// Verify history contains all operations
	history, err := l.GetDataHistory("scope1", "key1")
	if err != nil {
		t.Fatalf("GetDataHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}
	if history[0].Operation != OperationSet {
		t.Errorf("Expected first operation 'set', got %q", history[0].Operation)
	}
	if history[1].Operation != OperationDelete {
		t.Errorf("Expected second operation 'delete', got %q", history[1].Operation)
	}
	if history[2].Operation != OperationSet {
		t.Errorf("Expected third operation 'set', got %q", history[2].Operation)
	}
}

func TestNewEntry(t *testing.T) {
	entry, err := NewEntry[string]("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("NewEntry failed: %v", err)
	}

	if entry.Scope != "scope1" {
		t.Errorf("Expected scope 'scope1', got %q", entry.Scope)
	}
	if entry.Key != "key1" {
		t.Errorf("Expected key 'key1', got %q", entry.Key)
	}
	if entry.Operation != OperationSet {
		t.Errorf("Expected operation 'set', got %q", entry.Operation)
	}
	if entry.Timestamp.IsZero() {
		t.Fatal("Timestamp should not be zero")
	}

	// Verify value can be extracted
	value, err := GetValue[string](&entry)
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got %q", value)
	}
}

func TestGetValue(t *testing.T) {
	entry, err := NewEntry[int]("scope1", "key1", 42)
	if err != nil {
		t.Fatalf("NewEntry failed: %v", err)
	}

	value, err := GetValue[int](&entry)
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}

	// Test wrong type
	_, err = GetValue[string](&entry)
	if err == nil {
		t.Fatal("GetValue should return error for wrong type")
	}
}

func TestLedger_MarshalJSON(t *testing.T) {
	l := NewLedger()

	// Test empty ledger
	data, err := l.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result map[string]map[string][]Entry
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}

	// Set data in multiple scopes
	err = l.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope1", "key2", "value2")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l.SetData("scope2", "key3", "value3")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Marshal
	data, err = l.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal and verify
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(result))
	}

	// Verify scope1
	scope1, ok := result["scope1"]
	if !ok {
		t.Fatal("Expected scope1 to be present")
	}
	if len(scope1) != 2 {
		t.Errorf("Expected 2 keys in scope1, got %d", len(scope1))
	}

	// Verify scope2
	scope2, ok := result["scope2"]
	if !ok {
		t.Fatal("Expected scope2 to be present")
	}
	if len(scope2) != 1 {
		t.Errorf("Expected 1 key in scope2, got %d", len(scope2))
	}
}

func TestLedger_UnmarshalJSON(t *testing.T) {
	l := NewLedger()

	// Create JSON data by marshaling actual entries
	entry1, _ := NewEntry("scope1", "key1", "value1")
	entry2, _ := NewEntry("scope1", "key2", "value2")
	entry3, _ := NewEntry("scope2", "key3", "value3")

	jsonData := map[string]map[string][]Entry{
		"scope1": {
			"key1": {entry1},
			"key2": {entry2},
		},
		"scope2": {
			"key3": {entry3},
		},
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	err = l.UnmarshalJSON(jsonBytes)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Verify data
	value1, err := GetData[string](l, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Expected value 'value1', got %q", value1)
	}

	value2, err := GetData[string](l, "scope1", "key2")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Expected value 'value2', got %q", value2)
	}

	value3, err := GetData[string](l, "scope2", "key3")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value3 != "value3" {
		t.Errorf("Expected value 'value3', got %q", value3)
	}
}

func TestLedger_MarshalUnmarshalRoundTrip(t *testing.T) {
	l1 := NewLedger()

	// Set data
	err := l1.SetData("scope1", "key1", "value1")
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l1.SetData("scope1", "key2", 42)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	err = l1.SetData("scope2", "key3", true)
	if err != nil {
		t.Fatalf("SetData failed: %v", err)
	}

	// Marshal
	data, err := l1.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal into new ledger
	l2 := NewLedger()
	err = l2.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Verify data matches
	value1, err := GetData[string](l2, "scope1", "key1")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Expected value 'value1', got %q", value1)
	}

	value2, err := GetData[int](l2, "scope1", "key2")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value2 != 42 {
		t.Errorf("Expected value 42, got %d", value2)
	}

	value3, err := GetData[bool](l2, "scope2", "key3")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if value3 != true {
		t.Errorf("Expected value true, got %v", value3)
	}
}
