package service

import (
	"context"
	"time"

	"pr-reviewer-service/internal/domain"
)

type AuthRepository interface {
	SaveToken(ctx context.Context, userID, token string, expiresAt time.Time) error
	GetTokenByUserID(ctx context.Context, userID string) (*domain.AuthToken, error)
	ValidateToken(ctx context.Context, token string) (string, error)
}

type PRRepository interface {
	CreatePR(ctx context.Context, pr *domain.PullRequest) error
	PRExists(ctx context.Context, prID string) (bool, error)
	GetPRByID(ctx context.Context, prID string) (*domain.PullRequest, error)
	MergePR(ctx context.Context, prID string) error
	IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) error
	GetPRsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
}

type UserRepositoryForPR interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error)
}

type StatisticsRepository interface {
	GetStatistics(ctx context.Context) (*domain.Statistics, error)
}

type TeamRepository interface {
	CreateTeam(ctx context.Context, teamName string) error
	TeamExists(ctx context.Context, teamName string) (bool, error)
	GetTeamWithMembers(ctx context.Context, teamName string) (*domain.Team, error)
	GetAllTeams(ctx context.Context) ([]domain.Team, error)
}

type UserRepository interface {
	CreateOrUpdateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
	GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
}

type PRRepositoryForBatch interface {
	GetOpenPRsByReviewers(ctx context.Context, userIDs []string) (map[string][]string, error)
	BatchReassignReviewers(ctx context.Context, reassignments map[string]map[string]string) error
	GetPRWithReviewersAndAuthor(ctx context.Context, prID string) (string, string, []string, error)
}

type UserRepositoryForBatch interface {
	BatchDeactivateUsers(ctx context.Context, userIDs []string) ([]string, error)
	GetTeamMemberIDs(ctx context.Context, teamName string) ([]string, error)
}
