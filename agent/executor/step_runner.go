package executor

import (
	"fmt"
	"strings"

	"github.com/hlfshell/gotonomy/agent"
	"github.com/hlfshell/gotonomy/assets"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/plan"
	"github.com/hlfshell/gotonomy/tool"
)

// NewStepRunnerAgent creates an agent that executes a single plan step instruction.
// It is a normal gotonomy `agent.Agent` that can optionally call tools.
// The agent accepts a plan and step object directly, deriving all information from them.
func NewStepRunnerAgent(m model.Model, tools []tool.Tool) (*agent.Agent, error) {
	tmpl, err := assets.LoadPrompt("step.prompt")
	if err != nil {
		return nil, err
	}

	return agent.NewAgent(
		"step_runner",
		"Executes one plan step instruction and returns the produced output.",
		m,
		agent.WithTools(tools),
		agent.WithParameters([]tool.Parameter{
			tool.NewParameter[string]("objective", "Overall objective of the plan.", true, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[*plan.Plan]("plan", "The plan containing all steps (including sub-plans).", true, nil, func(v *plan.Plan) (string, error) {
				if v == nil {
					return "", fmt.Errorf("plan cannot be nil")
				}
				return v.ID, nil
			}),
			tool.NewParameter[*plan.Step]("step", "The step to execute (may contain a nested sub-plan).", true, nil, func(v *plan.Step) (string, error) {
				if v == nil {
					return "", fmt.Errorf("step cannot be nil")
				}
				return v.ID, nil
			}),
			tool.NewParameter[int]("attempt", "Attempt number.", false, 1, func(v int) (string, error) { return fmt.Sprintf("%d", v), nil }),
			tool.NewParameter[string]("prior_feedback", "Optional feedback from previous attempt/judge.", false, "", func(v string) (string, error) { return v, nil }),
			tool.NewParameter[[]DependencyOutput]("dependency_outputs", "Outputs from steps this step depends on.", false, nil, func(v []DependencyOutput) (string, error) {
				if len(v) == 0 {
					return "", nil
				}
				return fmt.Sprintf("%d dependency outputs", len(v)), nil
			}),
		}),
		agent.WithArgumentsToMessages(func(args tool.Arguments, sess *agent.Session) ([]model.Message, error) {
			if sess != nil && len(sess.Steps()) > 0 {
				return sess.Conversation(), nil
			}

			// Extract plan and step objects
			p, ok := args["plan"].(*plan.Plan)
			if !ok || p == nil {
				return nil, fmt.Errorf("plan argument must be a *plan.Plan")
			}

			step, ok := args["step"].(*plan.Step)
			if !ok || step == nil {
				return nil, fmt.Errorf("step argument must be a *plan.Step")
			}

			// Note: Steps with sub-plans are handled by the executor directly,
			// so the step runner is only called for steps without sub-plans.

			data := map[string]any{
				"objective":        args["objective"],
				"plan_id":          p.ID,
				"step_id":          step.ID,
				"step_name":        step.Name,
				"step_instruction": step.Instruction,
				"step_expectation": step.Expectation,
				"tools":            tools,
			}

			// Build context from dependency outputs and prior feedback
			var contextParts []string

			// Add dependency outputs if present
			if depOutputs, ok := args["dependency_outputs"].([]DependencyOutput); ok && len(depOutputs) > 0 {
				depParts := []string{"## OUTPUTS FROM DEPENDENT STEPS"}
				for _, dep := range depOutputs {
					depParts = append(depParts, fmt.Sprintf("\n### Step: %s (ID: %s)", dep.StepName, dep.StepID))
					depParts = append(depParts, fmt.Sprintf("Instruction: %s", dep.Instruction))
					depParts = append(depParts, fmt.Sprintf("Output:\n%s", dep.Output))
				}
				contextParts = append(contextParts, strings.Join(depParts, "\n"))
			}

			// Add prior feedback if present
			if pf, ok := args["prior_feedback"].(string); ok && pf != "" {
				contextParts = append(contextParts, fmt.Sprintf("## PRIOR FEEDBACK\n%s", pf))
			}

			if len(contextParts) > 0 {
				data["context"] = strings.Join(contextParts, "\n\n")
			}

			rendered, err := tmpl.Render(data)
			if err != nil {
				return nil, err
			}

			return []model.Message{
				{Role: model.RoleSystem, Content: rendered},
			}, nil
		}),
		agent.WithParser(func(output string) (any, error) { return output, nil }),
	), nil
}
