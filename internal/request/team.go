package request

type CreateTeamRequest struct {
	TeamName string            `json:"team_name" validate:"required,min=1,max=255"`
	Members  []TeamMemberInput `json:"members" validate:"required,min=1,dive"`
}

type TeamMemberInput struct {
	UserID   string `json:"user_id" validate:"required,min=1,max=255"`
	Username string `json:"username" validate:"required,min=1,max=255"`
	IsActive bool   `json:"is_active"`
}

type BatchDeactivateTeamRequest struct {
	TeamName string `json:"team_name" validate:"required,min=1,max=255"`
}
