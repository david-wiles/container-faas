package internal

import (
	"encoding/json"
	"errors"
	"net/http"
)

type basicResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// http.Error with json response type instead of text/plain
func ErrorResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(newError(message))
}

// Parses a message and creates a simple json body to return
func BasicResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(newResponse(message))
}

func newError(message string) []byte {
	e := &basicResponse{true, message}
	b, err := json.Marshal(e)
	if err != nil {
		G.Logger.Error("Could not write error json")
		return []byte("{\"error\":true}")
	}
	return b
}

func newResponse(message string) []byte {
	e := &basicResponse{false, message}
	b, err := json.Marshal(e)
	if err != nil {
		G.Logger.Error("Could not write error json")
		return []byte("{\"error\":true}")
	}
	return b
}

type NotFoundHandler struct{}

// Default 404 handler
func (h *NotFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ErrorResponse(w, "Resource not found", 404)
}

func trimPath(base string, r *http.Request) (string, error) {
	baseLen := len(base)

	if len(r.URL.Path) < baseLen {
		return "", errors.New("Invalid request")
	}

	return r.URL.Path[baseLen:], nil
}
