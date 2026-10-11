// Package apperr defines the API's client-facing errors. Business code returns these;
// the error middleware turns them into the standard error envelope. Any other error is
// logged and reported to the client as INTERNAL_ERROR, so internal details never leak.
package apperr

import (
	"errors"
	"net/http"
)

// Code is a stable, machine-readable error code. Clients (web, iOS, Android) branch on it.
// Codes are part of the API contract: add new ones, never rename or repurpose existing ones.
type Code string

const (
	CodeBadRequest       Code = "BAD_REQUEST"
	CodeValidation       Code = "VALIDATION_ERROR"
	CodeUnauthorized     Code = "UNAUTHORIZED"
	CodeForbidden        Code = "FORBIDDEN"
	CodeNotFound         Code = "NOT_FOUND"
	CodeMethodNotAllowed Code = "METHOD_NOT_ALLOWED"
	CodeConflict         Code = "CONFLICT"
	CodePayloadTooLarge  Code = "PAYLOAD_TOO_LARGE"
	CodeRateLimited      Code = "RATE_LIMITED"
	CodeInternal         Code = "INTERNAL_ERROR"
	CodeUnavailable      Code = "SERVICE_UNAVAILABLE"
)

var statusByCode = map[Code]int{
	CodeBadRequest:       http.StatusBadRequest,
	CodeValidation:       http.StatusUnprocessableEntity,
	CodeUnauthorized:     http.StatusUnauthorized,
	CodeForbidden:        http.StatusForbidden,
	CodeNotFound:         http.StatusNotFound,
	CodeMethodNotAllowed: http.StatusMethodNotAllowed,
	CodeConflict:         http.StatusConflict,
	CodePayloadTooLarge:  http.StatusRequestEntityTooLarge,
	CodeRateLimited:      http.StatusTooManyRequests,
	CodeInternal:         http.StatusInternalServerError,
	CodeUnavailable:      http.StatusServiceUnavailable,
}

// Error is an error that is safe to show to API clients.
type Error struct {
	Code    Code
	Message string
	Details map[string]any
	// Err is the underlying cause. It is logged, never sent to clients.
	Err error
}

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap attaches an internal cause that is logged but not returned to the client.
func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

func (e *Error) Error() string {
	if e.Err != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.Err.Error()
	}
	return string(e.Code) + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// Status returns the HTTP status for the error code.
func (e *Error) Status() int {
	if s, ok := statusByCode[e.Code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// From returns err as an *Error, or INTERNAL_ERROR (with err as the logged cause) when it
// is not one.
func From(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Wrap(CodeInternal, "Something went wrong. Please try again.", err)
}

func NotFound(message string) *Error     { return New(CodeNotFound, message) }
func BadRequest(message string) *Error   { return New(CodeBadRequest, message) }
func Unauthorized(message string) *Error { return New(CodeUnauthorized, message) }
func Forbidden(message string) *Error    { return New(CodeForbidden, message) }
func Conflict(message string) *Error     { return New(CodeConflict, message) }
