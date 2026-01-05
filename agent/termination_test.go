package agent

import (
	"errors"
	"testing"

	"github.com/hlfshell/gotonomy/model"
)

// TestExtractorFromParser_PendingToolCalls ensures that when the last step
// still has tool calls, the extractor signals that the agent is not done.
func TestExtractorFromParser_PendingToolCalls(t *testing.T) {
	parser := func(s string) (any, error) {
		t.Fatalf("parser should not be called when tool calls are pending")
		return nil, nil
	}

	extractor := ExtractorFromParser(parser, false)

	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "hi"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "tool call",
		},
		ToolCalls: []model.ToolCall{
			{Name: "tool1"},
		},
	})
	sess.AddStep(step)

	dec := extractor(nil, nil, sess)
	if dec.Done {
		t.Fatalf("expected Done=false when tool calls are pending")
	}
	if dec.Result != nil {
		t.Fatalf("expected no result when tool calls are pending, got %v", dec.Result)
	}
}

// TestExtractorFromParser_Success verifies that when there are no tool calls
// and the parser succeeds, the extractor returns Done=true with the parsed
// value and no warnings.
func TestExtractorFromParser_Success(t *testing.T) {
	parser := func(s string) (any, error) {
		return map[string]string{"seen": s}, nil
	}
	extractor := ExtractorFromParser(parser, false)

	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "hi"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "body",
		},
	})
	sess.AddStep(step)

	dec := extractor(nil, nil, sess)
	if !dec.Done {
		t.Fatalf("expected Done=true")
	}
	if len(dec.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", dec.Warnings)
	}

	m, ok := dec.Result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string result, got %T", dec.Result)
	}
	if m["seen"] != "body" {
		t.Fatalf("expected parsed value %q, got %q", "body", m["seen"])
	}
}

// TestExtractorFromParser_ParseError_NoRetry verifies that when the parser
// fails and retryOnError is false, the extractor still marks Done=true and
// surfaces a warning.
func TestExtractorFromParser_ParseError_NoRetry(t *testing.T) {
	parserErr := errors.New("parse failed")
	parser := func(s string) (any, error) {
		return nil, parserErr
	}
	extractor := ExtractorFromParser(parser, false)

	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "hi"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "bad",
		},
	})
	sess.AddStep(step)

	dec := extractor(nil, nil, sess)
	if !dec.Done {
		t.Fatalf("expected Done=true when retryOnError=false")
	}
	if len(dec.Warnings) == 0 {
		t.Fatalf("expected warnings when parser errors")
	}
}

// TestExtractorFromParser_ParseError_Retry verifies that when the parser
// fails and retryOnError is true, the extractor leaves Done=false so the
// agent can iterate again.
func TestExtractorFromParser_ParseError_Retry(t *testing.T) {
	parserErr := errors.New("parse failed")
	parser := func(s string) (any, error) {
		return nil, parserErr
	}
	extractor := ExtractorFromParser(parser, true)

	sess := NewSession()
	step := NewStep([]model.Message{
		{Role: model.RoleUser, Content: "hi"},
	})
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "bad",
		},
	})
	sess.AddStep(step)

	dec := extractor(nil, nil, sess)
	if dec.Done {
		t.Fatalf("expected Done=false when retryOnError=true")
	}
	if len(dec.Warnings) == 0 {
		t.Fatalf("expected warnings when parser errors")
	}
}


