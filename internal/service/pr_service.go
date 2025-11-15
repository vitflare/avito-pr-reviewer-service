package service

import (
	"context"
	"fmt"
	"math/rand"

	"pr-reviewer-service/internal/my_errors"

	"pr-reviewer-service/internal/domain"
)

type PRService struct {
	prRepo   PRRepository
	userRepo UserRepositoryForPR
}

func NewPRService(prRepo PRRepository, userRepo UserRepositoryForPR) *PRService {
	return &PRService{
		prRepo:   prRepo,
		userRepo: userRepo,
	}
}

func (s *PRService) CreatePR(ctx context.Context, pr *domain.PullRequest) (*domain.PullRequest, error) {
	if pr.PullRequestID == "" {
		return nil, fmt.Errorf("pull_request_id: %w", my_errors.ErrEmptyField)
	}
	if pr.PullRequestName == "" {
		return nil, fmt.Errorf("pull_request_name: %w", my_errors.ErrEmptyField)
	}
	if pr.AuthorID == "" {
		return nil, fmt.Errorf("author_id: %w", my_errors.ErrEmptyField)
	}

	exists, err := s.prRepo.PRExists(ctx, pr.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to check PR existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w", my_errors.ErrPRAlreadyExists)
	}

	author, err := s.userRepo.GetUserByID(ctx, pr.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("%w", my_errors.ErrAuthorNotFound)
	}

	if !author.IsActive {
		return nil, fmt.Errorf("author is not active")
	}

	activeMembers, err := s.userRepo.GetActiveTeamMembers(ctx, author.TeamName, author.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active team members: %w", err)
	}

	reviewers := selectRandomReviewers(activeMembers, 2)
	pr.AssignedReviewers = reviewers
	pr.Status = domain.StatusOpen

	if err := s.prRepo.CreatePR(ctx, pr); err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	createdPR, err := s.prRepo.GetPRByID(ctx, pr.PullRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created PR: %w", err)
	}

	return createdPR, nil
}

func (s *PRService) MergePR(ctx context.Context, prID string) (*domain.PullRequest, error) {
	if prID == "" {
		return nil, fmt.Errorf("pull_request_id: %w", my_errors.ErrEmptyField)
	}

	pr, err := s.prRepo.GetPRByID(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("%w", my_errors.ErrPRNotFound)
	}

	// Idempotency: if already merged, return the current state
	if pr.Status == domain.StatusMerged {
		return pr, nil
	}

	if err := s.prRepo.MergePR(ctx, prID); err != nil {
		return nil, fmt.Errorf("failed to merge PR: %w", err)
	}

	mergedPR, err := s.prRepo.GetPRByID(ctx, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merged PR: %w", err)
	}

	return mergedPR, nil
}

func (s *PRService) ReassignReviewer(ctx context.Context, prID, oldUserID string) (string, *domain.PullRequest, error) {
	if prID == "" {
		return "", nil, fmt.Errorf("pull_request_id: %w", my_errors.ErrEmptyField)
	}
	if oldUserID == "" {
		return "", nil, fmt.Errorf("old_user_id: %w", my_errors.ErrEmptyField)
	}

	pr, err := s.prRepo.GetPRByID(ctx, prID)
	if err != nil {
		return "", nil, fmt.Errorf("%w", my_errors.ErrPRNotFound)
	}

	if pr.Status == domain.StatusMerged {
		return "", nil, fmt.Errorf("%w", my_errors.ErrPRAlreadyMerged)
	}

	isAssigned, err := s.prRepo.IsReviewerAssigned(ctx, prID, oldUserID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to check reviewer assignment: %w", err)
	}
	if !isAssigned {
		return "", nil, fmt.Errorf("%w", my_errors.ErrReviewerIsNotAssigned)
	}

	oldUser, err := s.userRepo.GetUserByID(ctx, oldUserID)
	if err != nil {
		return "", nil, fmt.Errorf("%w", my_errors.ErrUserNotFound)
	}

	activeMembers, err := s.userRepo.GetActiveTeamMembers(ctx, oldUser.TeamName, oldUserID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get active team members: %w", err)
	}

	// Excluding current reviewers
	availableCandidates := []domain.User{}
	for _, member := range activeMembers {
		isCurrentReviewer := false
		for _, reviewerID := range pr.AssignedReviewers {
			if member.UserID == reviewerID {
				isCurrentReviewer = true
				break
			}
		}
		if !isCurrentReviewer {
			availableCandidates = append(availableCandidates, member)
		}
	}

	if len(availableCandidates) == 0 {
		return "", nil, fmt.Errorf("%w", my_errors.ErrNoActiveReviewerWasFound)
	}

	newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]

	if err := s.prRepo.ReassignReviewer(ctx, prID, oldUserID, newReviewer.UserID); err != nil {
		return "", nil, fmt.Errorf("failed to reassign reviewer: %w", err)
	}

	updatedPR, err := s.prRepo.GetPRByID(ctx, prID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get updated PR: %w", err)
	}

	return newReviewer.UserID, updatedPR, nil
}

func (s *PRService) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id: %w", my_errors.ErrEmptyField)
	}

	prs, err := s.prRepo.GetPRsByReviewer(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reviews: %w", err)
	}

	return prs, nil
}

func selectRandomReviewers(users []domain.User, count int) []string {
	if len(users) <= count {
		result := make([]string, len(users))
		for i, u := range users {
			result[i] = u.UserID
		}
		return result
	}

	selected := make(map[int]bool)
	result := []string{}

	for len(result) < count {
		idx := rand.Intn(len(users))
		if !selected[idx] {
			selected[idx] = true
			result = append(result, users[idx].UserID)
		}
	}

	return result
}
