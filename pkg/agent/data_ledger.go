// Package agent provides interfaces and implementations for building AI agents
// that can use language models to accomplish tasks.
package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

// DataLedgerEntry represents a single entry in the data ledger tracking state changes over time
type DataLedgerEntry struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Timestamp time.Time       `json:"timestamp"`
	Operation string          `json:"operation"` // "set", "update", "delete"
}

// SetData stores a value in the current node's data ledger
// T must be JSON-serializable
func SetData[T any](ec *ExecutionContext, key string, value T) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.current == nil {
		return fmt.Errorf("cannot set data: no current node")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	entry := DataLedgerEntry{
		Key:       key,
		Value:     data,
		Timestamp: time.Now(),
		Operation: "set",
	}

	if ec.current.Data == nil {
		ec.current.Data = []DataLedgerEntry{}
	}
	ec.current.Data = append(ec.current.Data, entry)
	return nil
}

// GetData retrieves the most recent value for a key from the current node's data ledger
func GetData[T any](ec *ExecutionContext, key string) (T, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var zero T
	if ec.current == nil {
		return zero, false
	}

	if ec.current.Data == nil {
		return zero, false
	}

	// Find the most recent entry for this key
	for i := len(ec.current.Data) - 1; i >= 0; i-- {
		entry := ec.current.Data[i]
		if entry.Key == key {
			// If the most recent entry is a delete, the key doesn't exist
			if entry.Operation == "delete" {
				return zero, false
			}
			// Found a set/update entry, return its value
			var value T
			if err := json.Unmarshal(entry.Value, &value); err != nil {
				return zero, false
			}
			return value, true
		}
	}

	return zero, false
}

// GetDataHistory returns all historical entries for a key in the current node
// in first to last order
func (ec *ExecutionContext) GetDataHistory(key string) []DataLedgerEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if ec.current == nil || ec.current.Data == nil {
		return []DataLedgerEntry{}
	}

	return ec.current.Data
}

// SetExecutionData stores a value in execution-level data ledger (shared across all children)
func SetExecutionData[T any](ec *ExecutionContext, key string, value T) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal execution data for key %s: %w", key, err)
	}

	entry := DataLedgerEntry{
		Key:       key,
		Value:     data,
		Timestamp: time.Now(),
		Operation: "set",
	}

	ec.executionDataLedger = append(ec.executionDataLedger, entry)
	return nil
}

// GetExecutionData retrieves the most recent value for a key from execution-level data ledger
func GetExecutionData[T any](ec *ExecutionContext, key string) (T, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var zero T
	if len(ec.executionDataLedger) == 0 {
		return zero, false
	}

	// Find the most recent entry for this key
	for i := len(ec.executionDataLedger) - 1; i >= 0; i-- {
		entry := ec.executionDataLedger[i]
		if entry.Key == key {
			// If the most recent entry is a delete, the key doesn't exist
			if entry.Operation == "delete" {
				return zero, false
			}
			// Found a set/update entry, return its value
			var value T
			if err := json.Unmarshal(entry.Value, &value); err != nil {
				return zero, false
			}
			return value, true
		}
	}

	return zero, false
}

// GetExecutionDataHistory returns all historical entries for a key in execution-level data
func (ec *ExecutionContext) GetExecutionDataHistory(key string) []DataLedgerEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var history []DataLedgerEntry
	for _, entry := range ec.executionDataLedger {
		if entry.Key == key {
			history = append(history, entry)
		}
	}
	return history
}

// DeleteData marks a key as deleted in the current node's data ledger
func (ec *ExecutionContext) DeleteData(key string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.current == nil {
		return // Silently ignore if no current node
	}

	if ec.current.Data == nil {
		ec.current.Data = []DataLedgerEntry{}
	}

	entry := DataLedgerEntry{
		Key:       key,
		Value:     nil,
		Timestamp: time.Now(),
		Operation: "delete",
	}
	ec.current.Data = append(ec.current.Data, entry)
}

// DeleteExecutionData marks a key as deleted in execution-level data ledger
func (ec *ExecutionContext) DeleteExecutionData(key string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	entry := DataLedgerEntry{
		Key:       key,
		Value:     nil,
		Timestamp: time.Now(),
		Operation: "delete",
	}
	ec.executionDataLedger = append(ec.executionDataLedger, entry)
}
