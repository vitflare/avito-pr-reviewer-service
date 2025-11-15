package service

import (
	"context"
	"fmt"

	"pr-reviewer-service/internal/my_errors"

	"pr-reviewer-service/internal/domain"
)

type TeamService struct {
	teamRepo TeamRepository
	userRepo UserRepository
}

func NewTeamService(teamRepo TeamRepository, userRepo UserRepository) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error) {
	if team.TeamName == "" {
		return nil, fmt.Errorf("team_name: %w", my_errors.ErrEmptyField)
	}

	if len(team.Members) == 0 {
		return nil, fmt.Errorf("team must have at least one member: %w", my_errors.ErrInvalidInput)
	}

	// Checking for duplicate user_id within a command
	userIDs := make(map[string]bool)
	for _, member := range team.Members {
		if member.UserID == "" {
			return nil, fmt.Errorf("user_id: %w", my_errors.ErrEmptyField)
		}
		if member.Username == "" {
			return nil, fmt.Errorf("username: %w", my_errors.ErrInvalidInput)
		}
		if userIDs[member.UserID] {
			return nil, fmt.Errorf("duplicate user_id in team members: %s", member.UserID)
		}
		userIDs[member.UserID] = true
	}

	exists, err := s.teamRepo.TeamExists(ctx, team.TeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to check team existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w", my_errors.ErrTeamAlreadyExists)
	}

	if err := s.teamRepo.CreateTeam(ctx, team.TeamName); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	for _, member := range team.Members {
		user := &domain.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := s.userRepo.CreateOrUpdateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create/update user: %w", err)
		}
	}

	createdTeam, err := s.teamRepo.GetTeamWithMembers(ctx, team.TeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get created team: %w", err)
	}

	return createdTeam, nil
}

func (s *TeamService) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	if teamName == "" {
		return nil, fmt.Errorf("team_name: %w", my_errors.ErrEmptyField)
	}

	team, err := s.teamRepo.GetTeamWithMembers(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("%w", my_errors.ErrTeamNotFound)
	}

	return team, nil
}

func (s *TeamService) GetAllTeams(ctx context.Context) ([]domain.Team, error) {
	teams, err := s.teamRepo.GetAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all teams: %w", err)
	}
	return teams, nil
}
