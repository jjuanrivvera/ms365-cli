package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError is a Microsoft Graph error with an actionable hint keyed by status.
type APIError struct {
	StatusCode int
	Code       string // Graph error code, e.g. "InvalidAuthenticationToken", "ErrorItemNotFound"
	Message    string
	RequestID  string // client-request-id / request-id for support escalation
	Body       []byte
}

// graphErrorBody is Graph's documented error envelope.
type graphErrorBody struct {
	Error struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		InnerError struct {
			RequestID string `json:"request-id"`
		} `json:"innerError"`
	} `json:"error"`
}

func parseAPIError(status int, body []byte, h http.Header) *APIError {
	e := &APIError{StatusCode: status, Body: body}
	var g graphErrorBody
	if json.Unmarshal(body, &g) == nil {
		e.Code = g.Error.Code
		e.Message = g.Error.Message
		e.RequestID = g.Error.InnerError.RequestID
	}
	if e.RequestID == "" && h != nil {
		e.RequestID = h.Get("request-id")
	}
	if e.Message == "" {
		e.Message = http.StatusText(status)
	}
	return e
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("Graph API error %d", e.StatusCode)
	if e.Code != "" {
		msg += " (" + e.Code + ")"
	}
	msg += ": " + e.Message
	if hint := e.hint(); hint != "" {
		msg += "\nHint: " + hint
	}
	return msg
}

// hint maps a status (and a few Graph-specific codes) to the remedy a user actually needs.
func (e *APIError) hint() string {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return "token missing or expired — run `ms365 auth login -a <account>`"
	case http.StatusForbidden:
		if e.Code == "ErrorAccessDenied" || e.Code == "Authorization_RequestDenied" {
			return "the signed-in account lacks consent for this scope — re-run `ms365 auth login` to grant it, or ask a tenant admin (org tenants can require admin consent)"
		}
		return "insufficient permissions — check the granted scopes with `ms365 auth status`"
	case http.StatusNotFound:
		return "resource not found — verify the id with the matching `list` command"
	case http.StatusTooManyRequests:
		return "throttled by Microsoft Graph — the CLI already honored Retry-After; slow down or narrow the query"
	case http.StatusBadRequest:
		return "Graph rejected the query — check $filter/$search syntax (note: $search and $filter cannot be combined on messages)"
	}
	if e.StatusCode >= 500 {
		return "Microsoft Graph server error — usually transient, retry shortly"
	}
	return ""
}
