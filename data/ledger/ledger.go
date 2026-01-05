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
	Scopes    []string        `json:"scopes"` // Nested scopes as a slice
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Timestamp time.Time       `json:"timestamp"`
	Operation Operation       `json:"operation"`
	// Scope is kept for backward compatibility
	// It is automatically populated from Scopes when marshaling
	Scope string `json:"scope,omitempty"`
}

// MarshalJSON ensures Scope is populated from Scopes for backward compatibility
func (e Entry) MarshalJSON() ([]byte, error) {
	// Populate Scope from Scopes if not set
	if e.Scope == "" && len(e.Scopes) > 0 {
		e.Scope = strings.Join(e.Scopes, internalScopeSeparator)
	}
	type Alias Entry
	return json.Marshal((*Alias)(&e))
}

// UnmarshalJSON ensures Scopes is populated from Scope for backward compatibility
func (e *Entry) UnmarshalJSON(data []byte) error {
	type Alias Entry
	aux := (*Alias)(e)
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	// Populate Scopes from Scope if Scopes is empty
	if len(e.Scopes) == 0 && e.Scope != "" {
		e.Scopes = parseScopeString(e.Scope)
	} else if len(e.Scopes) > 0 && e.Scope == "" {
		// Populate Scope from Scopes
		e.Scope = strings.Join(e.Scopes, internalScopeSeparator)
	}
	return nil
}

// parseScopeString parses a scope string (which may contain "::") into a slice of scopes
func parseScopeString(scope string) []string {
	if scope == "" {
		return []string{}
	}
	return strings.Split(scope, internalScopeSeparator)
}

func NewEntry[T any](scope, key string, value T) (Entry, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}
	scopes := parseScopeString(scope)
	return Entry{
		Scopes:    scopes,
		Scope:     scope, // Keep for backward compatibility
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

func NewLedger() *Ledger {
	return &Ledger{
		data: make(map[string][]Entry),
		mu:   sync.RWMutex{},
	}
}

const (
	// internalScopeSeparator is used internally by the ledger package
	// to create nested scopes. External callers cannot use "::" in
	// scope or key parameters.
	internalScopeSeparator = "::"
)

// splitScopeKey splits a fullKey of the form "scope::key" into scopes (as a slice) and key,
// where scopes themselves may contain "::" characters (and are thus split further). It always uses the LAST
// "::" as the separator for the key, but returns all individual scopes in the slice.
// It returns ok=false if the key is malformed.
func splitScopeKey(fullKey string) (scopes []string, key string, ok bool) {
	idx := strings.LastIndex(fullKey, internalScopeSeparator)
	if idx <= 0 || idx == len(fullKey)-1 {
		// Either no ":", starts with ":", or ends with ":" – treat as malformed.
		return nil, "", false
	}
	scopesPart := fullKey[:idx]
	key = fullKey[idx+len(internalScopeSeparator):]
	// Split all parent scopes by "::"
	scopes = strings.Split(scopesPart, internalScopeSeparator)
	return scopes, key, true
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

// setDataInternal sets data without validating scope/key (for internal use)
func (ledger *Ledger) setDataInternal(scope, key string, value any) error {
	fullKey := fmt.Sprintf("%s::%s", scope, key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	scopes := parseScopeString(scope)
	entry := Entry{
		Scopes:    scopes,
		Scope:     scope, // Keep for backward compatibility
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

func (ledger *Ledger) SetData(scope, key string, value any) error {
	if err := validateScopeKey(scope, key); err != nil {
		return err
	}
	return ledger.setDataInternal(scope, key, value)
}

func (ledger *Ledger) SetDataFunc(
	scope, key string,
	fn func(Entry) (Entry, error),
) error {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	fullKey := fmt.Sprintf("%s::%s", scope, key)
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

// setDataFuncInternal sets data using a function without validating scope/key (for internal use)
func setDataFuncInternal[T any](
	ledger *Ledger,
	scope, key string,
	fn func(T) (T, error),
) error {
	var zero T
	fullKey := fmt.Sprintf("%s::%s", scope, key)
	ledger.mu.RLock()
	entries, ok := ledger.data[fullKey]
	if !ok || len(entries) == 0 {
		ledger.mu.RUnlock()
		return fmt.Errorf("key %s does not exist", key)
	}
	entry := entries[len(entries)-1]
	ledger.mu.RUnlock()

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

	ledger.mu.Lock()
	ledger.append(fullKey, newEntry)
	ledger.mu.Unlock()

	return nil
}

func SetDataFunc[T any](
	ledger *Ledger,
	scope, key string,
	fn func(T) (T, error),
) error {
	if err := validateScopeKey(scope, key); err != nil {
		return err
	}
	return setDataFuncInternal[T](ledger, scope, key, fn)
}

// deleteDataInternal deletes data without validating scope/key (for internal use)
func (ledger *Ledger) deleteDataInternal(scope, key string) error {
	fullKey := fmt.Sprintf("%s::%s", scope, key)

	scopes := parseScopeString(scope)
	entry := Entry{
		Scopes:    scopes,
		Scope:     scope, // Keep for backward compatibility
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

func (ledger *Ledger) DeleteData(scope, key string) error {
	if err := validateScopeKey(scope, key); err != nil {
		return err
	}
	return ledger.deleteDataInternal(scope, key)
}

// getDataInternal gets data without validating scope/key (for internal use)
func (ledger *Ledger) getDataInternal(scope, key string) (Entry, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	fullKey := fmt.Sprintf("%s::%s", scope, key)
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

func (ledger *Ledger) GetData(scope, key string) (Entry, error) {
	if err := validateScopeKey(scope, key); err != nil {
		return Entry{}, err
	}
	return ledger.getDataInternal(scope, key)
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

// getDataHistoryInternal gets data history without validating scope/key (for internal use)
func (ledger *Ledger) getDataHistoryInternal(scope, key string) ([]Entry, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	fullKey := fmt.Sprintf("%s::%s", scope, key)
	entries, ok := ledger.data[fullKey]
	if !ok {
		return nil, fmt.Errorf("key %s does not exist", key)
	}

	return entries, nil
}

func (ledger *Ledger) GetDataHistory(scope, key string) ([]Entry, error) {
	if err := validateScopeKey(scope, key); err != nil {
		return nil, err
	}
	return ledger.getDataHistoryInternal(scope, key)
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
		scopes, _, ok := splitScopeKey(fullKey)
		if !ok {
			continue
		}
		for _, scope := range scopes {
			scopeMap[scope] = true
		}
	}

	scopes := make([]string, 0, len(scopeMap))
	for scope := range scopeMap {
		scopes = append(scopes, scope)
	}

	return scopes
}

// GetKeys grabs all keys organized by scope hierarchy as nested maps.
// For nested scopes like "parent::child", the structure will be:
//
//	{
//	  "parent": {
//	    "child": ["key1", "key2"]
//	  },
//	  "scope2": ["key3", "key4"]
//	}
func (ledger *Ledger) GetKeys() map[string]any {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	// Build nested structure
	result := make(map[string]any)

	for fullKey := range ledger.data {
		scopes, key, ok := splitScopeKey(fullKey)
		if !ok {
			continue
		}

		// Navigate/create nested structure
		current := result
		for i, scope := range scopes {
			if i == len(scopes)-1 {
				// Last scope level - this is where keys go
				if current[scope] == nil {
					current[scope] = make(map[string]bool)
				}
				// Check if it's already a map[string]any (nested scopes exist)
				if nextMap, ok := current[scope].(map[string]any); ok {
					// Nested scopes already exist, we need to add keys to a special "_keys" entry
					if nextMap["_keys"] == nil {
						nextMap["_keys"] = make(map[string]bool)
					}
					keysMap := nextMap["_keys"].(map[string]bool)
					keysMap[key] = true
				} else {
					// No nested scopes yet, use direct keys map
					keysMap, ok := current[scope].(map[string]bool)
					if !ok {
						// Type mismatch, skip
						continue
					}
					keysMap[key] = true
				}
			} else {
				// Intermediate scope level
				if current[scope] == nil {
					current[scope] = make(map[string]any)
				}
				next, ok := current[scope].(map[string]any)
				if !ok {
					// Type mismatch - might be map[string]bool (keys), convert it
					if keysMap, ok := current[scope].(map[string]bool); ok {
						// Convert to nested structure with keys
						newMap := make(map[string]any)
						newMap["_keys"] = keysMap
						current[scope] = newMap
						next = newMap
					} else {
						// Unknown type, create new
						current[scope] = make(map[string]any)
						next = current[scope].(map[string]any)
					}
				}
				current = next
			}
		}
	}

	// Convert boolean maps to string slices
	return keysToSlices(result)
}
