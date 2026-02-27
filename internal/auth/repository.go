// Package auth handles OTP-based phone authentication.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// otp is the internal representation of a one-time password record.
type otp struct {
	ID        string
	Phone     string
	Code      string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// Repository handles OTP persistence.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new auth Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// UpsertOTP invalidates all active OTPs for the phone and inserts a fresh one.
func (r *Repository) UpsertOTP(ctx context.Context, phone, code string, expiresAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`UPDATE otps SET used_at = NOW()
		 WHERE phone = $1 AND used_at IS NULL`,
		phone,
	)
	if err != nil {
		return fmt.Errorf("invalidate old otps: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO otps (phone, code, expires_at) VALUES ($1, $2, $3)`,
		phone, code, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert otp: %w", err)
	}

	return tx.Commit(ctx)
}

// GetActiveOTP returns the most recent unused, non-expired OTP for the phone.
func (r *Repository) GetActiveOTP(ctx context.Context, phone string) (*otp, error) {
	o := &otp{}
	err := r.db.QueryRow(ctx,
		`SELECT id, phone, code, expires_at, used_at, created_at
		 FROM otps
		 WHERE phone = $1 AND used_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC
		 LIMIT 1`,
		phone,
	).Scan(&o.ID, &o.Phone, &o.Code, &o.ExpiresAt, &o.UsedAt, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOTPNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active otp: %w", err)
	}
	return o, nil
}

// MarkOTPUsed marks the OTP record as consumed.
func (r *Repository) MarkOTPUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE otps SET used_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

// UserExists returns true if a user with the given phone already exists.
func (r *Repository) UserExists(ctx context.Context, phone string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`,
		phone,
	).Scan(&exists)
	return exists, err
}
