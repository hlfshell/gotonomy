package parser

import (
	"encoding/json"
	"fmt"

	"github.com/hlfshell/structured-parse/go/structuredparse"
	"gopkg.in/yaml.v3"
)

type Parser[T any] interface {
	Parse(input string) (T, error)
}

// #########################################################
// Implementations
// #########################################################

// ##### StructuredLabelsParser #####

// StructuredLabelsParser wraps the structured-parse library parser for single record parsing.
// It implements the Parser[T] interface for any type T.
type StructuredLabelsParser[T any] struct {
	parser      *structuredparse.Parser
	convertFunc func(map[string]interface{}) (T, error)
}

// NewStructuredParse creates a new StructuredParse parser with the given labels.
// Labels define the expected fields in the LLM output.
// The convertFunc is required and converts map[string]interface{} to type T.
func NewStructuredParse[T any](
	labels []structuredparse.Label,
	convertFunc func(map[string]interface{}) (T, error),
) (*StructuredLabelsParser[T], error) {
	parser, err := structuredparse.NewParser(labels, nil)
	if err != nil {
		return nil, err
	}

	return &StructuredLabelsParser[T]{
		parser:      parser,
		convertFunc: convertFunc,
	}, nil
}

// Parse parses the input text and returns the parsed data converted to type T.
// It returns the parsed data and any errors encountered during parsing.
// Errors are non-fatal and parsing continues even if some fields fail.
func (sp *StructuredLabelsParser[T]) Parse(input string) (T, error) {
	var result T
	parsedMap, parseErrors := sp.parser.Parse(input)

	converted, err := sp.convertFunc(parsedMap)
	if err != nil {
		return result, fmt.Errorf("failed to convert parsed data: %w", err)
	}

	if len(parseErrors) > 0 {
		return converted, &ParseErrors{Errors: parseErrors, Result: parsedMap}
	}

	return converted, nil
}

// ##### StructuredLabelBlocksParser #####

// SliceOf expresses that a type is a slice of element type E.
// This allows expressing relationships like "T is []E" in generic constraints.
type SliceOf[E any] interface {
	~[]E
}

// StructuredLabelBlocksParser wraps the structured-parse library parser for block parsing.
// It parses multiple blocks and returns a slice of type E.
// It implements the Parser[T] interface where T is constrained to be []E.
// T is the slice type (e.g., []Task), and E is the element type (e.g., Task).
type StructuredLabelBlocksParser[T SliceOf[E], E any] struct {
	parser      *structuredparse.Parser
	convertFunc func(map[string]interface{}) (E, error)
}

// NewStructuredParseBlocks creates a new StructuredParseBlocks parser with the given labels.
// Labels define the expected fields in the LLM output, with at least one label having IsBlockStart: true.
// The convertFunc is required and converts each block (map[string]interface{}) to the element type E.
// T must be a slice type of E (e.g., if E is Task, then T must be []Task).
func NewStructuredParseBlocks[T SliceOf[E], E any](
	labels []structuredparse.Label,
	convertFunc func(map[string]interface{}) (E, error),
) (*StructuredLabelBlocksParser[T, E], error) {
	parser, err := structuredparse.NewParser(labels, nil)
	if err != nil {
		return nil, err
	}

	return &StructuredLabelBlocksParser[T, E]{
		parser:      parser,
		convertFunc: convertFunc,
	}, nil
}

// Parse parses the input text into multiple blocks and returns a slice of type E.
// It returns the parsed data and any errors encountered during parsing.
// Errors are non-fatal and parsing continues even if some blocks fail.
func (sp *StructuredLabelBlocksParser[T, E]) Parse(input string) (T, error) {
	var result T
	blocks, parseErrors := sp.parser.ParseBlocks(input)

	var conversionErrors []string
	results := make([]E, 0, len(blocks))

	for _, block := range blocks {
		converted, err := sp.convertFunc(block)
		if err != nil {
			conversionErrors = append(conversionErrors, fmt.Sprintf("failed to convert block: %v", err))
			continue
		}
		results = append(results, converted)
	}

	// Convert []E to T (where T is constrained to be ~[]E)
	// Use JSON marshaling/unmarshaling to convert between []E and T
	// This works because T's underlying type is []E
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return result, fmt.Errorf("failed to marshal results: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal to target type: %w", err)
	}

	// Merge errors
	allErrors := append(parseErrors, conversionErrors...)
	if len(allErrors) > 0 {
		return result, &ParseErrors{Errors: allErrors, Result: nil}
	}

	return result, nil
}

// ParseErrors represents non-fatal parsing errors from structured-parse.
// The Result field contains the successfully parsed data even if errors occurred.
type ParseErrors struct {
	Errors []string
	Result map[string]interface{}
}

func (e *ParseErrors) Error() string {
	if len(e.Errors) == 0 {
		return "parse errors"
	}
	return e.Errors[0]
}

// AllErrors returns all parsing errors.
func (e *ParseErrors) AllErrors() []string {
	return e.Errors
}

// ##### JSON Parser #####

// JSONParser parses JSON strings into map[string]interface{}.
type JSONParser struct{}

// NewJSONParser creates a new JSON parser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse parses a JSON string into a map[string]interface{}.
func (jp *JSONParser) Parse(input string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ##### YAML Parser #####

// YAMLParser parses YAML strings into map[string]interface{}.
type YAMLParser struct{}

// NewYAMLParser creates a new YAML parser.
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Parse parses a YAML string into a map[string]interface{}.
func (yp *YAMLParser) Parse(input string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := yaml.Unmarshal([]byte(input), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
