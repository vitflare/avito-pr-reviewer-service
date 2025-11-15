package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"pr-reviewer-service/internal/request"
	"pr-reviewer-service/internal/response"
	"pr-reviewer-service/pkg/config"

	"pr-reviewer-service/internal/handler"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/router"
	"pr-reviewer-service/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type E2ETestSuite struct {
	pool   *pgxpool.Pool
	server *httptest.Server
	token  string
}

func setupE2ETest(t *testing.T) *E2ETestSuite {
	ctx := context.Background()

	cfg, err := config.Load(".env.tests")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := config.MustInitDB(ctx, *cfg)
	require.NoError(t, err)

	cleanupDB(t, pool)

	authRepo := repository.NewAuthRepository(pool)
	teamRepo := repository.NewTeamRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	prRepo := repository.NewPRRepository(pool)
	statsRepo := repository.NewStatisticsRepository(pool)

	validate := validator.New()

	authService := service.NewAuthService(authRepo, userRepo, cfg.JWTSecret)
	teamService := service.NewTeamService(teamRepo, userRepo)
	userService := service.NewUserService(userRepo, userRepo, prRepo)
	prService := service.NewPRService(prRepo, userRepo)
	statsService := service.NewStatisticsService(statsRepo)

	authHandler := handler.NewAuthHandler(authService, validate)
	teamHandler := handler.NewTeamHandler(teamService, validate)
	userHandler := handler.NewUserHandler(userService, prService, validate)
	prHandler := handler.NewPRHandler(prService, validate)
	healthHandler := handler.NewHealthHandler()
	statisticsHandler := handler.NewStatisticsHandler(statsService)

	r := router.SetupRouter(
		authHandler,
		teamHandler,
		userHandler,
		prHandler,
		healthHandler,
		statisticsHandler,
		authService,
	)

	server := httptest.NewServer(r)

	createAdmin(t, pool)

	token := getAdminToken(t, server.URL)

	return &E2ETestSuite{
		pool:   pool,
		server: server,
		token:  token,
	}
}

func (s *E2ETestSuite) teardown() {
	cleanupDB(nil, s.pool)
	s.server.Close()
	s.pool.Close()
}

func cleanupDB(t testing.TB, pool *pgxpool.Pool) {
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
		if t != nil {
			require.NoError(t, err)
		}
	}
}

func createAdmin(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO teams (team_name) VALUES ('admins')")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
        INSERT INTO users (user_id, username, team_name, is_active)
        VALUES ('admin', 'Admin', 'admins', true)
    `)
	require.NoError(t, err)
}

func getAdminToken(t *testing.T, baseURL string) string {
	reqBody := request.LoginRequest{UserID: "admin"}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	var loginResp response.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(t, err)

	return loginResp.Token
}

func TestE2E_CompleteWorkflow(t *testing.T) {
	suite := setupE2ETest(t)
	defer suite.teardown()

	t.Run("1. Create team with members", func(t *testing.T) {
		reqBody := request.CreateTeamRequest{
			TeamName: "backend",
			Members: []request.TeamMemberInput{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
				{UserID: "u4", Username: "Diana", IsActive: true},
			},
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", suite.server.URL+"/team/add", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+suite.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("2. Get user token", func(t *testing.T) {
		reqBody := request.LoginRequest{UserID: "u1"}
		body, _ := json.Marshal(reqBody)

		resp, err := http.Post(suite.server.URL+"/auth/login", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("3. Create multiple PRs", func(t *testing.T) {
		prs := []request.CreatePRRequest{
			{PullRequestID: "pr-1", PullRequestName: "Feature A", AuthorID: "u1"},
			{PullRequestID: "pr-2", PullRequestName: "Feature B", AuthorID: "u2"},
			{PullRequestID: "pr-3", PullRequestName: "Feature C", AuthorID: "u1"},
			{PullRequestID: "pr-4", PullRequestName: "Feature D", AuthorID: "u3"},
		}

		for _, pr := range prs {
			body, _ := json.Marshal(pr)
			req, _ := http.NewRequest("POST", suite.server.URL+"/pullRequest/create", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+suite.token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}
	})

	t.Run("4. Get statistics", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/statistics", nil)
		req.Header.Set("Authorization", "Bearer "+suite.token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var stats response.StatisticsResponse
		err = json.NewDecoder(resp.Body).Decode(&stats)
		require.NoError(t, err)

		assert.Equal(t, 4, stats.TotalPRs)
		assert.Equal(t, 4, stats.OpenPRs)
		assert.True(t, len(stats.UserAssignments) > 0)
	})

	t.Run("5. Batch deactivate team", func(t *testing.T) {
		reqBody := request.BatchDeactivateTeamRequest{
			TeamName: "backend",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", suite.server.URL+"/users/batchDeactivateTeam", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+suite.token)
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		duration := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var batchResp response.BatchDeactivateResponse
		err = json.NewDecoder(resp.Body).Decode(&batchResp)
		require.NoError(t, err)

		t.Logf("Batch deactivation completed in %v ms", duration.Milliseconds())
		t.Logf("Deactivated users: %d", batchResp.TotalDeactivated)
		t.Logf("Reassigned PRs: %d", batchResp.TotalPRsReassigned)
		t.Logf("Processing time (internal): %d ms", batchResp.ProcessingTimeMs)

		assert.True(t, duration.Milliseconds() < 200, "Batch operation should complete under 200ms")
		assert.True(t, batchResp.ProcessingTimeMs < 100, "Internal processing should be under 100ms")
	})
}

func TestE2E_BatchDeactivateUsers(t *testing.T) {
	suite := setupE2ETest(t)
	defer suite.teardown()

	reqBody := request.CreateTeamRequest{
		TeamName: "frontend",
		Members: []request.TeamMemberInput{
			{UserID: "f1", Username: "Frank", IsActive: true},
			{UserID: "f2", Username: "Grace", IsActive: true},
			{UserID: "f3", Username: "Henry", IsActive: true},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", suite.server.URL+"/team/add", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	prReq := request.CreatePRRequest{
		PullRequestID:   "pr-10",
		PullRequestName: "Frontend Feature",
		AuthorID:        "f1",
	}
	body, _ = json.Marshal(prReq)
	req, _ = http.NewRequest("POST", suite.server.URL+"/pullRequest/create", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()

	// deactivating specific users
	deactivateReq := request.BatchDeactivateUsersRequest{
		UserIDs: []string{"f2", "f3"},
	}

	body, _ = json.Marshal(deactivateReq)
	req, _ = http.NewRequest("POST", suite.server.URL+"/users/batchDeactivateUsers", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var batchResp response.BatchDeactivateResponse
	err = json.NewDecoder(resp.Body).Decode(&batchResp)
	require.NoError(t, err)

	t.Logf("Batch deactivation completed in %v ms", duration.Milliseconds())
	t.Logf("Processing time (internal): %d ms", batchResp.ProcessingTimeMs)

	assert.True(t, duration.Milliseconds() < 150)
	// check deactivation count
	assert.Equal(t, 2, batchResp.TotalDeactivated)
	// check, that no one was assined on PR because of team of three members (there is no another reviewer)
	assert.True(t, batchResp.TotalPRsReassigned == 0)
}
