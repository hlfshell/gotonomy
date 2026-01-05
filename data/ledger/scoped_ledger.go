package ledger

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validateScopeKey checks that scope and key don't contain "::"
// which is reserved for internal use.
func validateScopeKey(scope, key string) error {
	if strings.Contains(scope, internalScopeSeparator) {
		return fmt.Errorf("scope cannot contain %q (reserved for internal use)", internalScopeSeparator)
	}
	if strings.Contains(key, internalScopeSeparator) {
		return fmt.Errorf("key cannot contain %q (reserved for internal use)", internalScopeSeparator)
	}
	return nil
}

type ScopedLedger struct {
	ledger *Ledger
	scope  string
}

// NewScoped creates a new ScopedLedger with the given scope.
// The scope cannot contain "::" which is reserved for internal use.
func NewScoped(ledger *Ledger, scope string) (*ScopedLedger, error) {
	if err := validateScopeKey(scope, ""); err != nil {
		return nil, err
	}
	return &ScopedLedger{
		ledger: ledger,
		scope:  scope,
	}, nil
}

// newScopedInternal creates a new ScopedLedger without validation.
// This is used internally by the ledger package to create scoped ledgers
// that may use "::" as a separator.
func newScopedInternal(ledger *Ledger, scope string) *ScopedLedger {
	return &ScopedLedger{
		ledger: ledger,
		scope:  scope,
	}
}

func (sl *ScopedLedger) SetData(key string, value any) error {
	if err := validateScopeKey("", key); err != nil {
		return err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	return sl.ledger.setDataInternal(sl.scope, key, value)
}

func (sl *ScopedLedger) DeleteData(key string) error {
	if err := validateScopeKey("", key); err != nil {
		return err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	return sl.ledger.deleteDataInternal(sl.scope, key)
}

func SetDataFuncScoped[T any](
	sl *ScopedLedger,
	key string,
	fn func(T) (T, error),
) error {
	if err := validateScopeKey("", key); err != nil {
		return err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	return setDataFuncInternal[T](sl.ledger, sl.scope, key, fn)
}

func (sl *ScopedLedger) GetData(key string) (Entry, error) {
	if err := validateScopeKey("", key); err != nil {
		return Entry{}, err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	return sl.ledger.getDataInternal(sl.scope, key)
}

func GetDataScoped[T any](sl *ScopedLedger, key string) (T, error) {
	if err := validateScopeKey("", key); err != nil {
		var zero T
		return zero, err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	entry, err := sl.ledger.getDataInternal(sl.scope, key)
	if err != nil {
		var zero T
		return zero, err
	}
	return GetValue[T](&entry)
}

func (sl *ScopedLedger) GetDataHistory(key string) ([]Entry, error) {
	if err := validateScopeKey("", key); err != nil {
		return nil, err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	return sl.ledger.getDataHistoryInternal(sl.scope, key)
}

func GetDataHistoryScoped[T any](sl *ScopedLedger, key string) ([]T, error) {
	if err := validateScopeKey("", key); err != nil {
		return nil, err
	}
	// Use internal method since scope is already validated when ScopedLedger was created
	entries, err := sl.ledger.getDataHistoryInternal(sl.scope, key)
	if err != nil {
		return nil, err
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

func (sl *ScopedLedger) GetKeys() []string {
	allKeys := sl.ledger.GetKeys()
	scopes := parseScopeString(sl.scope)

	if len(scopes) == 0 {
		return []string{}
	}

	// Navigate through nested structure
	current := allKeys
	for i, scope := range scopes {
		next, ok := current[scope]
		if !ok {
			return []string{}
		}

		if i == len(scopes)-1 {
			// Last scope - check what we have
			if keys, ok := next.([]string); ok {
				// Direct keys at this scope level
				return keys
			}
			// If it's a map, check if it has "_keys" entry
			if nextMap, ok := next.(map[string]interface{}); ok {
				if keysRaw, ok := nextMap["_keys"]; ok {
					if keys, ok := keysRaw.([]string); ok {
						return keys
					}
				}
			}
			// No keys at this level
			return []string{}
		}

		// Intermediate scope - should be map[string]interface{}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return []string{}
		}
		current = nextMap
	}

	return []string{}
}

// Scoped creates a new ScopedLedger that is nested under the current scope.
// For example, if the current scope is "foo" and subScope is "bar",
// the resulting scope will be "foo::bar" (using internal separator).
// The subScope parameter cannot contain "::" which is reserved for internal use,
// except when it already starts with the current scope followed by "::" (in which
// case it is used as-is to avoid double prefixing).
func (sl *ScopedLedger) Scoped(subScope string) (*ScopedLedger, error) {
	// Avoid double separators if caller passed a value that already
	// contains the current scope prefix with internal separator
	prefix := sl.scope + internalScopeSeparator
	if strings.HasPrefix(subScope, prefix) {
		// Already has the prefix, use as-is (no validation needed since it's internal)
		return newScopedInternal(sl.ledger, subScope), nil
	}

	// Validate that subScope doesn't contain "::" (reserved for internal use)
	if err := validateScopeKey(subScope, ""); err != nil {
		return nil, err
	}

	// Use internal separator for nested scopes
	scope := sl.scope + internalScopeSeparator + subScope
	return newScopedInternal(sl.ledger, scope), nil
}

func (sl *ScopedLedger) MarshalJSON() ([]byte, error) {
	result := make(map[string][]Entry)
	keys := sl.GetKeys()
	for _, key := range keys {
		entries, err := sl.GetDataHistory(key)
		if err != nil {
			return nil, err
		}
		result[key] = entries
	}
	return json.Marshal(result)
}

func (sl *ScopedLedger) UnmarshalJSON(data []byte) error {
	return fmt.Errorf("unmarshalling scoped ledgers is not supported")
}
