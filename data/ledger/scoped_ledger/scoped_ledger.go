package scoped_ledger

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hlfshell/gotonomy/data/ledger"
)

type ScopedLedger struct {
	ledger *ledger.Ledger
	scope  string
}

func NewScopedLedger(ledger *ledger.Ledger, scope string) *ScopedLedger {
	return &ScopedLedger{
		ledger: ledger,
		scope:  scope,
	}
}

func (sl *ScopedLedger) SetData(key string, value any) error {
	return sl.ledger.SetData(sl.scope, key, value)
}

func (sl *ScopedLedger) DeleteData(key string) error {
	return sl.ledger.DeleteData(sl.scope, key)
}

func SetDataFunc[T any](
	sl *ScopedLedger,
	key string,
	fn func(T) (T, error),
) error {
	return ledger.SetDataFunc[T](sl.ledger, sl.scope, key, fn)
}

func (sl *ScopedLedger) GetData(key string) (ledger.Entry, error) {
	return sl.ledger.GetData(sl.scope, key)
}

func GetDataScoped[T any](sl *ScopedLedger, key string) (T, error) {
	return ledger.GetData[T](sl.ledger, sl.scope, key)
}

func (sl *ScopedLedger) GetDataHistory(key string) ([]ledger.Entry, error) {
	return sl.ledger.GetDataHistory(sl.scope, key)
}

func GetDataHistoryScoped[T any](sl *ScopedLedger, key string) ([]T, error) {
	return ledger.GetDataHistory[T](sl.ledger, sl.scope, key)
}

func (sl *ScopedLedger) GetKeys() []string {
	allKeys := sl.ledger.GetKeys()
	keys, ok := allKeys[sl.scope]
	if !ok {
		return []string{}
	}
	return keys
}

// Subscoped creates a new ScopedLedger that is nested under the current scope.
// For example, if the current scope is "foo" and subScope is "bar",
// the resulting scope will be "foo:bar".
func (sl *ScopedLedger) Subscoped(subScope string) *ScopedLedger {
	// Avoid double separators if caller passed a value that already
	// contains the current scope prefix, ie was passed foo:bar instead
	// of just bar for our foo scope.
	if strings.HasPrefix(subScope, sl.scope+":") {
		return NewScopedLedger(sl.ledger, subScope)
	}

	scope := sl.scope + ":" + subScope
	return NewScopedLedger(sl.ledger, scope)
}

func (sl *ScopedLedger) MarshalJSON() ([]byte, error) {
	result := make(map[string][]ledger.Entry)
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
