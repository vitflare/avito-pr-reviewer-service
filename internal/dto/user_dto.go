package dto

type UserDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type UserAssignmentStatDTO struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	TeamName          string `json:"team_name"`
	TotalAssignments  int    `json:"total_assignments"`
	OpenAssignments   int    `json:"open_assignments"`
	MergedAssignments int    `json:"merged_assignments"`
}
