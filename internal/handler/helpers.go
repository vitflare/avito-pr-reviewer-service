package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pr-reviewer-service/internal/dto"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Warn("failed to encode JSON response", "error", err)
	}
}

//nolint:unparam
func respondError(w http.ResponseWriter, status int, code, message string) {
	respondWithError(w, status, &dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func respondWithError(w http.ResponseWriter, status int, errResp *dto.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Warn("failed to encode error response", "error", err)
	}
}
