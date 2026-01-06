package errors

import (
	"fmt"
)

type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

var (
// ErrInvalidZoom is returned when zoom level is invalid
ErrInvalidZoom = &AppError{
Code:    CodeInvalidInput,
Message: "Invalid zoom level: must be between 0 and 18",
}

// ErrInvalidTransportType is returned when transport type is invalid
ErrInvalidTransportType = &AppError{
Code:    CodeInvalidInput,
Message: "Invalid transport type",
}
)
