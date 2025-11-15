package repository

import (
	"context"
	"fmt"

	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateOrUpdateUser(ctx context.Context, user *domain.User) error {
	query := `
        INSERT INTO users (user_id, username, team_name, is_active)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id)
        DO UPDATE SET
            username = EXCLUDED.username,
            team_name = EXCLUDED.team_name,
            is_active = EXCLUDED.is_active,
            updated_at = NOW()
    `
	_, err := r.pool.Exec(ctx, query, user.UserID, user.Username, user.TeamName, user.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create/update user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
        SELECT user_id, username, team_name, is_active, created_at, updated_at
        FROM users
        WHERE user_id = $1
    `
	var user domain.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.UserID,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	query := `
        UPDATE users
        SET is_active = $1, updated_at = NOW()
        WHERE user_id = $2
    `
	result, err := r.pool.Exec(ctx, query, isActive, userID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error) {
	query := `
        SELECT user_id, username, team_name, is_active
        FROM users
        WHERE team_name = $1 AND is_active = true AND user_id != $2
    `
	rows, err := r.pool.Query(ctx, query, teamName, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active team members: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	query := `
        SELECT user_id, username, team_name, is_active, created_at, updated_at
        FROM users
        ORDER BY team_name, username
    `
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

// Batch operations

func (r *UserRepository) BatchDeactivateUsers(ctx context.Context, userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return []string{}, nil
	}

	query := `
        UPDATE users
        SET is_active = false, updated_at = NOW()
        WHERE user_id = ANY($1) AND team_name != 'admins' AND is_active = true
        RETURNING user_id
    `

	rows, err := r.pool.Query(ctx, query, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch deactivate users: %w", err)
	}
	defer rows.Close()

	var deactivated []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan deactivated user: %w", err)
		}
		deactivated = append(deactivated, userID)
	}

	return deactivated, nil
}

func (r *UserRepository) GetTeamMemberIDs(ctx context.Context, teamName string) ([]string, error) {
	query := `
        SELECT user_id 
        FROM users 
        WHERE team_name = $1 AND team_name != 'admins'
    `

	rows, err := r.pool.Query(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team member IDs: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}
