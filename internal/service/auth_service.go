package service

import (
	"context"
	"fmt"
	"time"

	"pr-reviewer-service/internal/my_errors"

	"pr-reviewer-service/internal/domain"
	"pr-reviewer-service/internal/jwt"
)

type UserRepositoryForAuth interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
}

type AuthService struct {
	repo      AuthRepository
	userRepo  UserRepositoryForAuth
	jwtSecret string
}

func NewAuthService(repo AuthRepository, userRepo UserRepositoryForAuth, jwtSecret string) *AuthService {
	return &AuthService{
		repo:      repo,
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) GenerateToken(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user id: %w", my_errors.ErrEmptyField)
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("%w", my_errors.ErrUserNotFound)
	}

	if !user.IsActive {
		return "", fmt.Errorf("%w", my_errors.ErrUserIsNotActive)
	}

	// check existing token
	existingToken, err := s.repo.GetTokenByUserID(ctx, userID)
	if err == nil && existingToken.ExpiresAt.After(time.Now()) {
		return existingToken.Token, nil
	}

	// generate new token
	token, err := jwt.GenerateToken(userID, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt := time.Now().Add(time.Hour * 24 * 30)
	if err := s.repo.SaveToken(ctx, userID, token, expiresAt); err != nil {
		return "", fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	claims, err := jwt.ParseToken(tokenString, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	userID, err := s.repo.ValidateToken(ctx, tokenString)
	if err != nil {
		return "", fmt.Errorf("token not found in database: %w", err)
	}

	if userID != claims.UserID {
		return "", fmt.Errorf("%w", my_errors.ErrTokenMismatch)
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("%w", my_errors.ErrUserNotFound)
	}

	if !user.IsActive {
		return "", fmt.Errorf("%w", my_errors.ErrUserIsNotActive)
	}

	return userID, nil
}

func (s *AuthService) IsAdmin(ctx context.Context, userID string) (bool, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("%w", my_errors.ErrUserNotFound)
	}

	return user.TeamName == domain.TeamAdmins, nil
}
