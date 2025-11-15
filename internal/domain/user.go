package domain

import "time"

type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	TeamName  string    `json:"team_name"`
	IsActive  bool      `json:"is_active"`
}
