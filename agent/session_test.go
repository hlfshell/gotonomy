package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hlfshell/gotonomy/model"
)

func TestSessionBasics(t *testing.T) {
	sess := NewSession()
	if sess.Iterations() != 0 {
		t.Fatalf("expected 0 iterations, got %d", sess.Iterations())
	}
	if sess.Duration() != 0 {
		t.Fatalf("expected zero duration for empty session, got %v", sess.Duration())
	}
	if sess.Finished() {
		t.Fatalf("expected Finished=false for empty session")
	}
}

func TestSessionFinishedAndDuration(t *testing.T) {
	sess := NewSession()

	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "hi"},
	})
	// Simulate a small delay
	time.Sleep(1 * time.Millisecond)
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "ok",
		},
		ToolCalls: nil,
	})
	sess.AddStep(step)

	if !sess.Finished() {
		t.Fatalf("expected Finished=true when last step has no tool calls")
	}
	if sess.Iterations() != 1 {
		t.Fatalf("expected Iterations=1, got %d", sess.Iterations())
	}
	if sess.Duration() <= 0 {
		t.Fatalf("expected positive duration, got %v", sess.Duration())
	}
}

func TestSessionAppendSystemAndUserMessages(t *testing.T) {
	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "u"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "a",
		},
	})
	sess.AddStep(step)

	sess.AppendSystemMessage("sys")
	sess.AppendUserMessage("u2")

	input := sess.LastStep().GetInput()
	if len(input) != 1 {
		t.Fatalf("expected 1 message in step input, got %d", len(input))
	}

	appended := sess.LastStep().GetAppended()
	if len(appended) != 2 {
		t.Fatalf("expected 2 appended messages, got %d", len(appended))
	}
	if appended[0].Role != model.RoleSystem || appended[0].Content != "sys" {
		t.Errorf("expected first appended message to be system 'sys', got %#v", appended[0])
	}
	if appended[1].Role != model.RoleUser || appended[1].Content != "u2" {
		t.Errorf("expected second appended message to be user 'u2', got %#v", appended[1])
	}

	// Ensure conversation includes both input and output messages.
	conv := sess.Conversation()
	if len(conv) != 4 {
		t.Fatalf("expected 4 messages in conversation, got %d", len(conv))
	}
}

func TestSessionJSONRoundTrip(t *testing.T) {
	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "u"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "a",
		},
	})
	sess.AddStep(step)

	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}

	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal session: %v", err)
	}

	if decoded.Iterations() != 1 {
		t.Fatalf("expected 1 iteration after round-trip, got %d", decoded.Iterations())
	}
	if !decoded.Finished() {
		t.Fatalf("expected Finished=true after round-trip")
	}
}

func TestSessionConversation_OrderWithAppended(t *testing.T) {
	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "u"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "a",
		},
		ToolCalls: []model.ToolCall{{Name: "tool1"}}, // simulate tool calls happened
	})
	sess.AddStep(step)

	// Simulate tool output + extractor feedback appended after assistant response
	sess.AppendSystemMessage("tool-output-ish")
	sess.AppendSystemMessage("feedback-ish")

	conv := sess.Conversation()
	if len(conv) != 4 {
		t.Fatalf("expected 4 messages in conversation, got %d", len(conv))
	}
	if conv[0].Role != model.RoleUser || conv[0].Content != "u" {
		t.Fatalf("expected conv[0] to be user 'u', got %#v", conv[0])
	}
	if conv[1].Role != model.RoleAssistant || conv[1].Content != "a" {
		t.Fatalf("expected conv[1] to be assistant 'a', got %#v", conv[1])
	}
	if conv[2].Role != model.RoleSystem || conv[2].Content != "tool-output-ish" {
		t.Fatalf("expected conv[2] to be system 'tool-output-ish', got %#v", conv[2])
	}
	if conv[3].Role != model.RoleSystem || conv[3].Content != "feedback-ish" {
		t.Fatalf("expected conv[3] to be system 'feedback-ish', got %#v", conv[3])
	}
}


