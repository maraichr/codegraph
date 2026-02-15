package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/maraichr/lattice/pkg/apierr"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeAPIError writes a structured error response and logs 5xx errors.
func writeAPIError(w http.ResponseWriter, logger *slog.Logger, e *apierr.Error) {
	if e.Status() >= 500 && logger != nil {
		logger.Error(e.Message(), slog.String("code", string(e.Code())), slog.String("error", e.Error()))
	}
	writeJSON(w, e.Status(), e.Response())
}
