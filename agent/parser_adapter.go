package agent

import "github.com/hlfshell/gotonomy/parser"

// NewParserAdapter adapts a parser.Parser[map[string]interface{}] to the
// agent.ResponseParser function type. It converts structured-parse errors
// into a slice of warning strings while preserving the successfully parsed
// result whenever possible.
func NewParserAdapter(p parser.Parser[map[string]interface{}]) ResponseParser {
	return func(input string) (any, []string) {
		var result map[string]interface{}

		parsed, err := p.Parse(input)
		if err == nil {
			return parsed, nil
		}

		// Best-effort: preserve whatever the underlying parser produced.
		result = parsed

		if parseErr, ok := err.(*parser.ParseErrors); ok {
			return result, parseErr.AllErrors()
		}
		return result, []string{err.Error()}
	}
}
