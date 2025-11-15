package response

import "pr-reviewer-service/internal/dto"

type TeamResponse struct {
	Team dto.TeamDTO `json:"team"`
}

type AllTeamsResponse struct {
	Teams []dto.TeamDTO `json:"teams"`
	Count int           `json:"count"`
}
