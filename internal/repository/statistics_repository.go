package repository

import (
	"context"
	"fmt"

	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatisticsRepository struct {
	pool *pgxpool.Pool
}

func NewStatisticsRepository(pool *pgxpool.Pool) *StatisticsRepository {
	return &StatisticsRepository{pool: pool}
}

func (r *StatisticsRepository) GetStatistics(ctx context.Context) (*domain.Statistics, error) {
	stats := &domain.Statistics{}

	// General PR statistics
	prStatsQuery := `
        SELECT 
            COUNT(*) as total,
            COUNT(CASE WHEN status = 'OPEN' THEN 1 END) as open,
            COUNT(CASE WHEN status = 'MERGED' THEN 1 END) as merged
        FROM pull_requests
    `
	err := r.pool.QueryRow(ctx, prStatsQuery).Scan(&stats.TotalPRs, &stats.OpenPRs, &stats.MergedPRs)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR stats: %w", err)
	}

	// User statistics
	userStatsQuery := `
        SELECT 
            COUNT(*) as total,
            COUNT(CASE WHEN is_active = true THEN 1 END) as active
        FROM users
    `
	err = r.pool.QueryRow(ctx, userStatsQuery).Scan(&stats.TotalUsers, &stats.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Command count
	teamStatsQuery := `SELECT COUNT(*) FROM teams`
	err = r.pool.QueryRow(ctx, teamStatsQuery).Scan(&stats.TotalTeams)
	if err != nil {
		return nil, fmt.Errorf("failed to get team stats: %w", err)
	}

	// Assignment statistics by user
	userAssignmentsQuery := `
        SELECT 
            u.user_id,
            u.username,
            u.team_name,
            COUNT(prr.id) as total_assignments,
            COUNT(CASE WHEN pr.status = 'OPEN' THEN 1 END) as open_assignments,
            COUNT(CASE WHEN pr.status = 'MERGED' THEN 1 END) as merged_assignments
        FROM users u
        LEFT JOIN pr_reviewers prr ON u.user_id = prr.user_id
        LEFT JOIN pull_requests pr ON prr.pull_request_id = pr.pull_request_id
        GROUP BY u.user_id, u.username, u.team_name
        HAVING COUNT(prr.id) > 0
        ORDER BY total_assignments DESC
    `
	rows, err := r.pool.Query(ctx, userAssignmentsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get user assignments: %w", err)
	}
	defer rows.Close()

	var userAssignments []domain.UserAssignmentStat
	for rows.Next() {
		var ua domain.UserAssignmentStat
		if err := rows.Scan(
			&ua.UserID,
			&ua.Username,
			&ua.TeamName,
			&ua.TotalAssignments,
			&ua.OpenAssignments,
			&ua.MergedAssignments,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user assignment: %w", err)
		}
		userAssignments = append(userAssignments, ua)
	}
	stats.UserAssignments = userAssignments

	return stats, nil
}
