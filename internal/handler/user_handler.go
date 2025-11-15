package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"pr-reviewer-service/internal/dto"
	"pr-reviewer-service/internal/mapper"
	"pr-reviewer-service/internal/my_errors"
	"pr-reviewer-service/internal/request"
	"pr-reviewer-service/internal/response"

	"github.com/go-playground/validator/v10"

	"pr-reviewer-service/internal/domain"
)

type UserService interface {
	SetUserActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	BatchDeactivateUsers(ctx context.Context, userIDs []string) (*domain.BatchDeactivateResult, error)
	BatchDeactivateTeam(ctx context.Context, teamName string) (*domain.BatchDeactivateResult, error)
}

type PRServiceForUser interface {
	GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
}

type UserHandler struct {
	userService UserService
	prService   PRServiceForUser
	validator   *validator.Validate
}

func NewUserHandler(userService UserService, prService PRServiceForUser, validator *validator.Validate) *UserHandler {
	return &UserHandler{
		userService: userService,
		prService:   prService,
		validator:   validator,
	}
}

// SetIsActive godoc
// @Summary Set user active status (Admin only)
// @Description Update user's active status. Admin users cannot be deactivated.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.SetUserActiveRequest true "Set active request"
// @Success 200 {object} response.UserResponse "User status updated successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - Admin access required or cannot deactivate admin"
// @Failure 404 {object} dto.ErrorResponse "User not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /users/setIsActive [post]
func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req request.SetUserActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	user, err := h.userService.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, my_errors.ErrUserNotFound) {
			respondWithError(w, http.StatusNotFound, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNotFound,
					Message: my_errors.ErrUserNotFound.Error(),
				},
			})
			return
		}
		if errors.Is(err, my_errors.ErrCannotDeactivateAdmin) {
			respondError(w, http.StatusForbidden, dto.ErrCodeNotFound, my_errors.ErrCannotDeactivateAdmin.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := response.UserResponse{
		User: mapper.MapDomainUserToDTO(user),
	}

	respondJSON(w, http.StatusOK, resp)
}

// GetReview godoc
// @Summary Get PRs assigned to user
// @Description Get list of pull requests where user is assigned as reviewer
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id query string true "User ID"
// @Success 200 {object} response.UserReviewsResponse "User reviews retrieved successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /users/getReview [get]
func (h *UserHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "user_id query parameter is required")
		return
	}

	prs, err := h.prService.GetUserReviews(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := response.UserReviewsResponse{
		UserID:       userID,
		PullRequests: mapper.MapDomainPRsShortToDTO(prs),
	}

	respondJSON(w, http.StatusOK, resp)
}

// ListAllUsers godoc
// @Summary List all users (Admin only)
// @Description Get list of all users in the system
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.AllUsersResponse "Users retrieved successfully"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - Admin access required"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /admin/users [get]
func (h *UserHandler) ListAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.GetAllUsers(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := response.AllUsersResponse{
		Users: mapper.MapDomainUsersToDTO(users),
		Count: len(users),
	}

	respondJSON(w, http.StatusOK, resp)
}

// BatchDeactivateTeam godoc
// @Summary Batch deactivate team members (Admin only)
// @Description Deactivate all members of a team and safely reassign their open PRs
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.BatchDeactivateTeamRequest true "Batch deactivate team request"
// @Success 200 {object} response.BatchDeactivateResponse "Team members deactivated and PRs reassigned"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - Admin access required or cannot deactivate admin team"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /users/batchDeactivateTeam [post]
func (h *UserHandler) BatchDeactivateTeam(w http.ResponseWriter, r *http.Request) {
	var req request.BatchDeactivateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	result, err := h.userService.BatchDeactivateTeam(r.Context(), req.TeamName)
	if err != nil {
		if errors.Is(err, my_errors.ErrCannotDeactivateAdmin) {
			respondError(w, http.StatusForbidden, dto.ErrCodeNotFound, my_errors.ErrCannotDeactivateAdmin.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := mapper.MapBatchDeactivateResultToDTO(result)
	respondJSON(w, http.StatusOK, resp)
}

// BatchDeactivateUsers godoc
// @Summary Batch deactivate users (Admin only)
// @Description Deactivate specified users and safely reassign their open PRs
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.BatchDeactivateUsersRequest true "Batch deactivate users request"
// @Success 200 {object} response.BatchDeactivateResponse "Users deactivated and PRs reassigned"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - Admin access required"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /users/batchDeactivateUsers [post]
func (h *UserHandler) BatchDeactivateUsers(w http.ResponseWriter, r *http.Request) {
	var req request.BatchDeactivateUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	result, err := h.userService.BatchDeactivateUsers(r.Context(), req.UserIDs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := mapper.MapBatchDeactivateResultToDTO(result)
	respondJSON(w, http.StatusOK, resp)
}
