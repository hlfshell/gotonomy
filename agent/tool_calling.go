package agent

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/tool"
)

// toolResult represents the result of executing a single tool call
type toolResult struct {
	index   int
	call    model.ToolCall
	result  tool.ResultInterface
	content string
}

// validateToolsCalled ensures all requested tools exist in the
// agent's tool registry (ie no hallucinated tools)
func (a *Agent) validateToolsCalled(calls []model.ToolCall) error {
	for _, call := range calls {
		if _, ok := a.tools[call.Name]; !ok {
			return fmt.Errorf("unknown tool: %s", call.Name)
		}
	}
	return nil
}

// calculateWorkerPoolSize determines how many workers to use based
// on config and number of calls (ie don't create more workers than
// calls)
func (a *Agent) calculateWorkerPoolSize(numCalls int) int {
	maxWorkers := a.config.MaxToolWorkers
	if maxWorkers == 0 || maxWorkers > numCalls {
		return numCalls
	}
	return maxWorkers
}

// handleToolError processes a tool error based on the agent's error handling configuration
func (a *Agent) handleToolError(
	toolName string,
	res tool.ResultInterface,
	firstError *error,
	firstErrorMutex *sync.Mutex,
) (tool.ResultInterface, error) {
	// If the result is not an error, there is nothing to handle.
	if !res.Errored() {
		return res, nil
	}

	var finalResult tool.ResultInterface = res
	var contentErr error

	switch a.config.ToolErrorHandling {
	case StopOnFirstToolError:
		// Record the first error; we'll return it after all tools finish.
		firstErrorMutex.Lock()
		if *firstError == nil {
			*firstError = fmt.Errorf("tool %s failed: %v", toolName, res.GetError())
		}
		firstErrorMutex.Unlock()
		finalResult = res

	case FunctionOnError:
		if a.config.OnToolErrorFunction != nil {
			handledResult, handlerErr := a.config.OnToolErrorFunction(res)
			if handlerErr != nil {
				finalResult = res
				contentErr = handlerErr
			} else {
				finalResult = handledResult
			}
		} else {
			finalResult = res
		}

	case PassErrorsToModel:
		finalResult = res
	}

	return finalResult, contentErr
}

// processToolResult converts a tool result into a string content for the model
func processToolResult(
	toolName string,
	finalResult tool.ResultInterface,
	contentErr error,
	originalError error,
) string {
	content, err := finalResult.String()
	if err != nil {
		if contentErr != nil {
			return fmt.Sprintf("tool %s error: %v (handler error: %v)", toolName, originalError, contentErr)
		}
		return fmt.Sprintf("tool %s error: %v", toolName, err)
	}
	if contentErr != nil {
		return fmt.Sprintf("tool %s error: %v", toolName, contentErr)
	}
	return content
}

// appendToolMessagesToSession adds all tool results as messages to
// the session.
func appendToolMessagesToSession(sess *Session, results []toolResult) {
	for _, result := range results {
		content := fmt.Sprintf("Tool %s returned: %s", result.call.Name, result.content)
		if result.call.ID != "" {
			content = fmt.Sprintf("ToolCall %s (%s) returned: %s", result.call.ID, result.call.Name, result.content)
		}
		systemMsg := model.Message{
			Role:       model.RoleSystem,
			Content:    content,
			ToolCallID: result.call.ID,
		}
		sess.AppendToolMessage(systemMsg)
	}
}

func ensureToolCallIDs(calls []model.ToolCall) {
	for i := range calls {
		if calls[i].ID == "" {
			calls[i].ID = uuid.NewString()
		}
	}
}

// handleToolCalls executes tool calls requested by the model using the
// provided parent context. Tool results are optionally appended to the
// Session as tool messages so they can be fed back into subsequent LLM
// calls. Supports parallel execution based on MaxToolWorkers and error
// handling based on ToolErrorHandling set within the AgentConfig.
func (a *Agent) handleToolCalls(
	parentCtx *tool.Context,
	session *Session,
	step *Step,
) error {
	calls := step.GetResponse().ToolCalls
	if len(calls) == 0 {
		return nil
	}

	// Ensure we can correlate a tool call with its tool result message later.
	// Provider IDs are preferred; otherwise we generate stable IDs for this run.
	ensureToolCallIDs(calls)

	// Validate all tools exist before starting execution
	if err := a.validateToolsCalled(calls); err != nil {
		return err
	}

	maxWorkers := a.calculateWorkerPoolSize(len(calls))

	// Concurrency limiter (semaphore)
	sem := make(chan struct{}, maxWorkers)

	var wg sync.WaitGroup
	var firstError error
	var firstErrorMutex sync.Mutex

	// Preallocate result slice to preserve order
	results := make([]toolResult, len(calls))

	for i, call := range calls {
		wg.Add(1)
		sem <- struct{}{} // acquire a slot

		go func(idx int, call model.ToolCall) {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			toolName := call.Name
			t := a.tools[toolName]
			args := call.Arguments

			// Execute the tool
			res := t.Execute(parentCtx, args)

			var finalResult tool.ResultInterface = res
			var contentErr error
			var originalError error

			if res.Errored() {
				originalError = res.GetError()
				finalResult, contentErr = a.handleToolError(
					toolName,
					res,
					&firstError,
					&firstErrorMutex,
				)
			}

			// Convert result to string content
			content := processToolResult(
				toolName,
				finalResult,
				contentErr,
				originalError,
			)

			// Store result in the correct position
			results[idx] = toolResult{
				index:   idx,
				call:    call,
				result:  finalResult,
				content: content,
			}
		}(i, call)
	}

	wg.Wait()

	// If StopOnFirstToolError and we encountered an error, return it
	if a.config.ToolErrorHandling == StopOnFirstToolError && firstError != nil {
		return firstError
	}

	// Append tool messages to session in order
	appendToolMessagesToSession(session, results)

	return nil
}
