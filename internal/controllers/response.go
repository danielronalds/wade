package controllers

// TODO: Review properly

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Message string `json:"message"`
} // @name handlers.errorResponse

func WriteJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, errorResponse{Message: message})
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSONError(w, statusCode, message)
}
