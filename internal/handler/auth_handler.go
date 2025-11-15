package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"pr-reviewer-service/internal/dto"
	"pr-reviewer-service/internal/request"
	"pr-reviewer-service/internal/response"

	"github.com/go-playground/validator/v10"
)

type AuthService interface {
	GenerateToken(ctx context.Context, userID string) (string, error)
}

type AuthHandler struct {
	authService AuthService
	validator   *validator.Validate
}

func NewAuthHandler(authService AuthService, validator *validator.Validate) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator,
	}
}

// Login godoc
// @Summary Generate JWT token for user
// @Description Generate authentication token for a user by user_id
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "Login request"
// @Success 200 {object} response.LoginResponse "Token generated successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	token, err := h.authService.GenerateToken(r.Context(), req.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := response.LoginResponse{
		Token:  token,
		UserID: req.UserID,
	}

	respondJSON(w, http.StatusOK, resp)
}
