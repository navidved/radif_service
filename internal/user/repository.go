// Package user manages user accounts and their persistence.
package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a registered Radif user.
type User struct {
	ID          string    `json:"id"`
	Phone       string    `json:"phone"`
	AccountType string    `json:"accountType"`
	FullName    *string   `json:"fullName,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ErrNotFound is returned when a user does not exist.
var ErrNotFound = errors.New("user not found")

// ErrAlreadyExists is returned when a phone number is already registered.
var ErrAlreadyExists = errors.New("user already exists")

// Repository handles all user database operations.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository with the given connection pool.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new user and returns the created record.
func (r *Repository) Create(ctx context.Context, phone, accountType string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (phone, account_type)
		 VALUES ($1, $2)
		 RETURNING id, phone, account_type, full_name, created_at, updated_at`,
		phone, accountType,
	).Scan(&u.ID, &u.Phone, &u.AccountType, &u.FullName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetByID fetches a user by their UUID.
func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, phone, account_type, full_name, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Phone, &u.AccountType, &u.FullName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// GetByPhone fetches a user by their phone number.
func (r *Repository) GetByPhone(ctx context.Context, phone string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, phone, account_type, full_name, created_at, updated_at
		 FROM users WHERE phone = $1`,
		phone,
	).Scan(&u.ID, &u.Phone, &u.AccountType, &u.FullName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by phone: %w", err)
	}
	return u, nil
}

// isUniqueViolation checks whether an error is a PostgreSQL unique_violation (code 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
