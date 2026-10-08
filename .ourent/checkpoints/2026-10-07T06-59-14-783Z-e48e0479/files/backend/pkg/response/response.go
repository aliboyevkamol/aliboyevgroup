// Package response implements the JSON envelope used by every endpoint.
package response

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func E(status int, code, msg string) *Error { return &Error{Status: status, Code: code, Message: msg} }
func Validation(f map[string]string) *Error {
	return &Error{Status: 422, Code: "VALIDATION_ERROR", Message: "Please check the highlighted fields.", Fields: f}
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, status int, data any) {
	write(w, status, map[string]any{"success": true, "data": data})
}

func List(w http.ResponseWriter, data any, p Pagination) {
	write(w, 200, map[string]any{"success": true, "data": data, "pagination": p})
}

func Fail(w http.ResponseWriter, e *Error) {
	errObj := map[string]any{"code": e.Code}
	if len(e.Fields) > 0 {
		errObj["fields"] = e.Fields
	}
	write(w, e.Status, map[string]any{"success": false, "message": e.Message, "error": errObj})
}

func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// OKRaw writes a success envelope around an already-encoded JSON value.
func OKRaw(w http.ResponseWriter, raw string) { OK(w, 200, json.RawMessage(raw)) }
