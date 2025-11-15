package request

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id" validate:"required,min=1,max=255"`
	PullRequestName string `json:"pull_request_name" validate:"required,min=1,max=500"`
	AuthorID        string `json:"author_id" validate:"required,min=1,max=255"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id" validate:"required,min=1,max=255"`
}

type ReassignPRRequest struct {
	PullRequestID string `json:"pull_request_id" validate:"required,min=1,max=255"`
	OldUserID     string `json:"old_user_id" validate:"required,min=1,max=255"`
}
