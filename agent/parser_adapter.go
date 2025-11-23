package agent

import "github.com/hlfshell/gogentic/parser"

// parserAdapter adapts a parser.Parser to the ParserInterface.
type parserAdapter struct {
	parser parser.Parser[map[string]interface{}]
}

// Parse implements ParserInterface.
func (p *parserAdapter) Parse(input string) (map[string]interface{}, []string) {
	result, err := p.parser.Parse(input)
	if err != nil {
		if parseErr, ok := err.(*parser.ParseErrors); ok {
			return result, parseErr.AllErrors()
		}
		return result, []string{err.Error()}
	}
	return result, nil
}

// NewParserAdapter creates a ParserInterface from a parser.Parser[map[string]interface{}].
func NewParserAdapter(p parser.Parser[map[string]interface{}]) ResponseParserInterface {
	return &parserAdapter{parser: p}
}
