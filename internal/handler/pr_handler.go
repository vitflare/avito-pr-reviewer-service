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

type PRService interface {
	CreatePR(ctx context.Context, pr *domain.PullRequest) (*domain.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, *domain.PullRequest, error)
}

type PRHandler struct {
	service   PRService
	validator *validator.Validate
}

func NewPRHandler(service PRService, validator *validator.Validate) *PRHandler {
	return &PRHandler{
		service:   service,
		validator: validator,
	}
}

// CreatePR godoc
// @Summary Create a new pull request
// @Description Create a PR and automatically assign up to 2 reviewers from author's team
// @Tags PullRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreatePRRequest true "PR creation request"
// @Success 201 {object} response.PRResponse "PR created successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Author not found"
// @Failure 409 {object} dto.ErrorResponse "PR already exists"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /pullRequest/create [post]
func (h *PRHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req request.CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	pr := mapper.MapCreatePRRequestToDomain(&req)

	createdPR, err := h.service.CreatePR(r.Context(), pr)
	if err != nil {
		switch {
		case errors.Is(err, my_errors.ErrPRAlreadyExists):
			respondWithError(w, http.StatusConflict, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodePRExists,
					Message: my_errors.ErrPRAlreadyExists.Error(),
				},
			})
			return
		case errors.Is(err, my_errors.ErrAuthorNotFound):
			respondWithError(w, http.StatusNotFound, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNotFound,
					Message: my_errors.ErrAuthorNotFound.Error(),
				},
			})
			return
		default:
			respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
			return
		}
	}

	resp := response.PRResponse{
		PR: mapper.MapDomainPRToDTO(createdPR),
	}

	respondJSON(w, http.StatusCreated, resp)
}

// MergePR godoc
// @Summary Merge a pull request
// @Description Mark PR as MERGED (idempotent operation)
// @Tags PullRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.MergePRRequest true "Merge PR request"
// @Success 200 {object} response.PRResponse "PR merged successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "PR not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /pullRequest/merge [post]
func (h *PRHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req request.MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	mergedPR, err := h.service.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		if errors.Is(err, my_errors.ErrPRNotFound) {
			respondWithError(w, http.StatusNotFound, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNotFound,
					Message: my_errors.ErrPRNotFound.Error(),
				},
			})
			return
		}
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := response.PRResponse{
		PR: mapper.MapDomainPRToDTO(mergedPR),
	}

	respondJSON(w, http.StatusOK, resp)
}

// ReassignReviewer godoc
// @Summary Reassign a reviewer on PR
// @Description Replace one reviewer with another from the same team
// @Tags PullRequests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.ReassignPRRequest true "Reassign reviewer request"
// @Success 200 {object} response.ReassignResponse "Reviewer reassigned successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "PR or user not found"
// @Failure 409 {object} dto.ErrorResponse "Cannot reassign (PR merged, user not assigned, or no candidates)"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /pullRequest/reassign [post]
func (h *PRHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req request.ReassignPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	newReviewerID, updatedPR, err := h.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		switch {
		case errors.Is(err, my_errors.ErrPRNotFound):
			respondWithError(w, http.StatusNotFound, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNotFound,
					Message: my_errors.ErrPRNotFound.Error(),
				},
			})
			return
		case errors.Is(err, my_errors.ErrUserNotFound):
			respondWithError(w, http.StatusNotFound, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNotFound,
					Message: my_errors.ErrUserNotFound.Error(),
				},
			})
			return
		case errors.Is(err, my_errors.ErrPRAlreadyMerged):
			respondWithError(w, http.StatusConflict, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodePRMerged,
					Message: my_errors.ErrPRAlreadyMerged.Error(),
				},
			})
			return
		case errors.Is(err, my_errors.ErrReviewerIsNotAssigned):
			respondWithError(w, http.StatusConflict, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNotAssigned,
					Message: my_errors.ErrReviewerIsNotAssigned.Error(),
				},
			})
			return
		case errors.Is(err, my_errors.ErrNoActiveReviewerWasFound):
			respondWithError(w, http.StatusConflict, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeNoCandidate,
					Message: my_errors.ErrNoActiveReviewerWasFound.Error(),
				},
			})
			return
		default:
			respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
			return
		}
	}

	resp := response.ReassignResponse{
		PR:         mapper.MapDomainPRToDTO(updatedPR),
		ReplacedBy: newReviewerID,
	}

	respondJSON(w, http.StatusOK, resp)
}
