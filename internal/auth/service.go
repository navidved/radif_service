package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/radif/service/internal/config"
	"github.com/radif/service/internal/user"
)

const otpTTL = 2 * time.Minute

// ErrOTPNotFound is returned when no active OTP exists for the phone.
var ErrOTPNotFound = errors.New("OTP not found or expired")

// ErrInvalidOTP is returned when the provided code does not match.
var ErrInvalidOTP = errors.New("invalid OTP code")

// VerifyResult holds the result of a successful OTP verification.
type VerifyResult struct {
	IsNewUser bool
	Token     string
	UserID    string
}

// Service contains the business logic for phone-based authentication.
type Service struct {
	repo    *Repository
	userSvc *user.Service
	cfg     *config.Config
}

// NewService creates a new auth Service.
func NewService(repo *Repository, userSvc *user.Service, cfg *config.Config) *Service {
	return &Service{repo: repo, userSvc: userSvc, cfg: cfg}
}

// SendOTP generates a 5-digit OTP, persists it, and "sends" it (logged in dev).
func (s *Service) SendOTP(ctx context.Context, phone string) error {
	code, err := generateOTP()
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	expiresAt := time.Now().Add(otpTTL)
	if err := s.repo.UpsertOTP(ctx, phone, code, expiresAt); err != nil {
		return fmt.Errorf("store otp: %w", err)
	}

	if !s.cfg.IsProduction() {
		log.Printf("[OTP] phone=%s code=%s", phone, code)
	} else {
		log.Printf("[OTP] sent to phone=%s", phone)
		// TODO: integrate SMS provider (e.g. Kavenegar, SMS.ir)
	}

	return nil
}

// VerifyOTP validates the OTP code and returns user status.
// For existing users it also issues a JWT token immediately.
func (s *Service) VerifyOTP(ctx context.Context, phone, code string) (*VerifyResult, error) {
	activeOTP, err := s.repo.GetActiveOTP(ctx, phone)
	if err != nil {
		return nil, ErrInvalidOTP
	}

	if activeOTP.Code != code {
		return nil, ErrInvalidOTP
	}

	if err := s.repo.MarkOTPUsed(ctx, activeOTP.ID); err != nil {
		return nil, fmt.Errorf("mark otp used: %w", err)
	}

	exists, err := s.repo.UserExists(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("check user existence: %w", err)
	}

	result := &VerifyResult{IsNewUser: !exists}

	if exists {
		u, err := s.userSvc.GetByPhone(ctx, phone)
		if err != nil {
			return nil, fmt.Errorf("get existing user: %w", err)
		}
		token, err := s.issueToken(u.ID, u.Phone, u.AccountType)
		if err != nil {
			return nil, fmt.Errorf("issue token: %w", err)
		}
		result.Token = token
		result.UserID = u.ID
	}

	return result, nil
}

// Register creates a new user account and issues a JWT token.
// If the user already exists (idempotent re-registration), a new token is issued.
func (s *Service) Register(ctx context.Context, phone, accountType string) (string, *user.User, error) {
	// Idempotent: return existing user if already registered.
	existing, err := s.userSvc.GetByPhone(ctx, phone)
	if err == nil {
		token, err := s.issueToken(existing.ID, existing.Phone, existing.AccountType)
		if err != nil {
			return "", nil, fmt.Errorf("issue token for existing user: %w", err)
		}
		return token, existing, nil
	}

	u, err := s.userSvc.Create(ctx, phone, accountType)
	if err != nil {
		return "", nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.issueToken(u.ID, u.Phone, u.AccountType)
	if err != nil {
		return "", nil, fmt.Errorf("issue token: %w", err)
	}

	return token, u, nil
}

// issueToken creates a signed JWT for the given user.
func (s *Service) issueToken(userID, phone, accountType string) (string, error) {
	claims := jwt.MapClaims{
		"sub":         userID,
		"phone":       phone,
		"accountType": accountType,
		"iat":         time.Now().Unix(),
		"exp":         time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// generateOTP generates a cryptographically secure 5-digit code.
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%05d", n.Int64()), nil
}
