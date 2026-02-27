package user

import (
	"context"
	"errors"
	"fmt"
)

// Service contains business logic for user management.
type Service struct {
	repo *Repository
}

// NewService creates a new user Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new user account.
func (s *Service) Create(ctx context.Context, phone, accountType string) (*User, error) {
	u, err := s.repo.Create(ctx, phone, accountType)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetByID returns a user by their UUID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByPhone returns a user by their phone number.
func (s *Service) GetByPhone(ctx context.Context, phone string) (*User, error) {
	return s.repo.GetByPhone(ctx, phone)
}

// IsNotFound returns true when the error indicates a user was not found.
func (s *Service) IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
