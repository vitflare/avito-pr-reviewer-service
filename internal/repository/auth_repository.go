package repository

import (
	"context"
	"fmt"
	"time"

	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) SaveToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	query := `
        INSERT INTO auth_tokens (user_id, token, expires_at)
        VALUES ($1, $2, $3)
    `
	_, err := r.pool.Exec(ctx, query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetTokenByUserID(ctx context.Context, userID string) (*domain.AuthToken, error) {
	query := `
        SELECT id, user_id, token, expires_at, created_at
        FROM auth_tokens
        WHERE user_id = $1 AND expires_at > NOW()
        ORDER BY created_at DESC
        LIMIT 1
    `
	var token domain.AuthToken
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.ExpiresAt,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	return &token, nil
}

func (r *AuthRepository) ValidateToken(ctx context.Context, token string) (string, error) {
	query := `
        SELECT user_id
        FROM auth_tokens
        WHERE token = $1 AND expires_at > NOW()
    `
	var userID string
	err := r.pool.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to validate token: %w", err)
	}
	return userID, nil
}
