package tests

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"testing"
	"time"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"
	"pr-reviewer-service/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// prReviewers stores the initial distribution of reviewers for a PR
type prReviewers struct {
	prID      string
	reviewers []string
}

func BenchmarkBatchDeactivateUsers(b *testing.B) {
	ctx := context.Background()

	cfg, err := config.Load(".env.tests")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := config.MustInitDB(ctx, *cfg)
	require.NoError(b, err)
	defer func() {
		cleanupBenchmarkDB(b, pool)
		pool.Close()
	}()

	teamRepo := repository.NewTeamRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	prRepo := repository.NewPRRepository(pool)

	teamService := service.NewTeamService(teamRepo, userRepo)
	userService := service.NewUserService(userRepo, userRepo, prRepo)

	testCases := []struct {
		name       string
		teamSize   int
		prCount    int
		deactivate int
	}{
		{"Small_10users_5deactivate_20prs", 10, 20, 5},
		{"Medium_50users_25deactivate_100prs", 50, 100, 25},
		{"Large_100users_50deactivate_500prs", 100, 500, 50},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			cleanupBenchmarkDB(b, pool)
			createBenchmarkAdmin(b, pool)

			teamName := fmt.Sprintf("team_%d", time.Now().UnixNano())
			members := make([]domain.TeamMember, tc.teamSize)
			for i := 0; i < tc.teamSize; i++ {
				members[i] = domain.TeamMember{
					UserID:   fmt.Sprintf("u%d", i),
					Username: fmt.Sprintf("User%d", i),
					IsActive: true,
				}
			}

			team := &domain.Team{
				TeamName: teamName,
				Members:  members,
			}

			_, err := teamService.CreateTeam(ctx, team)
			require.NoError(b, err)

			originalAssignments := make([]prReviewers, 0, tc.prCount)

			for i := 0; i < tc.prCount; i++ {
				authorID := members[rand.Intn(len(members))].UserID
				pr := &domain.PullRequest{
					PullRequestID:   fmt.Sprintf("pr%d", i),
					PullRequestName: fmt.Sprintf("PR %d", i),
					AuthorID:        authorID,
					Status:          domain.StatusOpen,
				}

				reviewers := make([]string, 0, 2)
				usedReviewers := make(map[string]bool)
				usedReviewers[authorID] = true

				for len(reviewers) < 2 && len(usedReviewers) < len(members) {
					reviewerID := members[rand.Intn(len(members))].UserID
					if !usedReviewers[reviewerID] {
						reviewers = append(reviewers, reviewerID)
						usedReviewers[reviewerID] = true
					}
				}
				pr.AssignedReviewers = reviewers

				err := prRepo.CreatePR(ctx, pr)
				require.NoError(b, err)

				originalAssignments = append(originalAssignments, prReviewers{
					prID:      pr.PullRequestID,
					reviewers: reviewers,
				})
			}

			// select users to deactivate
			usersToDeactivate := make([]string, tc.deactivate)
			for i := 0; i < tc.deactivate; i++ {
				usersToDeactivate[i] = members[i].UserID
			}

			// start benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := userService.BatchDeactivateUsers(ctx, usersToDeactivate)
				require.NoError(b, err)

				if i == 0 {
					b.Logf("First run - Deactivated: %d users, PRs reassigned: %d, Time: %v",
						len(result.DeactivatedUsers),
						len(result.ReassignedPRs),
						result.ProcessingTime)
				}

				// restoring the state
				if i < b.N-1 {
					for _, userID := range result.DeactivatedUsers {
						userRepo.SetUserActive(ctx, userID, true)
					}

					restoreOriginalReviewers(b, ctx, pool, originalAssignments)
				}
			}

			b.StopTimer()
			cleanupBenchmarkDB(b, pool)
		})
	}
}

func BenchmarkSingleUserDeactivation(b *testing.B) {
	ctx := context.Background()

	cfg, err := config.Load(".env.tests")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := config.MustInitDB(ctx, *cfg)
	require.NoError(b, err)
	defer func() {
		cleanupBenchmarkDB(b, pool)
		pool.Close()
	}()

	cleanupBenchmarkDB(b, pool)
	createBenchmarkAdmin(b, pool)

	userRepo := repository.NewUserRepository(pool)
	teamRepo := repository.NewTeamRepository(pool)

	teamRepo.CreateTeam(ctx, "testteam")
	user := &domain.User{
		UserID:   "testuser",
		Username: "Test User",
		TeamName: "testteam",
		IsActive: true,
	}
	userRepo.CreateOrUpdateUser(ctx, user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := userRepo.SetUserActive(ctx, "testuser", false)
		require.NoError(b, err)

		userRepo.SetUserActive(ctx, "testuser", true)
	}
}

// restoreOriginalReviewers restores the original distribution of reviewers
func restoreOriginalReviewers(b testing.TB, ctx context.Context, pool *pgxpool.Pool, assignments []prReviewers) {
	tx, err := pool.Begin(ctx)
	require.NoError(b, err)
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM pr_reviewers")
	require.NoError(b, err)

	insertQuery := `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
	`

	for _, assignment := range assignments {
		for _, reviewerID := range assignment.reviewers {
			_, err = tx.Exec(ctx, insertQuery, assignment.prID, reviewerID)
			require.NoError(b, err)
		}
	}

	err = tx.Commit(ctx)
	require.NoError(b, err)
}

func cleanupBenchmarkDB(b testing.TB, pool *pgxpool.Pool) {
	ctx := context.Background()
	queries := []string{
		"TRUNCATE TABLE pr_reviewers CASCADE",
		"TRUNCATE TABLE pull_requests CASCADE",
		"TRUNCATE TABLE users CASCADE",
		"TRUNCATE TABLE teams CASCADE",
		"TRUNCATE TABLE auth_tokens CASCADE",
	}

	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		if b != nil {
			require.NoError(b, err)
		}
	}
}

func createBenchmarkAdmin(b testing.TB, pool *pgxpool.Pool) {
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO teams (team_name) VALUES ('admins') ON CONFLICT DO NOTHING")
	if b != nil {
		require.NoError(b, err)
	}

	_, err = pool.Exec(ctx, `
        INSERT INTO users (user_id, username, team_name, is_active)
        VALUES ('badmin', 'Admin', 'admins', true)
        ON CONFLICT DO NOTHING
    `)
	if b != nil {
		require.NoError(b, err)
	}
}
