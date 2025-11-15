package repository

import (
	"context"
	"fmt"

	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepository struct {
	pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

func (r *TeamRepository) CreateTeam(ctx context.Context, teamName string) error {
	query := `INSERT INTO teams (team_name) VALUES ($1)`
	_, err := r.pool.Exec(ctx, query, teamName)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func (r *TeamRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, teamName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check team existence: %w", err)
	}
	return exists, nil
}

func (r *TeamRepository) GetTeamWithMembers(ctx context.Context, teamName string) (*domain.Team, error) {
	teamQuery := `SELECT team_name, created_at FROM teams WHERE team_name = $1`
	var team domain.Team
	err := r.pool.QueryRow(ctx, teamQuery, teamName).Scan(&team.TeamName, &team.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("team not found")
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	membersQuery := `
        SELECT user_id, username, is_active
        FROM users
        WHERE team_name = $1
        ORDER BY username
    `
	rows, err := r.pool.Query(ctx, membersQuery, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	defer rows.Close()

	var members []domain.TeamMember
	for rows.Next() {
		var member domain.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, member)
	}

	team.Members = members
	return &team, nil
}

func (r *TeamRepository) GetAllTeams(ctx context.Context) ([]domain.Team, error) {
	query := `SELECT team_name, created_at FROM teams ORDER BY team_name`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all teams: %w", err)
	}
	defer rows.Close()

	var teams []domain.Team
	for rows.Next() {
		var team domain.Team
		if err := rows.Scan(&team.TeamName, &team.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}

		// Get members for each team
		membersQuery := `
            SELECT user_id, username, is_active
            FROM users
            WHERE team_name = $1
            ORDER BY username
        `
		memberRows, err := r.pool.Query(ctx, membersQuery, team.TeamName)
		if err != nil {
			return nil, fmt.Errorf("failed to get team members: %w", err)
		}

		members := []domain.TeamMember{}
		for memberRows.Next() {
			var member domain.TeamMember
			if err := memberRows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
				memberRows.Close()
				return nil, fmt.Errorf("failed to scan member: %w", err)
			}
			members = append(members, member)
		}
		memberRows.Close()

		team.Members = members
		teams = append(teams, team)
	}

	return teams, nil
}
