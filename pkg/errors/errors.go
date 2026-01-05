package errors

import (
	"fmt"
	"io"
	"runtime"
	"strings"
)

// Error is an enhanced error type with code, context, suggestions, and optional stack trace.
// It provides rich debugging information while remaining compatible with standard Go errors.
type Error struct {
	// Code is the unique error identifier
	Code ErrorCode

	// Message is the human-readable error message
	Message string

	// Cause is the underlying error that caused this error
	Cause error

	// Context contains additional structured data about the error
	Context map[string]any

	// Suggestions are actionable recommendations for resolving the error
	Suggestions []string

	// StackTrace contains the call stack (only populated in debug mode)
	StackTrace []string
}

// New creates a new enhanced error with the given code and message.
// Additional context and suggestions can be added using the With* methods.
//
// Example:
//
//	err := errors.New(errors.ErrConnectionFailed, "could not connect to database")
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Context: make(map[string]any),
	}
}

// Wrap wraps an existing error with an error code and message, preserving the error chain.
// This is the most common way to enhance errors throughout the application.
//
// Example:
//
//	if err != nil {
//	    return errors.Wrap(err, errors.ErrHashingFailed, "failed to hash table rows")
//	}
func Wrap(err error, code ErrorCode, message string) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
		Context: make(map[string]any),
	}
}

// With adds a key-value pair to the error's context.
// This allows attaching structured data that aids debugging.
//
// Example:
//
//	err.With("table", "users").With("row_count", 1000)
func (e *Error) With(key string, value any) *Error {
	if e == nil {
		return nil
	}
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds an actionable suggestion to the error.
// Suggestions help users resolve the error without consulting documentation.
//
// Example:
//
//	err.WithSuggestion("Verify database host and port are correct")
func (e *Error) WithSuggestion(suggestion string) *Error {
	if e == nil {
		return nil
	}
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithStackTrace captures the current call stack and adds it to the error.
// This is expensive and should only be used in debug mode.
//
// Example:
//
//	if debugMode {
//	    err.WithStackTrace(3) // Skip 3 frames
//	}
func (e *Error) WithStackTrace(skip int) *Error {
	if e == nil {
		return nil
	}
	e.StackTrace = captureStackTrace(skip + 1)
	return e
}

// Error implements the error interface, returning a formatted error message.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", e.Code, e.Message)

	if e.Cause != nil {
		fmt.Fprintf(&b, ": %v", e.Cause)
	}

	return b.String()
}

// Unwrap returns the underlying cause error, allowing error chain traversal.
// This enables use with errors.Is() and errors.As().
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// DebugString returns a detailed multi-line string representation with all error information.
// This includes the code, message, context, suggestions, and stack trace if available.
//
// Example output:
//
//	Error: [DDDB1001] connection to database failed
//	Category: Connection
//	Caused by: dial tcp: connection refused
//	Context:
//	  host = localhost
//	  port = 3306
//	Suggestions:
//	  - Verify database host and port are correct
//	  - Check network connectivity to database server
//	Stack trace:
//	  github.com/iamvirul/deepdiff-db/internal/drivers.Open
//	    /path/to/drivers.go:42
func (e *Error) DebugString() string {
	if e == nil {
		return ""
	}

	var b strings.Builder

	fmt.Fprintf(&b, "Error: [%s] %s\n", e.Code, e.Message)
	fmt.Fprintf(&b, "Category: %s\n", e.Code.Category())

	if e.Cause != nil {
		fmt.Fprintf(&b, "Caused by: %v\n", e.Cause)
	}

	if len(e.Context) > 0 {
		fmt.Fprintf(&b, "Context:\n")
		for k, v := range e.Context {
			fmt.Fprintf(&b, "  %s = %v\n", k, v)
		}
	}

	if len(e.Suggestions) > 0 {
		fmt.Fprintf(&b, "Suggestions:\n")
		for _, s := range e.Suggestions {
			fmt.Fprintf(&b, "  - %s\n", s)
		}
	}

	if len(e.StackTrace) > 0 {
		fmt.Fprintf(&b, "Stack trace:\n")
		for _, frame := range e.StackTrace {
			fmt.Fprintf(&b, "  %s\n", frame)
		}
	}

	return b.String()
}

// Format implements fmt.Formatter for custom formatting.
// Use %v for standard output, %+v for detailed output.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// Detailed format
			io.WriteString(s, e.DebugString())
		} else {
			// Standard format
			io.WriteString(s, e.Error())
		}
	case 's':
		io.WriteString(s, e.Error())
	}
}

// captureStackTrace captures the current call stack, skipping the specified number of frames.
func captureStackTrace(skip int) []string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr

	// Skip captureStackTrace itself plus the requested skip
	n := runtime.Callers(skip+1, pcs[:])

	frames := runtime.CallersFrames(pcs[:n])
	var trace []string

	for {
		frame, more := frames.Next()

		// Skip runtime frames
		if !strings.Contains(frame.Function, "runtime.") {
			trace = append(trace, fmt.Sprintf("%s\n    %s:%d",
				frame.Function,
				frame.File,
				frame.Line))
		}

		if !more {
			break
		}
	}

	return trace
}

// IsRetryable returns true if the error code indicates the operation might succeed on retry.
func (e *Error) IsRetryable() bool {
	if e == nil {
		return false
	}
	return e.Code.IsRetryable()
}
