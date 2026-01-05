package agent

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

func TestDefaultArgumentsToPrompt_JSONMarshalling(t *testing.T) {
	args := tool.Arguments{
		"foo": "bar",
		"n":   42,
	}

	m, err := DefaultArgumentsToPrompt(args)
	if err != nil {
		t.Fatalf("DefaultArgumentsToPrompt returned error: %v", err)
	}

	input, ok := m["input"]
	if !ok {
		t.Fatalf("expected 'input' key in prompt map")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(input), &decoded); err != nil {
		t.Fatalf("failed to unmarshal input JSON: %v", err)
	}

	// We expect foo and n to be present after round-trip.
	if decoded["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", decoded["foo"])
	}
	// JSON numbers come back as float64
	if decoded["n"] != float64(42) {
		t.Errorf("expected n=42, got %v", decoded["n"])
	}
}

func TestDefaultArgumentsToMessages_FirstIteration(t *testing.T) {
	args := tool.Arguments{
		"foo": "bar",
	}

	msgs, err := DefaultArgumentsToMessages(args, nil)
	if err != nil {
		t.Fatalf("DefaultArgumentsToMessages returned error: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != model.RoleUser {
		t.Errorf("expected RoleUser, got %v", msgs[0].Role)
	}
	if msgs[0].Content == "" {
		t.Errorf("expected non-empty content")
	}
}

func TestDefaultArgumentsToMessages_ReusesConversation(t *testing.T) {
	sess := NewSession()

	step1 := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "u1"},
	})
	step1.SetResponse(Response{
		Output: model.Message{Role: model.RoleAssistant, Content: "a1"},
	})
	sess.AddStep(step1)

	step2 := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "u2"},
	})
	step2.SetResponse(Response{
		Output: model.Message{Role: model.RoleAssistant, Content: "a2"},
	})
	sess.AddStep(step2)

	msgs, err := DefaultArgumentsToMessages(tool.Arguments{}, sess)
	if err != nil {
		t.Fatalf("DefaultArgumentsToMessages returned error: %v", err)
	}

	conv := sess.Conversation()
	if !reflect.DeepEqual(msgs, conv) {
		t.Errorf("expected messages to equal session conversation.\nmsgs=%v\nconv=%v", msgs, conv)
	}
}


