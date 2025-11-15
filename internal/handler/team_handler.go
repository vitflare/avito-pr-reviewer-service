package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"pr-reviewer-service/internal/mapper"
	"pr-reviewer-service/internal/my_errors"
	"pr-reviewer-service/internal/request"
	"pr-reviewer-service/internal/response"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/dto"

	"github.com/go-playground/validator/v10"
)

type TeamService interface {
	CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error)
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
	GetAllTeams(ctx context.Context) ([]domain.Team, error)
}

type TeamHandler struct {
	service   TeamService
	validator *validator.Validate
}

func NewTeamHandler(service TeamService, validator *validator.Validate) *TeamHandler {
	return &TeamHandler{
		service:   service,
		validator: validator,
	}
}

// CreateTeam godoc
// @Summary Create a new team with members
// @Description Create a team and add/update users as members
// @Tags Teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreateTeamRequest true "Team creation request"
// @Success 201 {object} response.TeamResponse "Team created successfully"
// @Failure 400 {object} dto.ErrorResponse "Team already exists or validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /team/add [post]
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req request.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "invalid request body")
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "validation error: "+err.Error())
		return
	}

	team := mapper.MapCreateTeamRequestToDomain(&req)

	createdTeam, err := h.service.CreateTeam(r.Context(), team)
	if err != nil {
		if errors.Is(err, my_errors.ErrTeamAlreadyExists) {
			respondWithError(w, http.StatusBadRequest, &dto.ErrorResponse{
				Error: dto.ErrorDetail{
					Code:    dto.ErrCodeTeamExists,
					Message: my_errors.ErrTeamAlreadyExists.Error(),
				},
			})
			return
		}
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	resp := response.TeamResponse{
		Team: mapper.MapDomainTeamToDTO(createdTeam),
	}

	respondJSON(w, http.StatusCreated, resp)
}

// GetTeam godoc
// @Summary Get team by name
// @Description Get team information with all members
// @Tags Teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param team_name query string true "Team name"
// @Success 200 {object} dto.TeamDTO "Team retrieved successfully"
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Team not found"
// @Router /team/get [get]
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondError(w, http.StatusBadRequest, dto.ErrCodeNotFound, "team_name query parameter is required")
		return
	}

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		respondWithError(w, http.StatusNotFound, &dto.ErrorResponse{
			Error: dto.ErrorDetail{
				Code:    dto.ErrCodeNotFound,
				Message: my_errors.ErrTeamNotFound.Error(),
			},
		})
		return
	}

	teamDTO := mapper.MapDomainTeamToDTO(team)
	respondJSON(w, http.StatusOK, teamDTO)
}

// ListAllTeams godoc
// @Summary List all teams (Admin only)
// @Description Get list of all teams with their members
// @Tags Teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.AllTeamsResponse "Teams retrieved successfully"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - Admin access required"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /admin/teams [get]
func (h *TeamHandler) ListAllTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.service.GetAllTeams(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, dto.ErrCodeNotFound, err.Error())
		return
	}

	teamDTOs := make([]dto.TeamDTO, len(teams))
	for i, team := range teams {
		teamDTOs[i] = mapper.MapDomainTeamToDTO(&team)
	}

	resp := response.AllTeamsResponse{
		Teams: teamDTOs,
		Count: len(teamDTOs),
	}

	respondJSON(w, http.StatusOK, resp)
}
