package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"pr-reviewer-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PRRepository struct {
	pool *pgxpool.Pool
}

func NewPRRepository(pool *pgxpool.Pool) *PRRepository {
	return &PRRepository{pool: pool}
}

func (r *PRRepository) CreatePR(ctx context.Context, pr *domain.PullRequest) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("failed to rollback transaction", "error", err)
		}
	}()

	query := `
        INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
        VALUES ($1, $2, $3, $4)
    `
	_, err = tx.Exec(ctx, query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	if len(pr.AssignedReviewers) > 0 {
		reviewerQuery := `
            INSERT INTO pr_reviewers (pull_request_id, user_id)
            VALUES ($1, $2)
        `
		for _, reviewerID := range pr.AssignedReviewers {
			_, err = tx.Exec(ctx, reviewerQuery, pr.PullRequestID, reviewerID)
			if err != nil {
				return fmt.Errorf("failed to assign reviewer: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *PRRepository) PRExists(ctx context.Context, prID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check PR existence: %w", err)
	}
	return exists, nil
}

func (r *PRRepository) GetPRByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	query := `
        SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
        FROM pull_requests
        WHERE pull_request_id = $1
    `
	var pr domain.PullRequest
	err := r.pool.QueryRow(ctx, query, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("PR not found")
		}
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	reviewersQuery := `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
	rows, err := r.pool.Query(ctx, reviewersQuery, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan reviewer: %w", err)
		}
		reviewers = append(reviewers, userID)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *PRRepository) MergePR(ctx context.Context, prID string) error {
	query := `
        UPDATE pull_requests
        SET status = $1, merged_at = $2
        WHERE pull_request_id = $3 AND status = $4
    `
	_, err := r.pool.Exec(ctx, query, domain.StatusMerged, time.Now(), prID, domain.StatusOpen)
	if err != nil {
		return fmt.Errorf("failed to merge PR: %w", err)
	}
	return nil
}

func (r *PRRepository) IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, prID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check reviewer assignment: %w", err)
	}
	return exists, nil
}

func (r *PRRepository) ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) error {
	query := `
        UPDATE pr_reviewers
        SET user_id = $1
        WHERE pull_request_id = $2 AND user_id = $3
    `
	result, err := r.pool.Exec(ctx, query, newUserID, prID, oldUserID)
	if err != nil {
		return fmt.Errorf("failed to reassign reviewer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("reviewer assignment not found")
	}
	return nil
}

func (r *PRRepository) GetPRsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	query := `
        SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
        FROM pull_requests pr
        INNER JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        WHERE prr.user_id = $1
        ORDER BY pr.created_at DESC
    `
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs by reviewer: %w", err)
	}
	defer rows.Close()

	var prs []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

// Batch operations here

func (r *PRRepository) GetOpenPRsByReviewers(ctx context.Context, userIDs []string) (map[string][]string, error) {
	if len(userIDs) == 0 {
		return make(map[string][]string), nil
	}

	query := `
        SELECT DISTINCT pr.pull_request_id, prr.user_id
        FROM pull_requests pr
        INNER JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        WHERE pr.status = 'OPEN' AND prr.user_id = ANY($1)
        ORDER BY pr.pull_request_id
    `

	rows, err := r.pool.Query(ctx, query, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get open PRs by reviewers: %w", err)
	}
	defer rows.Close()

	// map[pr_id][]reviewer_ids
	result := make(map[string][]string)
	for rows.Next() {
		var prID, reviewerID string
		if err := rows.Scan(&prID, &reviewerID); err != nil {
			return nil, fmt.Errorf("failed to scan PR reviewer: %w", err)
		}
		result[prID] = append(result[prID], reviewerID)
	}

	return result, nil
}

func (r *PRRepository) BatchReassignReviewers(ctx context.Context, reassignments map[string]map[string]string) error {
	if len(reassignments) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("failed to rollback transaction", "error", err)
		}
	}()

	updateQuery := `
        UPDATE pr_reviewers
        SET user_id = $1
        WHERE pull_request_id = $2 AND user_id = $3
    `

	for prID, reviewerMap := range reassignments {
		for oldReviewerID, newReviewerID := range reviewerMap {
			// check if such reviewer already exists for this PR
			checkQuery := `
                SELECT EXISTS(
                    SELECT 1 FROM pr_reviewers 
                    WHERE pull_request_id = $1 AND user_id = $2
                )
            `
			var exists bool
			err := tx.QueryRow(ctx, checkQuery, prID, newReviewerID).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check existing reviewer: %w", err)
			}

			if exists {
				continue
			}

			_, err = tx.Exec(ctx, updateQuery, newReviewerID, prID, oldReviewerID)
			if err != nil {
				return fmt.Errorf("failed to reassign reviewer in PR %s: %w", prID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PRRepository) GetPRWithReviewersAndAuthor(ctx context.Context, prID string) (string, string, []string, error) {
	query := `
        SELECT pr.author_id, u.team_name, COALESCE(array_agg(prr.user_id) FILTER (WHERE prr.user_id IS NOT NULL), '{}')
        FROM pull_requests pr
        INNER JOIN users u ON pr.author_id = u.user_id
        LEFT JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        WHERE pr.pull_request_id = $1
        GROUP BY pr.author_id, u.team_name
    `

	var authorID, teamName string
	var reviewers []string
	err := r.pool.QueryRow(ctx, query, prID).Scan(&authorID, &teamName, &reviewers)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get PR with reviewers and author: %w", err)
	}

	return authorID, teamName, reviewers, nil
}
