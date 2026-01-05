package ledger

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Ledgers are marshalled to be only the data, stored
// via nested scopes. For nested scopes like "parent::child", the structure will be:
//
//	{
//	  "parent": {
//	    "child": {
//	      "key1": [
//	        {
//	          "scopes": ["parent", "child"],
//	          "key": "key1",
//	          "value": "value1",
//	          "timestamp": "2021-01-01T00:00:00Z",
//	          "operation": "set"
//	        }
//	      ],
//	      "subchild": {
//	        "key2": [
//	          {
//	            "scopes": ["parent", "child", "subchild"],
//	            "key": "key2",
//	            "value": "value2",
//	            "timestamp": "2021-01-01T00:00:00Z",
//	            "operation": "set"
//	          }
//	        ]
//	      }
//	    },
//	    ...
//	  }
//	}
func (ledger *Ledger) MarshalJSON() ([]byte, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	result := make(map[string]any)
	for fullKey, entries := range ledger.data {
		scopes, key, ok := splitScopeKey(fullKey)
		if !ok {
			// Skip malformed keys; ledger invariants should normally prevent this.
			continue
		}

		// Navigate/create nested structure
		current := result
		for i, scope := range scopes {
			if i == len(scopes)-1 {
				// Last scope level - this is where keys go
				if current[scope] == nil {
					current[scope] = make(map[string]any)
				}
				// Check if it's already a map[string]any (might have nested scopes)
				scopeMap, ok := current[scope].(map[string]any)
				if !ok {
					// It's map[string][]Entry, convert to map[string]any
					if keysMap, ok := current[scope].(map[string][]Entry); ok {
						scopeMap = make(map[string]any)
						for k, v := range keysMap {
							scopeMap[k] = v
						}
						current[scope] = scopeMap
					} else {
						// Unknown type, create new
						scopeMap = make(map[string]any)
						current[scope] = scopeMap
					}
				}
				scopeMap[key] = entries
			} else {
				// Intermediate scope level
				if current[scope] == nil {
					current[scope] = make(map[string]any)
				}
				next, ok := current[scope].(map[string]any)
				if !ok {
					// Type mismatch, shouldn't happen but handle gracefully
					current[scope] = make(map[string]any)
					next = current[scope].(map[string]any)
				}
				current = next
			}
		}
	}
	return json.Marshal(result)
}

func (ledger *Ledger) UnmarshalJSON(data []byte) error {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	// Unmarshal into a generic structure to handle nested scopes
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	// Transform from nested structure back to map["scope::key"][]Entry
	ledger.data = make(map[string][]Entry)
	if err := unmarshalNestedScopes(result, []string{}, ledger.data); err != nil {
		return err
	}
	return nil
}

// keysToSlices recursively converts map[string]bool to []string
func keysToSlices(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		switch val := v.(type) {
		case map[string]bool:
			// Convert keys map to string slice
			keys := make([]string, 0, len(val))
			for key := range val {
				keys = append(keys, key)
			}
			result[k] = keys
		case map[string]any:
			// Check if this has a "_keys" entry
			if keysMap, ok := val["_keys"].(map[string]bool); ok {
				// Has both keys and nested scopes
				keys := make([]string, 0, len(keysMap))
				for key := range keysMap {
					keys = append(keys, key)
				}
				// Process nested scopes (excluding "_keys")
				nested := make(map[string]any)
				for nk, nv := range val {
					if nk != "_keys" {
						nested[nk] = keysToSlices(map[string]any{nk: nv})[nk]
					}
				}
				// Combine keys and nested scopes
				combined := make(map[string]any)
				if len(keys) > 0 {
					combined["_keys"] = keys
				}
				for nk, nv := range nested {
					combined[nk] = nv
				}
				result[k] = combined
			} else {
				// Recursively process nested scopes only
				result[k] = keysToSlices(val)
			}
		default:
			result[k] = val
		}
	}
	return result
}

// unmarshalNestedScopes recursively processes nested scope structure
func unmarshalNestedScopes(m map[string]any, currentScopes []string, data map[string][]Entry) error {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			// Check if this map has both keys ([]Entry) and nested scopes (map[string]any)
			// or just one type
			hasKeys := false
			hasNestedScopes := false
			keysToProcess := make(map[string]any)
			nestedScopesToProcess := make(map[string]any)

			for subK, subVal := range val {
				if subK == "_keys" {
					// Skip _keys metadata
					continue
				}
				switch subVal.(type) {
				case []any:
					// This is a key with []Entry
					hasKeys = true
					keysToProcess[subK] = subVal
				case map[string]any:
					// This is a nested scope
					hasNestedScopes = true
					nestedScopesToProcess[subK] = subVal
				default:
					// Unknown type, skip
				}
			}

			// Process keys at this level
			if hasKeys {
				for key, entriesRaw := range keysToProcess {
					entriesSlice, ok := entriesRaw.([]any)
					if !ok {
						return fmt.Errorf("expected []Entry for key %s, got %T", key, entriesRaw)
					}
					entries := make([]Entry, 0, len(entriesSlice))
					for _, entryRaw := range entriesSlice {
						entryBytes, err := json.Marshal(entryRaw)
						if err != nil {
							return err
						}
						var entry Entry
						if err := json.Unmarshal(entryBytes, &entry); err != nil {
							return err
						}
						// Ensure Scopes is set
						if len(entry.Scopes) == 0 && entry.Scope != "" {
							entry.Scopes = parseScopeString(entry.Scope)
						} else if len(entry.Scopes) == 0 {
							// Use currentScopes which includes k
							finalScopes := append(currentScopes, k)
							entry.Scopes = finalScopes
							entry.Scope = strings.Join(finalScopes, internalScopeSeparator)
						}
						entries = append(entries, entry)
					}
					// Reconstruct fullKey - k is the scope, key is the key name
					finalScopes := append(currentScopes, k)
					scopeStr := strings.Join(finalScopes, internalScopeSeparator)
					fullKey := fmt.Sprintf("%s::%s", scopeStr, key)
					data[fullKey] = entries
				}
			}

			// Process nested scopes - recurse
			if hasNestedScopes {
				newScopes := append(currentScopes, k)
				if err := unmarshalNestedScopes(nestedScopesToProcess, newScopes, data); err != nil {
					return err
				}
			}
		case []any:
			// Direct array - shouldn't happen at top level, but handle it
			return fmt.Errorf("unexpected array at scope level %v", currentScopes)
		default:
			return fmt.Errorf("unexpected type %T in nested structure at scope %v", val, currentScopes)
		}
	}
	return nil
}
