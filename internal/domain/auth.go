package domain

import "time"

type AuthToken struct {
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ID        int64     `json:"id"`
}
