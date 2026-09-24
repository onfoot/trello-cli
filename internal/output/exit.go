package output

import (
	"errors"
	"fmt"
)

// Exit codes are the documented contract (SPEC §3.5). Every command error
// maps to exactly one of these via CodeFor.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitUsage      = 2
	ExitConfig     = 3
	ExitAuth       = 4
	ExitNotFound   = 5
	ExitValidation = 6
	ExitRateLimit  = 7
	ExitNetwork    = 8
)

// ExitCoder is implemented by errors that carry a specific exit code.
type ExitCoder interface {
	ExitCode() int
}

// CodedError attaches an exit code to an arbitrary error.
type CodedError struct {
	Code int
	Err  error
}

func (e *CodedError) Error() string { return e.Err.Error() }
func (e *CodedError) Unwrap() error { return e.Err }
func (e *CodedError) ExitCode() int { return e.Code }

// WithCode returns err annotated with code, or nil when err is nil.
func WithCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &CodedError{Code: code, Err: err}
}

// CodeFor maps an error to its exit code: the code carried by the error when
// it (or anything it wraps) implements ExitCoder, otherwise ExitError.
func CodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var ec ExitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return ExitError
}

// Errorf builds a plain error carrying code.
func Errorf(code int, format string, args ...any) error {
	return WithCode(fmt.Errorf(format, args...), code)
}
