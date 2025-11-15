package handler

import (
	"context"
	"net/http"

	"pr-reviewer-service/internal/dto"
	"pr-reviewer-service/internal/mapper"

	"pr-reviewer-service/internal/domain"
)

type StatisticsService interface {
	GetStatistics(ctx context.Context) (*domain.Statistics, error)
}

type StatisticsHandler struct {
	service StatisticsService
}

func NewStatisticsHandler(service StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{
		service: service,
	}
}

// GetStatistics godoc
// @Summary Get service statistics
// @Description Get comprehensive statistics about PRs, users, teams, and reviewer assignments
// @Tags Statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.StatisticsResponse "Statistics retrieved successfully"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /statistics [get]
func (h *StatisticsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStatistics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := mapper.MapDomainStatisticsToDTO(stats)
	respondJSON(w, http.StatusOK, resp)
}
