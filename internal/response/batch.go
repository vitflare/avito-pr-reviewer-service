package response

type BatchDeactivateResponse struct {
	DeactivatedUsers   []string             `json:"deactivated_users"`
	ReassignedPRs      []PRReassignmentInfo `json:"reassigned_prs"`
	SkippedUsers       []string             `json:"skipped_users"`
	TotalDeactivated   int                  `json:"total_deactivated"`
	TotalPRsReassigned int                  `json:"total_prs_reassigned"`
	ProcessingTimeMs   int64                `json:"processing_time_ms"`
}

type PRReassignmentInfo struct {
	PullRequestID string   `json:"pull_request_id"`
	OldReviewers  []string `json:"old_reviewers"`
	NewReviewers  []string `json:"new_reviewers"`
}
