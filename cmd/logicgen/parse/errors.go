package parse

import (
	"fmt"
	"strings"
)

// ValidationErrors contains multiple validation errors
type ValidationErrors struct {
	Errors []error
}

// Error implements the error interface
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}

	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation errors:\n", len(e.Errors)))
	for _, err := range e.Errors {
		sb.WriteString("  - ")
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}
	return sb.String()
}

// Unwrap returns the first error (for errors.Is/As compatibility)
func (e *ValidationErrors) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// HasErrors returns true if there are any errors
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// ParseError represents an error during parsing
type ParseError struct {
	Source  string
	Line    int
	Column  int
	Message string
	Cause   error
}

// Error implements the error interface
func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.Source, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Source, e.Message)
}

// Unwrap returns the underlying cause
func (e *ParseError) Unwrap() error {
	return e.Cause
}

// RuleError represents an error in a specific rule
type RuleError struct {
	RuleName string
	Field    string
	Message  string
	Cause    error
}

// Error implements the error interface
func (e *RuleError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("rule '%s'.%s: %s", e.RuleName, e.Field, e.Message)
	}
	return fmt.Sprintf("rule '%s': %s", e.RuleName, e.Message)
}

// Unwrap returns the underlying cause
func (e *RuleError) Unwrap() error {
	return e.Cause
}

// PathError represents an error in a path expression
type PathError struct {
	Path    string
	Message string
}

// Error implements the error interface
func (e *PathError) Error() string {
	return fmt.Sprintf("invalid path '%s': %s", e.Path, e.Message)
}

// TransformError represents an error in a transform
type TransformError struct {
	Type    string
	Message string
	Cause   error
}

// Error implements the error interface
func (e *TransformError) Error() string {
	return fmt.Sprintf("transform %s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *TransformError) Unwrap() error {
	return e.Cause
}

// NewParseError creates a new parse error
func NewParseError(source, message string, cause error) *ParseError {
	return &ParseError{
		Source:  source,
		Message: message,
		Cause:   cause,
	}
}

// NewRuleError creates a new rule error
func NewRuleError(ruleName, field, message string) *RuleError {
	return &RuleError{
		RuleName: ruleName,
		Field:    field,
		Message:  message,
	}
}

// NewPathError creates a new path error
func NewPathError(path, message string) *PathError {
	return &PathError{
		Path:    path,
		Message: message,
	}
}

// NewTransformError creates a new transform error
func NewTransformError(transformType, message string, cause error) *TransformError {
	return &TransformError{
		Type:    transformType,
		Message: message,
		Cause:   cause,
	}
}

// CombineErrors combines multiple errors into one
func CombineErrors(errors ...error) error {
	var nonNil []error
	for _, err := range errors {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}

	if len(nonNil) == 0 {
		return nil
	}
	if len(nonNil) == 1 {
		return nonNil[0]
	}

	return &ValidationErrors{Errors: nonNil}
}
