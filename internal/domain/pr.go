package domain

import "time"

const (
	StatusOpen   = "OPEN"
	StatusMerged = "MERGED"

	TeamAdmins = "admins"
)

type PullRequest struct {
	CreatedAt         *time.Time `json:"created_at,omitempty"`
	MergedAt          *time.Time `json:"merged_at,omitempty"`
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
}

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type PRReassignment struct {
	PullRequestID string
	OldReviewers  []string
	NewReviewers  []string
}
