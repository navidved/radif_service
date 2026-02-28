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

// UpdateProfile applies partial updates to a user's profile.
func (s *Service) UpdateProfile(ctx context.Context, id string, p UpdateProfileParams) (*User, error) {
	u, err := s.repo.UpdateProfile(ctx, id, p)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return u, nil
}

// UsernameAvailable returns true when the username is not yet taken.
func (s *Service) UsernameAvailable(ctx context.Context, username string) (bool, error) {
	exists, err := s.repo.UsernameExists(ctx, username)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// UpdateAvatarKey saves a new avatar object storage key for the user.
func (s *Service) UpdateAvatarKey(ctx context.Context, id, key string) (*User, error) {
	u, err := s.repo.UpdateAvatarKey(ctx, id, key)
	if err != nil {
		return nil, fmt.Errorf("update avatar key: %w", err)
	}
	return u, nil
}

// IsNotFound returns true when the error indicates a user was not found.
func (s *Service) IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUsernameTaken returns true when the error indicates a username conflict.
func (s *Service) IsUsernameTaken(err error) bool {
	return errors.Is(err, ErrUsernameTaken)
}
