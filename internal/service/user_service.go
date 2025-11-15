package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pr-reviewer-service/internal/my_errors"

	"pr-reviewer-service/internal/domain"
)

type UserService struct {
	userRepo      UserRepository
	userBatchRepo UserRepositoryForBatch
	prRepo        PRRepositoryForBatch
}

func NewUserService(userRepo UserRepository, userBatchRepo UserRepositoryForBatch, prRepo PRRepositoryForBatch) *UserService {
	return &UserService{
		userRepo:      userRepo,
		userBatchRepo: userBatchRepo,
		prRepo:        prRepo,
	}
}

func (s *UserService) SetUserActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id: %w", my_errors.ErrEmptyField)
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w", my_errors.ErrUserNotFound)
	}

	// Disable deactivation of admins through this
	if user.TeamName == domain.TeamAdmins && !isActive {
		return nil, fmt.Errorf("%w", my_errors.ErrCannotDeactivateAdmin)
	}

	if err := s.userRepo.SetUserActive(ctx, userID, isActive); err != nil {
		return nil, fmt.Errorf("failed to set user active: %w", err)
	}

	user.IsActive = isActive
	return user, nil
}

func (s *UserService) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.userRepo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}

// Batch commands

func (s *UserService) BatchDeactivateTeam(ctx context.Context, teamName string) (*domain.BatchDeactivateResult, error) {
	startTime := time.Now()

	if teamName == "" {
		return nil, fmt.Errorf("team_name: %w", my_errors.ErrEmptyField)
	}

	if teamName == domain.TeamAdmins {
		return nil, fmt.Errorf("%w", my_errors.ErrCannotDeactivateAdminTeam)
	}

	userIDs, err := s.userBatchRepo.GetTeamMemberIDs(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	if len(userIDs) == 0 {
		return &domain.BatchDeactivateResult{
			DeactivatedUsers: []string{},
			ReassignedPRs:    []domain.PRReassignment{},
			SkippedUsers:     []string{},
			ProcessingTime:   time.Since(startTime),
		}, nil
	}

	return s.batchDeactivateUsers(ctx, userIDs, startTime)
}

func (s *UserService) BatchDeactivateUsers(ctx context.Context, userIDs []string) (*domain.BatchDeactivateResult, error) {
	startTime := time.Now()

	if len(userIDs) == 0 {
		return nil, fmt.Errorf("user_id: %w", my_errors.ErrEmptyField)
	}

	return s.batchDeactivateUsers(ctx, userIDs, startTime)
}

func (s *UserService) batchDeactivateUsers(ctx context.Context, userIDs []string, startTime time.Time) (*domain.BatchDeactivateResult, error) {
	result := &domain.BatchDeactivateResult{
		DeactivatedUsers: []string{},
		ReassignedPRs:    []domain.PRReassignment{},
		SkippedUsers:     []string{},
	}

	// get open prs for all users
	prsByReviewer, err := s.prRepo.GetOpenPRsByReviewers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get open PRs: %w", err)
	}

	// deactivate users
	deactivated, err := s.userBatchRepo.BatchDeactivateUsers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate users: %w", err)
	}

	result.DeactivatedUsers = deactivated

	// determine the missing users
	deactivatedMap := make(map[string]bool)
	for _, uid := range deactivated {
		deactivatedMap[uid] = true
	}
	for _, uid := range userIDs {
		if !deactivatedMap[uid] {
			result.SkippedUsers = append(result.SkippedUsers, uid)
		}
	}

	// if there are no open prs, return result
	if len(prsByReviewer) == 0 {
		result.ProcessingTime = time.Since(startTime)
		return result, nil
	}

	// group PRs by unique IDs
	uniquePRs := make(map[string][]string) // map[pr_id]deactivated_reviewers
	for prID, reviewers := range prsByReviewer {
		for _, reviewerID := range reviewers {
			if deactivatedMap[reviewerID] {
				uniquePRs[prID] = append(uniquePRs[prID], reviewerID)
			}
		}
	}

	// find replacements for each pr in separate goroutine
	tasks := make([]domain.ReassignmentTask, 0, len(uniquePRs))

	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(uniquePRs))

	for prID, deactivatedRevs := range uniquePRs {
		wg.Add(1)
		go func(prID string, deactivatedRevs []string) {
			defer wg.Done()

			authorID, teamName, currentReviewers, err := s.prRepo.GetPRWithReviewersAndAuthor(ctx, prID)
			if err != nil {
				errChan <- fmt.Errorf("failed to get PR %s info: %w", prID, err)
				return
			}

			mu.Lock()
			tasks = append(tasks, domain.ReassignmentTask{
				PrID:             prID,
				AuthorID:         authorID,
				DeactivatedRevs:  deactivatedRevs,
				TeamName:         teamName,
				CurrentReviewers: currentReviewers,
			})
			mu.Unlock()
		}(prID, deactivatedRevs)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	// find replacements for each pr
	reassignments := make(map[string]map[string]string) // map[pr_id]map[old_reviewer]new_reviewer

	for _, task := range tasks {
		// get active members except pr author
		activeMembers, err := s.userRepo.GetActiveTeamMembers(ctx, task.TeamName, task.AuthorID)
		if err != nil {
			continue
		}

		currentReviewerSet := make(map[string]bool)
		for _, rev := range task.CurrentReviewers {
			currentReviewerSet[rev] = true
		}

		// filter candidates (except current and deactivated reviewers)
		var candidates []domain.User
		for _, member := range activeMembers {
			if !currentReviewerSet[member.UserID] && !deactivatedMap[member.UserID] {
				candidates = append(candidates, member)
			}
		}

		if len(candidates) == 0 {
			continue // no free candidates
		}

		prReassignments := make(map[string]string)
		usedCandidates := make(map[string]bool)

		deactivatedReviewersForPR := make([]string, 0)
		for _, oldReviewerID := range task.DeactivatedRevs {
			if currentReviewerSet[oldReviewerID] {
				deactivatedReviewersForPR = append(deactivatedReviewersForPR, oldReviewerID)
			}
		}

		// reassign each deactivated reviewer to a unique candidate
		for _, oldReviewerID := range deactivatedReviewersForPR {
			var newReviewer *domain.User

			for i := range candidates {
				candidateID := candidates[i].UserID
				if !usedCandidates[candidateID] && !currentReviewerSet[candidateID] {
					newReviewer = &candidates[i]
					usedCandidates[candidateID] = true
					currentReviewerSet[candidateID] = true
					break
				}
			}

			if newReviewer == nil {
				continue
			}

			prReassignments[oldReviewerID] = newReviewer.UserID
		}

		if len(prReassignments) > 0 {
			reassignments[task.PrID] = prReassignments
		}
	}

	if len(reassignments) > 0 {
		if err := s.prRepo.BatchReassignReviewers(ctx, reassignments); err != nil {
			return nil, fmt.Errorf("failed to batch reassign reviewers: %w", err)
		}

		for prID, reviewerMap := range reassignments {
			oldRevs := make([]string, 0, len(reviewerMap))
			newRevs := make([]string, 0, len(reviewerMap))
			for old, new := range reviewerMap {
				oldRevs = append(oldRevs, old)
				newRevs = append(newRevs, new)
			}

			result.ReassignedPRs = append(result.ReassignedPRs, domain.PRReassignment{
				PullRequestID: prID,
				OldReviewers:  oldRevs,
				NewReviewers:  newRevs,
			})
		}
	}

	result.ProcessingTime = time.Since(startTime)
	return result, nil
}
