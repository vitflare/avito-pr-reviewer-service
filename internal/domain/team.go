package domain

import "time"

type Team struct {
	CreatedAt time.Time    `json:"created_at"`
	TeamName  string       `json:"team_name"`
	Members   []TeamMember `json:"members"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}
