package executor

import (
	"fmt"
	"testing"

	"github.com/hlfshell/gotonomy/agent/judging"
	"github.com/hlfshell/gotonomy/agent/planning"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/plan"
	"github.com/hlfshell/gotonomy/tool"
)

func TestExecutor_Pass(t *testing.T) {
	p := plan.NewPlan("p1")
	s1 := plan.NewStep("s1", "S1", "do s1", "ok", nil, nil)
	s2 := plan.NewStep("s2", "S2", "do s2", "ok", []*plan.Step{&s1}, nil)
	p.AddStep(s1)
	p.AddStep(s2)

	runner := tool.NewTool[string](
		"runner",
		"fake runner",
		executorRunnerParams(),
		func(ctx *tool.Context, args tool.Arguments) (string, error) {
			_ = ctx
			_ = args
			return "ok", nil
		},
	)

	judge := tool.NewTool[judging.JudgeResult](
		"judge",
		"fake judge",
		executorJudgeParams(),
		func(ctx *tool.Context, args tool.Arguments) (judging.JudgeResult, error) {
			_ = ctx
			exp := args["expectation"].(string)
			out := args["output"].(string)
			if exp == "ok" && out == "ok" {
				return judging.JudgeResult{Verdict: judging.VerdictPass, Justification: "ok"}, nil
			}
			return judging.JudgeResult{Verdict: judging.VerdictFail, Justification: "not ok", SuggestedFix: "make it ok"}, nil
		},
	)

	exec := &Executor{
		Config:     ExecutorConfig{MaxAttemptsPerStep: 1, AllowReplan: false},
		StepRunner: runner,
		Judge:      judge,
		Planner:    nil,
	}

	report, err := exec.Execute(nil, p, "objective")
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if report == nil {
		t.Fatalf("expected report")
	}
	if len(report.Steps) != 2 {
		t.Fatalf("expected 2 step executions, got %d", len(report.Steps))
	}
	if report.Steps[0].FinalVerdict != judging.VerdictPass {
		t.Fatalf("expected step1 pass, got %q", report.Steps[0].FinalVerdict)
	}
	if report.Steps[1].FinalVerdict != judging.VerdictPass {
		t.Fatalf("expected step2 pass, got %q", report.Steps[1].FinalVerdict)
	}
}

func TestExecutor_FailAfterAttempts(t *testing.T) {
	p := plan.NewPlan("p1")
	s1 := plan.NewStep("s1", "S1", "do s1", "ok", nil, nil)
	p.AddStep(s1)

	runner := tool.NewTool[string](
		"runner",
		"fake runner",
		executorRunnerParams(),
		func(ctx *tool.Context, args tool.Arguments) (string, error) {
			_ = ctx
			_ = args
			return "bad", nil
		},
	)

	judge := tool.NewTool[judging.JudgeResult](
		"judge",
		"fake judge",
		executorJudgeParams(),
		func(ctx *tool.Context, args tool.Arguments) (judging.JudgeResult, error) {
			_ = ctx
			_ = args
			return judging.JudgeResult{Verdict: judging.VerdictFail, Justification: "nope", SuggestedFix: "try harder"}, nil
		},
	)

	exec := &Executor{
		Config:     ExecutorConfig{MaxAttemptsPerStep: 2, AllowReplan: false},
		StepRunner: runner,
		Judge:      judge,
	}

	report, err := exec.Execute(nil, p, "objective")
	if err == nil {
		t.Fatalf("expected error")
	}
	if report == nil || len(report.Steps) != 1 {
		t.Fatalf("expected report with 1 step, got %#v", report)
	}
	if len(report.Steps[0].Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(report.Steps[0].Attempts))
	}
}

func TestExecutor_Replan(t *testing.T) {
	// Initial plan: expectation triggers replan.
	p := plan.NewPlan("p1")
	s1 := plan.NewStep("s1", "S1", "do s1", "needs_replan", nil, nil)
	p.AddStep(s1)

	runner := tool.NewTool[string](
		"runner",
		"fake runner",
		executorRunnerParams(),
		func(ctx *tool.Context, args tool.Arguments) (string, error) {
			_ = ctx
			_ = args
			return "ok", nil
		},
	)

	judge := tool.NewTool[judging.JudgeResult](
		"judge",
		"fake judge",
		executorJudgeParams(),
		func(ctx *tool.Context, args tool.Arguments) (judging.JudgeResult, error) {
			_ = ctx
			exp := args["expectation"].(string)
			if exp == "ok" {
				return judging.JudgeResult{Verdict: judging.VerdictPass, Justification: "ok"}, nil
			}
			return judging.JudgeResult{Verdict: judging.VerdictReplan, Justification: "need different plan", SuggestedFix: "change expectation to ok"}, nil
		},
	)

	// Planner that always returns a plan with expectation "ok".
	call := 0
	pm := &plannerMockModel{
		complete: func(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
			_ = ctx
			_ = req
			call++
			return model.CompletionResponse{
				Text: `{"steps":[{"id":"s1","name":"S1","instruction":"do s1","expectation":"ok","dependencies":[]}]}`,
			}, nil
		},
	}
	planner, err := planning.NewPlannerAgent("planner", "planner", "planner", planning.Config{Model: pm, Temperature: 0.0})
	if err != nil {
		t.Fatalf("failed to create planner: %v", err)
	}

	exec := &Executor{
		Config:     ExecutorConfig{MaxAttemptsPerStep: 1, AllowReplan: true, MaxReplans: 2},
		StepRunner: runner,
		Judge:      judge,
		Planner:    planner,
	}

	report, err := exec.Execute(nil, p, "objective")
	if err != nil {
		t.Fatalf("expected success after replan, got err: %v", err)
	}
	if report == nil {
		t.Fatalf("expected report")
	}
	if len(report.Replans) != 1 {
		t.Fatalf("expected 1 replan, got %d", len(report.Replans))
	}
	if call == 0 {
		t.Fatalf("expected planner to be called")
	}
	if report.Steps[len(report.Steps)-1].FinalVerdict != judging.VerdictPass {
		t.Fatalf("expected final verdict pass, got %q", report.Steps[len(report.Steps)-1].FinalVerdict)
	}
}

type plannerMockModel struct {
	complete func(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error)
}

func (m *plannerMockModel) Description() model.ModelDescription {
	return model.ModelDescription{
		Model:            "mock",
		Provider:         "mock",
		MaxContextTokens: 8192,
		Description:      "mock",
		CanUseTools:      false,
	}
}

func (m *plannerMockModel) Complete(ctx *tool.Context, req model.CompletionRequest) (model.CompletionResponse, error) {
	return m.complete(ctx, req)
}

func executorRunnerParams() []tool.Parameter {
	return []tool.Parameter{
		tool.NewParameter[string]("objective", "objective", true, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[*plan.Plan]("plan", "plan", true, nil, func(v *plan.Plan) (string, error) {
			if v == nil {
				return "", fmt.Errorf("plan cannot be nil")
			}
			return v.ID, nil
		}),
		tool.NewParameter[*plan.Step]("step", "step", true, nil, func(v *plan.Step) (string, error) {
			if v == nil {
				return "", fmt.Errorf("step cannot be nil")
			}
			return v.ID, nil
		}),
		tool.NewParameter[int]("attempt", "attempt", false, 1, func(v int) (string, error) { return "1", nil }),
		tool.NewParameter[string]("prior_feedback", "prior_feedback", false, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[[]DependencyOutput]("dependency_outputs", "dependency_outputs", false, nil, func(v []DependencyOutput) (string, error) {
			if len(v) == 0 {
				return "", nil
			}
			return fmt.Sprintf("%d outputs", len(v)), nil
		}),
	}
}

func executorJudgeParams() []tool.Parameter {
	return []tool.Parameter{
		tool.NewParameter[string]("objective", "objective", true, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[string]("step_name", "step_name", false, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[string]("instruction", "instruction", true, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[string]("expectation", "expectation", true, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[string]("output", "output", true, "", func(v string) (string, error) { return v, nil }),
		tool.NewParameter[string]("context", "context", false, "", func(v string) (string, error) { return v, nil }),
	}
}
