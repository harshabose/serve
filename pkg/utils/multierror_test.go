package utils

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestMultiError_Add(t *testing.T) {
	multiErr := NewMultiError()

	// Add nil error (should be ignored)
	multiErr.Add(nil)
	if multiErr.Len() != 0 {
		t.Errorf("Expected length 0 after adding nil, got %d", multiErr.Len())
	}

	// Add an error
	err1 := errors.New("error 1")
	multiErr.Add(err1)
	if multiErr.Len() != 1 {
		t.Errorf("Expected length 1 after adding 1 error, got %d", multiErr.Len())
	}

	// Add another error
	err2 := errors.New("error 2")
	multiErr.Add(err2)
	if multiErr.Len() != 2 {
		t.Errorf("Expected length 2 after adding 2 errors, got %d", multiErr.Len())
	}
}

func TestMultiError_AddAll(t *testing.T) {
	multiErr := NewMultiError()

	// Add multiple errors at once
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	multiErr.AddAll(err1, nil, err2)

	if multiErr.Len() != 2 {
		t.Errorf("Expected length 2 after adding 2 non-nil errors, got %d", multiErr.Len())
	}
}

func TestMultiError_Error(t *testing.T) {
	t.Run("empty error", func(t *testing.T) {
		multiErr := NewMultiError()
		if multiErr.Error() != "" {
			t.Errorf("Expected empty string for empty MultiError, got %q", multiErr.Error())
		}
	})

	t.Run("single error", func(t *testing.T) {
		multiErr := NewMultiError()
		err := errors.New("test error")
		multiErr.Add(err)

		if multiErr.Error() != "test error" {
			t.Errorf("Expected %q, got %q", err.Error(), multiErr.Error())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		multiErr := NewMultiError()
		multiErr.Add(errors.New("error 1"))
		multiErr.Add(errors.New("error 2"))

		result := multiErr.Error()

		if !strings.Contains(result, "2 errors occurred") {
			t.Errorf("Expected error message to contain count, got: %s", result)
		}

		if !strings.Contains(result, "error 1") || !strings.Contains(result, "error 2") {
			t.Errorf("Expected error message to contain all error messages, got: %s", result)
		}
	})
}

func TestMultiError_ErrorOrNil(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		multiErr := NewMultiError()
		if multiErr.ErrorOrNil() != nil {
			t.Error("Expected nil for empty MultiError")
		}
	})

	t.Run("non-empty returns error", func(t *testing.T) {
		multiErr := NewMultiError()
		multiErr.Add(errors.New("test error"))

		if multiErr.ErrorOrNil() == nil {
			t.Error("Expected non-nil for non-empty MultiError")
		}
	})
}

func TestMultiError_Flatten(t *testing.T) {
	t.Run("flatten nested errors", func(t *testing.T) {
		// Create a hierarchy of errors
		inner := NewMultiError()
		inner.Add(errors.New("inner error 1"))
		inner.Add(errors.New("inner error 2"))

		outer := NewMultiError()
		outer.Add(errors.New("outer error"))
		outer.Add(inner)

		flattened := outer.Flatten()

		if flattened.Len() != 3 {
			t.Errorf("Expected 3 flattened errors, got %d", flattened.Len())
		}

		errorText := flattened.Error()
		if !strings.Contains(errorText, "inner error 1") ||
			!strings.Contains(errorText, "inner error 2") ||
			!strings.Contains(errorText, "outer error") {
			t.Errorf("Flattened error missing expected content: %s", errorText)
		}
	})

	t.Run("deeply nested errors", func(t *testing.T) {
		level3 := NewMultiError()
		level3.Add(errors.New("level 3 error"))

		level2 := NewMultiError()
		level2.Add(level3)
		level2.Add(errors.New("level 2 error"))

		level1 := NewMultiError()
		level1.Add(level2)
		level1.Add(errors.New("level 1 error"))

		flattened := level1.Flatten()

		if flattened.Len() != 3 {
			t.Errorf("Expected 3 flattened errors, got %d", flattened.Len())
		}
	})
}

func TestMultiError_ErrorInterface(t *testing.T) {
	// Test that MultiError satisfies the error interface
	var err error
	multiErr := NewMultiError()
	multiErr.Add(errors.New("test error"))

	// Should compile if MultiError implements error
	err = multiErr

	if err.Error() != "test error" {
		t.Errorf("Expected %q, got %q", "test error", err.Error())
	}
}

func ExampleMultiError() {
	// Create a new MultiError
	multiErr := NewMultiError()

	// Add some errors
	multiErr.Add(fmt.Errorf("failed to connect to database"))
	multiErr.Add(fmt.Errorf("failed to authenticate user"))

	// Check if there are any errors
	if err := multiErr.ErrorOrNil(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// 2 errors occurred:
	//   * failed to connect to database
	//
	//   * failed to authenticate user
}
