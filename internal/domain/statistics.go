package domain

type Statistics struct {
	UserAssignments []UserAssignmentStat `json:"user_assignments"`
	TotalPRs        int                  `json:"total_prs"`
	OpenPRs         int                  `json:"open_prs"`
	MergedPRs       int                  `json:"merged_prs"`
	TotalUsers      int                  `json:"total_users"`
	ActiveUsers     int                  `json:"active_users"`
	TotalTeams      int                  `json:"total_teams"`
}

type UserAssignmentStat struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	TeamName          string `json:"team_name"`
	TotalAssignments  int    `json:"total_assignments"`
	OpenAssignments   int    `json:"open_assignments"`
	MergedAssignments int    `json:"merged_assignments"`
}
