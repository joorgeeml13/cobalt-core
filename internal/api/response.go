package api

import (
	"encoding/json"
	"net/http"
)

// Response representa el formato unificado y estándar de respuesta JSON de Cobalt Core.
type Response struct {
	Success bool  `json:"success"`
	Data    any   `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError encapsula los detalles de un error estándar.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// JSON escribe una respuesta JSON con el código de estado indicado.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

// Error escribe una respuesta de error estándar en formato JSON.
func Error(w http.ResponseWriter, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
