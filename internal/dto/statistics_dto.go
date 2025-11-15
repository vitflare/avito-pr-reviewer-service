package dto

type PRReviewerDistributionDTO struct {
	PullRequestID   string   `json:"pull_request_id"`
	PullRequestName string   `json:"pull_request_name"`
	AuthorID        string   `json:"author_id"`
	Status          string   `json:"status"`
	ReviewerIDs     []string `json:"reviewer_ids"`
	ReviewerCount   int      `json:"reviewer_count"`
}
