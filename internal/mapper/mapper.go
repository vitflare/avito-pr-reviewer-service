package mapper

import (
	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/dto"
	"pr-reviewer-service/internal/request"
	"pr-reviewer-service/internal/response"
)

// Team mappers
func MapDomainTeamToDTO(team *domain.Team) dto.TeamDTO {
	members := make([]dto.TeamMemberDTO, len(team.Members))
	for i, m := range team.Members {
		members[i] = dto.TeamMemberDTO{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return dto.TeamDTO{
		TeamName: team.TeamName,
		Members:  members,
	}
}

func MapCreateTeamRequestToDomain(req *request.CreateTeamRequest) *domain.Team {
	members := make([]domain.TeamMember, len(req.Members))
	for i, m := range req.Members {
		members[i] = domain.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return &domain.Team{
		TeamName: req.TeamName,
		Members:  members,
	}
}

// User mappers
func MapDomainUserToDTO(user *domain.User) dto.UserDTO {
	return dto.UserDTO{
		UserID:   user.UserID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func MapDomainUsersToDTO(users []domain.User) []dto.UserDTO {
	result := make([]dto.UserDTO, len(users))
	for i, u := range users {
		result[i] = MapDomainUserToDTO(&u)
	}
	return result
}

// PR mappers
func MapDomainPRToDTO(pr *domain.PullRequest) dto.PullRequestDTO {
	return dto.PullRequestDTO{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}

func MapDomainPRShortToDTO(pr *domain.PullRequestShort) dto.PullRequestShortDTO {
	return dto.PullRequestShortDTO{
		PullRequestID:   pr.PullRequestID,
		PullRequestName: pr.PullRequestName,
		AuthorID:        pr.AuthorID,
		Status:          pr.Status,
	}
}

func MapDomainPRsShortToDTO(prs []domain.PullRequestShort) []dto.PullRequestShortDTO {
	result := make([]dto.PullRequestShortDTO, len(prs))
	for i, pr := range prs {
		result[i] = MapDomainPRShortToDTO(&pr)
	}
	return result
}

func MapCreatePRRequestToDomain(req *request.CreatePRRequest) *domain.PullRequest {
	return &domain.PullRequest{
		PullRequestID:     req.PullRequestID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          req.AuthorID,
		Status:            domain.StatusOpen,
		AssignedReviewers: []string{},
	}
}

// Statistics mappers
func MapDomainStatisticsToDTO(stats *domain.Statistics) response.StatisticsResponse {
	userAssignments := make([]dto.UserAssignmentStatDTO, len(stats.UserAssignments))
	for i, ua := range stats.UserAssignments {
		userAssignments[i] = dto.UserAssignmentStatDTO{
			UserID:            ua.UserID,
			Username:          ua.Username,
			TeamName:          ua.TeamName,
			TotalAssignments:  ua.TotalAssignments,
			OpenAssignments:   ua.OpenAssignments,
			MergedAssignments: ua.MergedAssignments,
		}
	}

	return response.StatisticsResponse{
		TotalPRs:        stats.TotalPRs,
		OpenPRs:         stats.OpenPRs,
		MergedPRs:       stats.MergedPRs,
		TotalUsers:      stats.TotalUsers,
		ActiveUsers:     stats.ActiveUsers,
		TotalTeams:      stats.TotalTeams,
		UserAssignments: userAssignments,
	}
}

// Batch mapper
func MapBatchDeactivateResultToDTO(result *domain.BatchDeactivateResult) response.BatchDeactivateResponse {
	reassignedPRs := make([]response.PRReassignmentInfo, len(result.ReassignedPRs))
	for i, pr := range result.ReassignedPRs {
		reassignedPRs[i] = response.PRReassignmentInfo{
			PullRequestID: pr.PullRequestID,
			OldReviewers:  pr.OldReviewers,
			NewReviewers:  pr.NewReviewers,
		}
	}

	return response.BatchDeactivateResponse{
		DeactivatedUsers:   result.DeactivatedUsers,
		ReassignedPRs:      reassignedPRs,
		SkippedUsers:       result.SkippedUsers,
		TotalDeactivated:   len(result.DeactivatedUsers),
		TotalPRsReassigned: len(result.ReassignedPRs),
		ProcessingTimeMs:   result.ProcessingTime.Milliseconds(),
	}
}
