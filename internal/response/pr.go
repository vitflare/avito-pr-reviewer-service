package response

import (
	"pr-reviewer-service/internal/dto"
)

type PRResponse struct {
	PR dto.PullRequestDTO `json:"pr"`
}

type ReassignResponse struct {
	ReplacedBy string             `json:"replaced_by"`
	PR         dto.PullRequestDTO `json:"pr"`
}

type UserReviewsResponse struct {
	UserID       string                    `json:"user_id"`
	PullRequests []dto.PullRequestShortDTO `json:"pull_requests"`
}
