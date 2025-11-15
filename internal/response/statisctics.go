package response

import "pr-reviewer-service/internal/dto"

type StatisticsResponse struct {
	UserAssignments []dto.UserAssignmentStatDTO `json:"user_assignments"`
	TotalPRs        int                         `json:"total_prs"`
	OpenPRs         int                         `json:"open_prs"`
	MergedPRs       int                         `json:"merged_prs"`
	TotalUsers      int                         `json:"total_users"`
	ActiveUsers     int                         `json:"active_users"`
	TotalTeams      int                         `json:"total_teams"`
}
