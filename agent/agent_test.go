package agent

import (
	"context"
	"testing"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

// mockModel implements model.Model for testing Agent behavior.
type mockModel struct {
	desc      model.ModelDescription
	responses []model.CompletionResponse
	err       error

	calls    int
	requests []model.CompletionRequest
}

func (m *mockModel) Description() model.ModelDescription {
	return m.desc
}

func (m *mockModel) Complete(ctx context.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
	m.requests = append(m.requests, req)
	m.calls++

	if m.err != nil {
		return model.CompletionResponse{}, m.err
	}
	if len(m.responses) == 0 {
		return model.CompletionResponse{}, nil
	}

	idx := m.calls - 1
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	return m.responses[idx], nil
}

// Ensure mockModel satisfies the model.Model interface.
var _ model.Model = (*mockModel)(nil)

// TestExecute_NoTools_DefaultExtractorReturnsText verifies that the default
// extractor returns the raw model text when there are no tool calls and no
// custom parser/extractor is configured.
func TestExecute_NoTools_DefaultExtractorReturnsText(t *testing.T) {
	m := &mockModel{
		desc: model.ModelDescription{
			Model:            "test-model",
			Provider:         "test",
			MaxContextTokens: 1024,
		},
		responses: []model.CompletionResponse{
			{Text: "hello world"},
		},
	}

	agent := NewAgent("test-agent", "Test Agent", m)

	result := agent.Execute(nil, tool.Arguments{
		"input": "ignored",
	})

	if result.Errored() {
		t.Fatalf("expected non-error result, got: %v", result.GetError())
	}

	val, ok := result.GetResult().(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result.GetResult())
	}
	if val != "hello world" {
		t.Errorf("expected result %q, got %q", "hello world", val)
	}
}

// TestExecute_WithParser_UsesExtractorFromParser verifies that providing a
// ResponseParser via WithParser results in the extractor returning the parsed
// value.
func TestExecute_WithParser_UsesExtractorFromParser(t *testing.T) {
	m := &mockModel{
		desc: model.ModelDescription{
			Model:            "test-model",
			Provider:         "test",
			MaxContextTokens: 1024,
		},
		responses: []model.CompletionResponse{
			{Text: "body"},
		},
	}

	parser := func(output string) (any, error) {
		return map[string]string{"wrapped": output}, nil
	}

	agent := NewAgent(
		"parser-agent",
		"Parser Agent",
		m,
		WithParser(parser),
	)

	result := agent.Execute(nil, tool.Arguments{
		"input": "ignored",
	})

	if result.Errored() {
		t.Fatalf("expected non-error result, got: %v", result.GetError())
	}

	val, ok := result.GetResult().(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string result, got %T", result.GetResult())
	}
	if val["wrapped"] != "body" {
		t.Errorf("expected wrapped value %q, got %q", "body", val["wrapped"])
	}
}

// TestExecute_CustomExtractorFeedback verifies that a custom ExtractResult
// implementation can inject feedback messages and control when the agent
// stops iterating.
func TestExecute_CustomExtractorFeedback(t *testing.T) {
	m := &mockModel{
		desc: model.ModelDescription{
			Model:            "test-model",
			Provider:         "test",
			MaxContextTokens: 1024,
		},
		responses: []model.CompletionResponse{
			{Text: "first"},
			{Text: "second"},
		},
	}

	var extractCalls int
	const feedbackText = "please refine your answer"

	extractor := func(a *Agent, ctx *tool.Context, sess *Session) ExtractDecision {
		extractCalls++

		// On first call, request another iteration with system feedback.
		if extractCalls == 1 {
			return ExtractDecision{
				Done: false,
				Feedback: []model.Message{
					{
						Role:    model.RoleSystem,
						Content: feedbackText,
					},
				},
			}
		}

		// On second call, stop and return a final value.
		return ExtractDecision{
			Done:   true,
			Result: "final-result",
		}
	}

	agent := NewAgent(
		"feedback-agent",
		"Feedback Agent",
		m,
		WithExtractor(extractor),
	)

	result := agent.Execute(nil, tool.Arguments{
		"input": "ignored",
	})

	if result.Errored() {
		t.Fatalf("expected non-error result, got: %v", result.GetError())
	}

	if extractCalls != 2 {
		t.Fatalf("expected extractor to be called twice, got %d", extractCalls)
	}

	val, ok := result.GetResult().(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result.GetResult())
	}
	if val != "final-result" {
		t.Errorf("expected result %q, got %q", "final-result", val)
	}

	// Verify that the feedback message was included in the second model call.
	if len(m.requests) < 2 {
		t.Fatalf("expected at least 2 model requests, got %d", len(m.requests))
	}
	secondReq := m.requests[1]

	foundFeedback := false
	for _, msg := range secondReq.Messages {
		if msg.Role == model.RoleSystem && msg.Content == feedbackText {
			foundFeedback = true
			break
		}
	}
	if !foundFeedback {
		t.Errorf("expected second request to include system message %q", feedbackText)
	}
}

// TestIterationChecker_MaxSteps verifies that AgentConfig's IterationChecker
// returns an error once the number of steps reaches MaxSteps.
func TestIterationChecker_MaxSteps(t *testing.T) {
	cfg := AgentConfig{
		MaxSteps: 1,
	}
	check := cfg.IterationChecker()

	sess := NewSession()
	if err := check(sess); err != nil {
		t.Fatalf("unexpected error for empty session: %v", err)
	}

	// After adding one step, we should hit the max.
	step := NewStep(nil)
	step.SetResponse(Response{
		Output: model.Message{
			Role:    model.RoleAssistant,
			Content: "done",
		},
	})
	sess.AddStep(step)

	if err := check(sess); err == nil {
		t.Fatalf("expected error after reaching max steps")
	}
}

// TestToolsSlice_Sorted ensures toolsSlice returns tools sorted by name.
func TestToolsSlice_Sorted(t *testing.T) {
	a := &Agent{
		tools: map[string]tool.Tool{
			"beta":  newMockTool("beta", nil),
			"alpha": newMockTool("alpha", nil),
		},
	}

	slice := a.toolsSlice()
	if len(slice) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(slice))
	}
	if slice[0].Name() != "alpha" || slice[1].Name() != "beta" {
		t.Fatalf("expected tools in order [alpha, beta], got [%s, %s]", slice[0].Name(), slice[1].Name())
	}
}

// TestResponseFromModel verifies that ResponseFromModel copies text and tool
// calls correctly from the model.CompletionResponse.
func TestResponseFromModel(t *testing.T) {
	resp := model.CompletionResponse{
		Text: "answer",
		ToolCalls: []model.ToolCall{
			{Name: "tool1"},
		},
	}

	r := ResponseFromModel(resp)
	if r.Output.Content != "answer" {
		t.Fatalf("expected Output.Content=%q, got %q", "answer", r.Output.Content)
	}
	if len(r.ToolCalls) != 1 || r.ToolCalls[0].Name != "tool1" {
		t.Fatalf("unexpected ToolCalls: %#v", r.ToolCalls)
	}
}
