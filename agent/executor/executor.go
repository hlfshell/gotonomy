package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hlfshell/gotonomy/agent/judging"
	"github.com/hlfshell/gotonomy/agent/planning"
	"github.com/hlfshell/gotonomy/assets"
	"github.com/hlfshell/gotonomy/plan"
	"github.com/hlfshell/gotonomy/tool"
)

type ExecutorConfig struct {
	// MaxAttemptsPerStep controls how many times we retry a failed step before aborting.
	MaxAttemptsPerStep int

	// AllowReplan enables calling the planner when the judge returns verdict=replan.
	AllowReplan bool

	// MaxReplans is the maximum number of replans allowed during a single execution.
	MaxReplans int
}

func (c ExecutorConfig) withDefaults() ExecutorConfig {
	if c.MaxAttemptsPerStep <= 0 {
		c.MaxAttemptsPerStep = 2
	}
	if c.MaxReplans <= 0 {
		c.MaxReplans = 2
	}
	return c
}

// StepAttempt records one attempt at executing a single step.
type StepAttempt struct {
	Attempt     int                 `json:"attempt"`
	StartedAt   time.Time           `json:"started_at"`
	EndedAt     time.Time           `json:"ended_at"`
	Output      string              `json:"output"`
	JudgeResult judging.JudgeResult `json:"judge_result"`
}

// StepExecution is the full execution record for a single step, including attempts.
type StepExecution struct {
	StepID       string           `json:"step_id"`
	StepName     string           `json:"step_name"`
	Instruction  string           `json:"instruction"`
	Expectation  string           `json:"expectation"`
	StartedAt    time.Time        `json:"started_at"`
	EndedAt      time.Time        `json:"ended_at"`
	Attempts     []StepAttempt    `json:"attempts"`
	SubPlan      *ExecutionReport `json:"sub_plan,omitempty"`
	FinalVerdict judging.Verdict  `json:"final_verdict"`
}

// ExecutionReport is a JSON-serializable record of executing a plan.
type ExecutionReport struct {
	Objective string          `json:"objective"`
	PlanID    string          `json:"plan_id"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at"`
	Steps     []StepExecution `json:"steps"`
	Replans   []plan.PlanDiff `json:"replans,omitempty"`
	FinalPlan *plan.Plan      `json:"final_plan,omitempty"`
}

func (r *ExecutionReport) Duration() time.Duration {
	if r == nil {
		return 0
	}
	if r.EndedAt.IsZero() {
		return time.Since(r.StartedAt)
	}
	return r.EndedAt.Sub(r.StartedAt)
}

// Executor runs plan steps using a step runner and validates results with a judge.
type Executor struct {
	Config ExecutorConfig

	// StepRunner is a tool that executes a step instruction and returns a string output.
	StepRunner tool.Tool

	// Judge is a tool that returns judging.JudgeResult based on expectation vs output.
	Judge tool.Tool

	// Planner is optional; used only when AllowReplan=true and judge returns verdict=replan.
	Planner *planning.PlannerAgent
}

func (e *Executor) Execute(ctx *tool.Context, p *plan.Plan, objective string) (*ExecutionReport, error) {
	if p == nil {
		return nil, fmt.Errorf("plan is nil")
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}
	if e.StepRunner == nil {
		return nil, fmt.Errorf("executor missing StepRunner")
	}
	if e.Judge == nil {
		return nil, fmt.Errorf("executor missing Judge")
	}
	cfg := e.Config.withDefaults()

	report := &ExecutionReport{
		Objective: objective,
		PlanID:    p.ID,
		StartedAt: time.Now(),
		Steps:     []StepExecution{},
		Replans:   []plan.PlanDiff{},
		FinalPlan: nil,
	}

	replans := 0

	// Execute to completion. Replanning is handled by mutating *p in-place.
	steps, err := e.executePlan(ctx, p, objective, cfg, &replans, &report.Replans)
	report.Steps = append(report.Steps, steps...)
	report.FinalPlan = p
	report.PlanID = p.ID
	report.EndedAt = time.Now()
	if err != nil {
		return report, err
	}
	return report, nil
}

func (e *Executor) executePlan(
	ctx *tool.Context,
	p *plan.Plan,
	objective string,
	cfg ExecutorConfig,
	replans *int,
	replanCollector *[]plan.PlanDiff,
) ([]StepExecution, error) {
	completed := map[string]bool{}
	stepOutputs := make(map[string]string) // step ID -> output
	out := make([]StepExecution, 0, len(p.Steps))

	for len(completed) < len(p.Steps) {
		readyAll := p.NextSteps(completed)
		ready := make([]*plan.Step, 0, len(readyAll))
		for _, s := range readyAll {
			if !completed[s.ID] {
				ready = append(ready, s)
			}
		}
		if len(ready) == 0 {
			return out, fmt.Errorf("no executable steps available; remaining=%d", len(p.Steps)-len(completed))
		}

		// Deterministic: execute the first ready step (plan order).
		step := ready[0]

		// Collect outputs from dependencies
		dependencyOutputs := e.collectDependencyOutputs(*step, stepOutputs)

		stepExec, verdict, replanDiff, newPlan, err := e.executeStep(ctx, p, step, objective, cfg, dependencyOutputs)
		out = append(out, stepExec)

		// Check if this is an escalated replan from a sub-plan (before general error handling)
		isEscalated := err != nil && strings.HasPrefix(err.Error(), "escalated: ")

		if err != nil && !isEscalated {
			return out, err
		}

		switch verdict {
		case judging.VerdictPass:
			completed[step.ID] = true
			// Store the output from the last successful attempt
			if len(stepExec.Attempts) > 0 {
				lastAttempt := stepExec.Attempts[len(stepExec.Attempts)-1]
				stepOutputs[step.ID] = lastAttempt.Output
			}

		case judging.VerdictFail:
			return out, fmt.Errorf("step %s failed after %d attempt(s)", step.ID, len(stepExec.Attempts))

		case judging.VerdictReplan:
			if !cfg.AllowReplan {
				return out, fmt.Errorf("step %s requested replan but replanning is disabled", step.ID)
			}
			if e.Planner == nil {
				return out, fmt.Errorf("step %s requested replan but executor has no Planner", step.ID)
			}

			// Handle escalated replan from a sub-plan
			if isEscalated {
				// Extract escalation feedback
				escalationFeedback := strings.TrimPrefix(err.Error(), "escalated: ")

				// Replan at this (parent) level
				*replans++
				if *replans > cfg.MaxReplans {
					return out, fmt.Errorf("exceeded max replans (%d)", cfg.MaxReplans)
				}

				// Save old plan state for incremental replanning
				oldPlan := *p
				oldCompleted := make(map[string]bool)
				oldStepOutputs := make(map[string]string)
				for k, v := range completed {
					oldCompleted[k] = v
				}
				for k, v := range stepOutputs {
					oldStepOutputs[k] = v
				}

				res, replanErr := e.Planner.Replan(ctx, p, escalationFeedback, planning.PlannerInput{
					Objective: objective,
					Context:   escalationFeedback,
				})
				if replanErr != nil {
					return out, fmt.Errorf("escalated replan failed: %w", replanErr)
				}

				if replanCollector != nil && res.Plan.RevisionDiff != nil {
					*replanCollector = append(*replanCollector, *res.Plan.RevisionDiff)
				}
				if res.Plan == nil {
					return out, fmt.Errorf("escalated replan returned nil plan")
				}

				*p = *res.Plan
				// Apply incremental replan: preserve outputs from unchanged steps
				completed, stepOutputs = applyIncrementalReplan(
					&oldPlan,
					res.Plan,
					res.Plan.RevisionDiff,
					oldCompleted,
					oldStepOutputs,
				)
				continue
			}

			// Normal replan handling
			*replans++
			if *replans > cfg.MaxReplans {
				return out, fmt.Errorf("exceeded max replans (%d)", cfg.MaxReplans)
			}
			if replanDiff != nil && replanCollector != nil {
				*replanCollector = append(*replanCollector, *replanDiff)
			}
			if newPlan == nil {
				return out, fmt.Errorf("replan requested but new plan is nil")
			}

			// Save old plan state for incremental replanning
			oldPlan := *p
			oldCompleted := make(map[string]bool)
			oldStepOutputs := make(map[string]string)
			for k, v := range completed {
				oldCompleted[k] = v
			}
			for k, v := range stepOutputs {
				oldStepOutputs[k] = v
			}

			// Swap to the new plan and apply incremental replan
			*p = *newPlan
			completed, stepOutputs = applyIncrementalReplan(
				&oldPlan,
				newPlan,
				replanDiff,
				oldCompleted,
				oldStepOutputs,
			)
		default:
			return out, fmt.Errorf("unknown verdict %q", verdict)
		}
	}

	return out, nil
}

// DependencyOutput represents the output from a step that the current step depends on.
type DependencyOutput struct {
	StepID      string
	StepName    string
	Instruction string
	Output      string
}

// collectDependencyOutputs collects outputs from steps that the given step depends on.
// It handles both direct dependencies and dependencies through sub-plans.
func (e *Executor) collectDependencyOutputs(step plan.Step, stepOutputs map[string]string) []DependencyOutput {
	if len(step.Dependencies) == 0 {
		return nil
	}

	outputs := make([]DependencyOutput, 0, len(step.Dependencies))
	for _, dep := range step.Dependencies {
		if dep == nil {
			continue
		}

		// Check if we have the output for this dependency
		if output, ok := stepOutputs[dep.ID]; ok {
			outputs = append(outputs, DependencyOutput{
				StepID:      dep.ID,
				StepName:    dep.Name,
				Instruction: dep.Instruction,
				Output:      output,
			})
		}
		// If the dependency had a sub-plan, the output would be the sub-plan's execution summary
		// which is already stored in stepOutputs when the sub-plan step passed
	}

	return outputs
}

// instructionHash computes a hash of a step's instruction for change detection.
// Used to determine if a step's instruction changed during replanning.
func instructionHash(stepID, instruction string) string {
	h := sha256.Sum256([]byte(stepID + ":" + instruction))
	return hex.EncodeToString(h[:])
}

// applyIncrementalReplan applies a replan while preserving outputs from unchanged steps.
// Returns updated completed map and stepOutputs map with preserved values.
func applyIncrementalReplan(
	oldPlan *plan.Plan,
	newPlan *plan.Plan,
	replanDiff *plan.PlanDiff,
	oldCompleted map[string]bool,
	oldStepOutputs map[string]string,
) (map[string]bool, map[string]string) {
	// Start with empty state
	newCompleted := make(map[string]bool)
	newStepOutputs := make(map[string]string)

	// If no diff, preserve everything
	if replanDiff == nil {
		return oldCompleted, oldStepOutputs
	}

	// Build maps of old steps by ID for quick lookup
	oldStepMap := make(map[string]*plan.Step)
	if oldPlan != nil {
		for i := range oldPlan.Steps {
			oldStepMap[oldPlan.Steps[i].ID] = &oldPlan.Steps[i]
		}
	}

	// Build maps of new steps by ID
	newStepMap := make(map[string]*plan.Step)
	if newPlan != nil {
		for i := range newPlan.Steps {
			newStepMap[newPlan.Steps[i].ID] = &newPlan.Steps[i]
		}
	}

	// Preserve outputs for steps that:
	// 1. Exist in both old and new plans (not removed, not added)
	// 2. Have the same ID and instruction (unchanged)
	// 3. Were completed in the old plan
	for stepID, wasCompleted := range oldCompleted {
		if !wasCompleted {
			continue
		}

		oldStep, oldExists := oldStepMap[stepID]
		newStep, newExists := newStepMap[stepID]

		// Step must exist in both plans
		if !oldExists || !newExists {
			continue
		}

		// Check if step was changed (in the diff's Changed map)
		if change, wasChanged := replanDiff.Steps.Changed[stepID]; wasChanged {
			// Step was changed - check if instruction is the same
			oldHash := instructionHash(change.From.ID, change.From.Instruction)
			newHash := instructionHash(change.To.ID, change.To.Instruction)
			if oldHash == newHash {
				// Instruction unchanged, preserve output
				if output, ok := oldStepOutputs[stepID]; ok {
					newStepOutputs[stepID] = output
					newCompleted[stepID] = true
				}
			}
			// If instruction changed, don't preserve
		} else {
			// Step not in Changed map - check if it's truly unchanged
			// (not in Added or Removed)
			if _, wasAdded := replanDiff.Steps.Added[stepID]; wasAdded {
				continue
			}
			if _, wasRemoved := replanDiff.Steps.Removed[stepID]; wasRemoved {
				continue
			}

			// Step exists in both and wasn't explicitly changed/added/removed
			// Double-check instruction hasn't changed
			if oldStep.Instruction == newStep.Instruction {
				// Preserve output
				if output, ok := oldStepOutputs[stepID]; ok {
					newStepOutputs[stepID] = output
					newCompleted[stepID] = true
				}
			}
		}
	}

	return newCompleted, newStepOutputs
}

// shouldEscalateReplan determines if a replan request from a sub-plan should
// be escalated to the parent plan level. Returns true if escalation is needed.
//
// NOTE: This function reuses the JudgeAgent with a special escalation prompt.
// The escalation prompt uses the same JSON schema as the judge (JudgeResult),
// but with different semantic interpretation:
//   - Verdict "replan" = ESCALATE to parent (structural issue requiring parent-level changes)
//   - Verdict "pass" or "fail" = LOCAL replan is sufficient (issue can be fixed in sub-plan)
//
// This semantic overload is intentional for code reuse, but be aware that "replan"
// means different things in different contexts:
//   - In normal judging: "the step/plan structure needs changes"
//   - In escalation context: "this problem should bubble up to the parent plan"
func (e *Executor) shouldEscalateReplan(
	ctx *tool.Context,
	parentStep *plan.Step,
	parentPlan *plan.Plan,
	subPlan *plan.Plan,
	replanReason string,
	objective string,
) (bool, string, error) {
	if e.Judge == nil {
		return false, "", nil // No judge, default to local replan
	}

	// Load the escalation prompt template
	tmpl, err := assets.LoadPrompt("escalation.prompt")
	if err != nil {
		return false, "", fmt.Errorf("failed to load escalation prompt: %w", err)
	}

	// Render the escalation prompt
	templateData := map[string]interface{}{
		"objective":               objective,
		"parent_step_name":        parentStep.Name,
		"parent_step_instruction": parentStep.Instruction,
		"parent_step_expectation": parentStep.Expectation,
		"sub_plan_structure":      subPlan.ToText(),
		"replan_reason":           replanReason,
	}

	rendered, err := tmpl.Render(templateData)
	if err != nil {
		return false, "", fmt.Errorf("failed to render escalation prompt: %w", err)
	}

	// Use judge to determine escalation
	// The escalation prompt uses the same JSON schema as judge, but interprets:
	// - "replan" verdict = escalate to parent (structural issue)
	// - "pass" or "fail" verdict = local replan is fine
	judgeRes := e.Judge.Execute(ctx, tool.Arguments{
		"objective":   objective,
		"step_name":   parentStep.Name,
		"instruction": parentStep.Instruction,
		"expectation": parentStep.Expectation,
		"output":      fmt.Sprintf("Sub-plan execution requested replan: %s", replanReason),
		"context":     rendered,
	})

	if judgeRes.Errored() {
		return false, "", judgeRes.GetError()
	}

	jr, ok := judgeRes.GetResult().(judging.JudgeResult)
	if !ok {
		return false, "", fmt.Errorf("judge returned unexpected type %T", judgeRes.GetResult())
	}

	// Semantic mapping: "replan" verdict in escalation context means "escalate to parent"
	// "pass" or "fail" means "local replan is fine"
	shouldEscalate := jr.Verdict == judging.VerdictReplan

	return shouldEscalate, jr.Justification, nil
}

func (e *Executor) executeStep(
	ctx *tool.Context,
	currentPlan *plan.Plan,
	step *plan.Step,
	objective string,
	cfg ExecutorConfig,
	dependencyOutputs []DependencyOutput,
) (StepExecution, judging.Verdict, *plan.PlanDiff, *plan.Plan, error) {
	stepExec := StepExecution{
		StepID:       step.ID,
		StepName:     step.Name,
		Instruction:  step.Instruction,
		Expectation:  step.Expectation,
		StartedAt:    time.Now(),
		Attempts:     []StepAttempt{},
		SubPlan:      nil,
		FinalVerdict: "",
	}

	// If the step contains a sub-plan, treat it as a delegation: execute the sub-plan
	// and use its results as the step's output. No need to call the step runner.
	if step.Plan != nil {
		subReport, err := e.Execute(ctx, step.Plan, objective)
		if err != nil {
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictFail
			return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("sub-plan for step %s failed: %w", step.ID, err)
		}
		stepExec.SubPlan = subReport

		// Use the sub-plan execution report as the step's output
		// Format it as a summary string for judging
		// For dependency outputs, we'll use a more concise format
		subOutput := fmt.Sprintf("Sub-plan executed successfully. Duration: %s. Steps completed: %d/%d",
			subReport.Duration(), len(subReport.Steps), len(subReport.Steps))
		// Store the detailed JSON for when this step's output is used as a dependency
		subOutputDetailed := subOutput
		if b, err := json.MarshalIndent(subReport, "", "  "); err == nil {
			subOutputDetailed = string(b)
		}
		subOutput = subOutputDetailed

		// Judge the sub-plan execution result
		judgeRes := e.Judge.Execute(ctx, tool.Arguments{
			"objective":   objective,
			"step_name":   step.Name,
			"instruction": step.Instruction,
			"expectation": step.Expectation,
			"output":      subOutput,
			"context":     fmt.Sprintf("This step executed a nested sub-plan (id=%s)", step.Plan.ID),
		})
		if judgeRes.Errored() {
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictFail
			return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("judge errored for step %s: %w", step.ID, judgeRes.GetError())
		}
		jr, ok := judgeRes.GetResult().(judging.JudgeResult)
		if !ok {
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictFail
			return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("judge returned unexpected type %T", judgeRes.GetResult())
		}

		// Record the attempt
		at := StepAttempt{
			Attempt:     1,
			StartedAt:   stepExec.StartedAt,
			EndedAt:     time.Now(),
			Output:      subOutput,
			JudgeResult: jr,
		}
		stepExec.Attempts = append(stepExec.Attempts, at)
		stepExec.EndedAt = time.Now()
		stepExec.FinalVerdict = jr.Verdict

		// Handle replan verdict for sub-plan steps
		if jr.Verdict == judging.VerdictReplan {
			if e.Planner == nil {
				return stepExec, judging.VerdictReplan, nil, nil, nil
			}

			// Check if we should escalate to parent plan
			// Only check if we have a parent context (currentPlan != step.Plan)
			shouldEscalate := false
			escalationReason := ""

			if currentPlan != nil && currentPlan.ID != step.Plan.ID {
				feedback := fmt.Sprintf("Judge requested replan at step %s (%s) after sub-plan execution: %s\nSuggestedFix: %s",
					step.ID, step.Name, jr.Justification, jr.SuggestedFix)

				var err error
				shouldEscalate, escalationReason, err = e.shouldEscalateReplan(
					ctx,
					step,        // parent step (pointer)
					currentPlan, // parent plan
					step.Plan,   // sub-plan
					feedback,    // replan reason
					objective,
				)

				if err != nil {
					// If escalation check fails, default to local replan
					shouldEscalate = false
				}
			}

			if shouldEscalate {
				// Escalate: return replan verdict to parent executor
				// The parent will handle replanning at its level
				escalationFeedback := fmt.Sprintf(
					"Sub-plan execution escalated replan request to parent level.\n"+
						"Sub-plan: %s\n"+
						"Escalation reason: %s\n"+
						"Original replan reason: %s",
					step.Plan.ID,
					escalationReason,
					jr.Justification,
				)
				return stepExec, judging.VerdictReplan, nil, nil, fmt.Errorf("escalated: %s", escalationFeedback)
			}

			// Local replan: proceed with sub-plan replanning
			feedback := fmt.Sprintf("Judge requested replan at step %s (%s) after sub-plan execution: %s\nSuggestedFix: %s",
				step.ID, step.Name, jr.Justification, jr.SuggestedFix)
			res, err := e.Planner.Replan(ctx, step.Plan, feedback, planning.PlannerInput{
				Objective: objective,
				Context:   feedback,
			})
			if err != nil {
				stepExec.FinalVerdict = judging.VerdictFail
				return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("replan failed: %w", err)
			}

			// Update the step's sub-plan in place
			step.Plan = res.Plan
			return stepExec, judging.VerdictReplan, res.Plan.RevisionDiff, nil, nil
		}

		return stepExec, jr.Verdict, nil, nil, nil
	}

	// No sub-plan: execute the step using the step runner
	var lastJudge judging.JudgeResult
	var lastOutput string
	for attempt := 1; attempt <= cfg.MaxAttemptsPerStep; attempt++ {
		at := StepAttempt{
			Attempt:   attempt,
			StartedAt: time.Now(),
		}

		runArgs := tool.Arguments{
			"objective": objective,
			"plan":      currentPlan,
			"step":      step,
			"attempt":   attempt,
			"prior_feedback": func() string {
				if attempt == 1 {
					return ""
				}
				return lastJudge.SuggestedFix
			}(),
			"dependency_outputs": dependencyOutputs,
		}

		runRes := e.StepRunner.Execute(ctx, runArgs)
		if runRes.Errored() {
			lastOutput = fmt.Sprintf("step runner error: %v", runRes.GetError())
		} else {
			s, err := runRes.String()
			if err != nil {
				lastOutput = fmt.Sprintf("step runner stringify error: %v", err)
			} else {
				// String() returns JSON for strings; try to unwrap for readability.
				var maybeString string
				if err := json.Unmarshal([]byte(s), &maybeString); err == nil {
					lastOutput = maybeString
				} else {
					lastOutput = s
				}
			}
		}

		judgeRes := e.Judge.Execute(ctx, tool.Arguments{
			"objective":   objective,
			"step_name":   step.Name,
			"instruction": step.Instruction,
			"expectation": step.Expectation,
			"output":      lastOutput,
		})
		if judgeRes.Errored() {
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictFail
			return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("judge errored for step %s: %w", step.ID, judgeRes.GetError())
		}
		jr, ok := judgeRes.GetResult().(judging.JudgeResult)
		if !ok {
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictFail
			return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("judge returned unexpected type %T", judgeRes.GetResult())
		}
		lastJudge = jr

		at.Output = lastOutput
		at.EndedAt = time.Now()
		at.JudgeResult = jr
		stepExec.Attempts = append(stepExec.Attempts, at)

		switch jr.Verdict {
		case judging.VerdictPass:
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictPass
			return stepExec, judging.VerdictPass, nil, nil, nil
		case judging.VerdictFail:
			// retry until max attempts
			continue
		case judging.VerdictReplan:
			if e.Planner == nil {
				stepExec.EndedAt = time.Now()
				stepExec.FinalVerdict = judging.VerdictReplan
				return stepExec, judging.VerdictReplan, nil, nil, nil
			}
			// Ask planner for a revised plan.
			feedback := fmt.Sprintf("Judge requested replan at step %s (%s): %s\nSuggestedFix: %s\nLastOutput: %s",
				step.ID, step.Name, jr.Justification, jr.SuggestedFix, lastOutput)

			res, err := e.Planner.Replan(ctx, currentPlan, feedback, planning.PlannerInput{
				Objective: objective,
				Context:   feedback,
			})
			if err != nil {
				stepExec.EndedAt = time.Now()
				stepExec.FinalVerdict = judging.VerdictFail
				return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("replan failed: %w", err)
			}
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictReplan
			return stepExec, judging.VerdictReplan, res.Plan.RevisionDiff, res.Plan, nil
		default:
			stepExec.EndedAt = time.Now()
			stepExec.FinalVerdict = judging.VerdictFail
			return stepExec, judging.VerdictFail, nil, nil, fmt.Errorf("unknown judge verdict %q", jr.Verdict)
		}
	}

	stepExec.EndedAt = time.Now()
	stepExec.FinalVerdict = judging.VerdictFail
	return stepExec, judging.VerdictFail, nil, nil, nil
}
