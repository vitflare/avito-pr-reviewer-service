package service

import (
	"context"
	"fmt"

	"pr-reviewer-service/internal/domain"
)

type StatisticsService struct {
	repo StatisticsRepository
}

func NewStatisticsService(repo StatisticsRepository) *StatisticsService {
	return &StatisticsService{
		repo: repo,
	}
}

func (s *StatisticsService) GetStatistics(ctx context.Context) (*domain.Statistics, error) {
	stats, err := s.repo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}
