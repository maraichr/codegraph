package apierr

import "fmt"

// Error is a structured API error with a machine-readable code, human-readable
// message, HTTP status, and an optional wrapped cause (never serialized).
type Error struct {
	code    Code
	message string
	status  int
	cause   error
}

// New creates an Error without a cause.
func New(code Code, status int, message string) *Error {
	return &Error{code: code, message: message, status: status}
}

// Wrap creates an Error that wraps a cause for logging/unwrapping.
func Wrap(code Code, status int, message string, cause error) *Error {
	return &Error{code: code, message: message, status: status, cause: cause}
}

// Error implements the error interface. Includes the cause for log output.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// Unwrap returns the wrapped cause for errors.Is/errors.As chaining.
func (e *Error) Unwrap() error { return e.cause }

// Code returns the machine-readable error code.
func (e *Error) Code() Code { return e.code }

// Message returns the human-readable message.
func (e *Error) Message() string { return e.message }

// Status returns the HTTP status code.
func (e *Error) Status() int { return e.status }

// ErrorResponse is the wire format written as JSON to the client.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the inner object of ErrorResponse.
type ErrorBody struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

// Response returns the wire-format representation of this error.
func (e *Error) Response() ErrorResponse {
	return ErrorResponse{
		Error: ErrorBody{
			Code:    e.code,
			Message: e.message,
		},
	}
}
