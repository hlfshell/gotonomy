package judging

import (
	"testing"

	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

type mockModel struct {
	complete func(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error)
}

func (m *mockModel) Description() model.ModelDescription {
	return model.ModelDescription{
		Model:            "mock",
		Provider:         "mock",
		MaxContextTokens: 8192,
		Description:      "mock model",
		CanUseTools:      false,
	}
}

func (m *mockModel) Complete(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
	return m.complete(ctx, req)
}

func TestJudgeAgent_Pass(t *testing.T) {
	m := &mockModel{
		complete: func(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
			_ = ctx
			_ = req
			return model.CompletionResponse{
				Text: `{"verdict":"pass","justification":"Matches expectation."}`,
			}, nil
		},
	}

	judge := NewJudgeAgent(m)
	res := judge.Execute(nil, tool.Arguments{
		"objective":   "Do a thing",
		"step_name":   "Test step",
		"instruction": "Produce output X",
		"expectation": "Output contains X",
		"output":      "X",
	})

	if res.Errored() {
		t.Fatalf("expected ok, got error: %v", res.GetError())
	}

	jr, ok := res.GetResult().(JudgeResult)
	if !ok {
		t.Fatalf("expected JudgeResult, got %T", res.GetResult())
	}
	if jr.Verdict != VerdictPass {
		t.Fatalf("expected pass, got %q", jr.Verdict)
	}
	if jr.Justification == "" {
		t.Fatalf("expected justification")
	}
}

func TestJudgeAgent_RetryOnInvalidJSON(t *testing.T) {
	call := 0
	m := &mockModel{
		complete: func(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
			_ = ctx
			_ = req
			call++
			if call == 1 {
				return model.CompletionResponse{Text: "not json"}, nil
			}
			return model.CompletionResponse{Text: `{"verdict":"fail","justification":"Does not match expectation.","suggested_fix":"Include X."}`}, nil
		},
	}

	judge := NewJudgeAgent(m)
	res := judge.Execute(nil, tool.Arguments{
		"objective":   "Do a thing",
		"step_name":   "Test step",
		"instruction": "Produce output X",
		"expectation": "Output contains X",
		"output":      "Y",
	})

	if res.Errored() {
		t.Fatalf("expected ok, got error: %v", res.GetError())
	}
	if call < 2 {
		t.Fatalf("expected retry, model calls=%d", call)
	}

	jr, ok := res.GetResult().(JudgeResult)
	if !ok {
		t.Fatalf("expected JudgeResult, got %T", res.GetResult())
	}
	if jr.Verdict != VerdictFail {
		t.Fatalf("expected fail, got %q", jr.Verdict)
	}
	if jr.SuggestedFix == "" {
		t.Fatalf("expected suggested_fix")
	}
}
