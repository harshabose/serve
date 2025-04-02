package utils

import (
	"errors"
	"fmt"
	"strings"
)

// MultiError is a collection of errors that implements the error interface
type MultiError struct {
	errors []error
}

// NewMultiError creates a new empty MultiError
func NewMultiError() *MultiError {
	return &MultiError{errors: []error{}}
}

// Add appends an error to the collection if it's not nil
func (multiErr *MultiError) Add(err error) {
	if err != nil {
		multiErr.errors = append(multiErr.errors, err)
	}
}

// AddAll appends multiple errors to the collection, ignoring nil errors
func (multiErr *MultiError) AddAll(errs ...error) {
	for _, err := range errs {
		multiErr.Add(err)
	}
}

// Len returns the number of errors in the collection
func (multiErr *MultiError) Len() int {
	return len(multiErr.errors)
}

// Error implements the error interface
func (multiErr *MultiError) Error() string {
	if multiErr.Len() == 0 {
		return ""
	}

	if multiErr.Len() == 1 {
		return multiErr.errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d errors occurred:\n", multiErr.Len()))

	for i, err := range multiErr.errors {
		sb.WriteString(fmt.Sprintf("  * %s\n", err.Error()))
		if i < multiErr.Len()-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ErrorOrNil returns nil if the collection is empty, otherwise returns the flattened MultiError
func (multiErr *MultiError) ErrorOrNil() error {
	if multiErr.Len() == 0 {
		return nil
	}
	return multiErr.Flatten()
}

// Errors returns all errors in the collection
func (multiErr *MultiError) Errors() []error {
	return multiErr.errors
}

// Flatten returns a new MultiError with all nested MultiErrors flattened
func (multiErr *MultiError) Flatten() *MultiError {
	flattened := NewMultiError()

	for _, err := range multiErr.errors {
		var merr *MultiError
		if errors.As(err, &merr) {
			flattened.AddAll(merr.Flatten().Errors()...)
		} else {
			flattened.Add(err)
		}
	}

	return flattened
}
