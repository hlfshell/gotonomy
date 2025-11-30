package ledger

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Operation string

const (
	OperationSet    = "set"
	OperationDelete = "delete"
)

// Entry represents a single entry in the data ledger tracking state changes over time
type Entry struct {
	Scope     string          `json:"scope"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Timestamp time.Time       `json:"timestamp"`
	Operation Operation       `json:"operation"`
}

func NewEntry[T any](scope, key string, value T) (Entry, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}
	return Entry{
		Scope:     scope,
		Key:       key,
		Value:     data,
		Timestamp: time.Now(),
		Operation: OperationSet,
	}, nil
}

func GetValue[T any](entry *Entry) (T, error) {
	var zero T
	if err := json.Unmarshal(entry.Value, &zero); err != nil {
		return zero, fmt.Errorf("failed to unmarshal value for key %s: %w", entry.Key, err)
	}
	return zero, nil
}

type Ledger struct {
	data map[string][]Entry
	mu   sync.RWMutex
}

// splitScopeKey splits a fullKey of the form "scope:key" into scope and key,
// where scopes themselves may contain ":" characters. It always uses the LAST
// ":" as the separator. It returns ok=false if the key is malformed.
func splitScopeKey(fullKey string) (scope string, key string, ok bool) {
	idx := strings.LastIndex(fullKey, ":")
	if idx <= 0 || idx == len(fullKey)-1 {
		// Either no ":", starts with ":", or ends with ":" – treat as malformed.
		return "", "", false
	}
	return fullKey[:idx], fullKey[idx+1:], true
}

func NewLedger() *Ledger {
	return &Ledger{
		data: make(map[string][]Entry),
	}
}

func (ledger *Ledger) append(fullKey string, entry Entry) {
	if ledger.data == nil {
		ledger.data = make(map[string][]Entry)
	}
	if _, ok := ledger.data[fullKey]; !ok {
		ledger.data[fullKey] = []Entry{}
	}
	ledger.data[fullKey] = append(ledger.data[fullKey], entry)
}

func (ledger *Ledger) SetData(scope, key string, value any) error {
	fullKey := fmt.Sprintf("%s:%s", scope, key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	entry := Entry{
		Scope:     scope,
		Key:       key,
		Value:     data,
		Timestamp: time.Now(),
		Operation: OperationSet,
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	ledger.append(fullKey, entry)

	return nil
}

func (ledger *Ledger) SetDataFunc(
	scope, key string,
	fn func(Entry) (Entry, error),
) error {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	fullKey := fmt.Sprintf("%s:%s", scope, key)
	if _, ok := ledger.data[fullKey]; !ok {
		return fmt.Errorf("key %s does not exist", key)
	}

	entry := ledger.data[fullKey][len(ledger.data[fullKey])-1]
	newEntry, err := fn(entry)
	if err != nil {
		return fmt.Errorf("failed to set data for key %s: %w", key, err)
	}

	ledger.append(fullKey, newEntry)
	return nil
}

func SetDataFunc[T any](
	ledger *Ledger,
	scope, key string,
	fn func(T) (T, error),
) error {
	var zero T
	fullKey := fmt.Sprintf("%s:%s", scope, key)
	if _, ok := ledger.data[fullKey]; !ok {
		return fmt.Errorf("key %s does not exist", key)
	}

	entry := ledger.data[fullKey][len(ledger.data[fullKey])-1]
	err := json.Unmarshal(entry.Value, &zero)
	if err != nil {
		return fmt.Errorf("failed to unmarshal value for key %s: %w", key, err)
	}

	data, err := fn(zero)
	if err != nil {
		return fmt.Errorf("failed to set data for key %s: %w", key, err)
	}

	newEntry, err := NewEntry[T](scope, key, data)
	if err != nil {
		return fmt.Errorf("failed to create new entry for key %s: %w", key, err)
	}
	ledger.append(fullKey, newEntry)

	return nil
}

func (ledger *Ledger) DeleteData(scope, key string) error {
	fullKey := fmt.Sprintf("%s:%s", scope, key)

	entry := Entry{
		Scope:     scope,
		Key:       key,
		Value:     nil,
		Timestamp: time.Now(),
		Operation: OperationDelete,
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	ledger.append(fullKey, entry)

	return nil
}

func (ledger *Ledger) GetData(scope, key string) (Entry, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	fullKey := fmt.Sprintf("%s:%s", scope, key)
	entries, ok := ledger.data[fullKey]
	if !ok || len(entries) == 0 {
		return Entry{}, fmt.Errorf("key %s does not exist", key)
	}

	latestEntry := entries[len(entries)-1]
	if latestEntry.Operation == OperationDelete {
		return Entry{}, fmt.Errorf("key %s has been deleted", key)
	}

	return latestEntry, nil
}

func GetData[T any](ledger *Ledger, scope string, key string) (T, error) {
	var zero T

	data, err := ledger.GetData(scope, key)
	if err != nil {
		return zero, fmt.Errorf("failed to get data for key %s: %w", key, err)
	}

	if err := json.Unmarshal(data.Value, &zero); err != nil {
		return zero, fmt.Errorf("failed to unmarshal value for key %s: %w", key, err)
	}
	return zero, nil
}

func (ledger *Ledger) GetDataHistory(scope, key string) ([]Entry, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	fullKey := fmt.Sprintf("%s:%s", scope, key)
	entries, ok := ledger.data[fullKey]
	if !ok {
		return nil, fmt.Errorf("key %s does not exist", key)
	}

	return entries, nil
}

func GetDataHistory[T any](ledger *Ledger, scope string, key string) ([]T, error) {
	entries, err := ledger.GetDataHistory(scope, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get data history for key %s: %w", key, err)
	}

	var history []T
	for _, entry := range entries {
		// Skip deleted entries in typed history
		if entry.Operation == OperationDelete {
			continue
		}
		var value T
		if err := json.Unmarshal(entry.Value, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value for key %s: %w", key, err)
		}
		history = append(history, value)
	}
	return history, nil
}

func (ledger *Ledger) GetScopes() []string {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	scopeMap := make(map[string]bool)
	for fullKey := range ledger.data {
		// Extract scope from fullKey (format: "scope:key") using the helper.
		scope, _, ok := splitScopeKey(fullKey)
		if !ok {
			continue
		}
		scopeMap[scope] = true
	}

	scopes := make([]string, 0, len(scopeMap))
	for scope := range scopeMap {
		scopes = append(scopes, scope)
	}

	return scopes
}

// GetKeys grabs all keys organized by scope; for example:
//
//	{
//	  "scope1": ["key1", "key2"],
//	  "scope2": ["key3", "key4"]
//	}
func (ledger *Ledger) GetKeys() map[string][]string {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	result := make(map[string][]string)
	keyMap := make(map[string]map[string]bool) // scope -> key -> bool

	for fullKey := range ledger.data {
		scope, key, ok := splitScopeKey(fullKey)
		if !ok {
			continue
		}

		if keyMap[scope] == nil {
			keyMap[scope] = make(map[string]bool)
		}
		keyMap[scope][key] = true
	}

	// Convert to map[string][]string
	for scope, keys := range keyMap {
		keyList := make([]string, 0, len(keys))
		for key := range keys {
			keyList = append(keyList, key)
		}
		result[scope] = keyList
	}

	return result
}

// Ledgers are marshalled to be only the data, stored
// via scope. So:
//
//	{
//	  "scope": [
//	    {
//	      "key": "key1",
//	      "history": [
//	        {
//	          "key": "key1",
//	          "value": "value1",
//	          "timestamp": "2021-01-01T00:00:00Z",
//	          "operation": "set"
//	        }
//	      ]
//	    },
//	    ...
//	  ]
//	}
func (ledger *Ledger) MarshalJSON() ([]byte, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	result := make(map[string]map[string][]Entry)
	for fullKey, entries := range ledger.data {
		scope, key, ok := splitScopeKey(fullKey)
		if !ok {
			// Skip malformed keys; ledger invariants should normally prevent this.
			continue
		}

		if _, ok := result[scope]; !ok {
			result[scope] = map[string][]Entry{}
		}
		result[scope][key] = entries
	}
	return json.Marshal(result)
}

func (ledger *Ledger) UnmarshalJSON(data []byte) error {
	// We convert to a map of maps first due to the
	// style that we marshal ledgers to
	var result map[string]map[string][]Entry
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	// Transform from map[scope]map[key][]Entry back to
	// map["scope:key"][]Entry
	ledger.data = make(map[string][]Entry)
	for scope, keys := range result {
		for key, entries := range keys {
			fullKey := fmt.Sprintf("%s:%s", scope, key)
			ledger.data[fullKey] = entries
		}
	}
	return nil
}
