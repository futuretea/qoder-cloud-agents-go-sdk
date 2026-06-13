package qoderhttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// APIError represents a parsed Qoder API error response.
type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"-"`
	ErrorType  string `json:"type"`
	Message    string `json:"message"`
	Param      string `json:"param,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := fmt.Sprintf("qoder API error %d (%s): %s", e.StatusCode, e.ErrorType, e.Message)
	if e.Param != "" {
		msg += fmt.Sprintf(" (param: %s)", e.Param)
	}
	return msg
}

// IsNotFound returns true for 404 errors.
func (e *APIError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

// IsConflict returns true for 409 conflict errors (e.g., version mismatch).
func (e *APIError) IsConflict() bool { return e.StatusCode == http.StatusConflict }

// IsUnauthorized returns true for 401 errors.
func (e *APIError) IsUnauthorized() bool { return e.StatusCode == http.StatusUnauthorized }

// IsPermissionError returns true for 403 errors.
func (e *APIError) IsPermissionError() bool { return e.StatusCode == http.StatusForbidden }

// IsInvalidRequest returns true for invalid_request_error type.
func (e *APIError) IsInvalidRequest() bool { return e.ErrorType == "invalid_request_error" }

// IsServerError returns true for 5xx status codes.
func (e *APIError) IsServerError() bool { return e.StatusCode >= 500 }

// IsAPIError checks if an error is or wraps a *APIError and returns it.
// It uses errors.As to unwrap the error chain, which handles go-http-client
// response middleware wrapping: "response middleware error: <actual error>".
func IsAPIError(err error) (*APIError, bool) {
	if err == nil {
		return nil, false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// apiErrorEnvelope matches the Qoder error response structure.
type apiErrorEnvelope struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

// QoderErrorMiddleware is a go-http-client ResponseMiddleware that parses
// Qoder API error envelopes on non-2xx responses and returns typed *APIError.
func QoderErrorMiddleware(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return &APIError{
			StatusCode: resp.StatusCode,
			ErrorType:  "unknown_error",
			Message:    fmt.Sprintf("failed to read error body: %v", err),
		}
	}
	// Restore body for downstream use
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var env apiErrorEnvelope
	if err := json.Unmarshal(body, &env); err != nil || env.Error.Message == "" {
		msg := string(body)
		if len(msg) > 1024 {
			msg = msg[:1024] + "..."
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			ErrorType:  "unknown_error",
			Message:    msg,
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Type:       env.Type,
		ErrorType:  env.Error.Type,
		Message:    env.Error.Message,
		Param:      env.Error.Param,
	}
}

// Ensure APIError implements the error interface.
var _ error = (*APIError)(nil)
