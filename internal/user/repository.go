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
	ID            string  `json:"id"`
	Phone         string  `json:"phone"`
	AccountType   string  `json:"accountType"`
	Username      *string `json:"username,omitempty"`
	FullName      *string `json:"fullName,omitempty"`
	Bio           *string `json:"bio,omitempty"`
	BusinessPhone *string `json:"businessPhone,omitempty"`
	Address       *string `json:"address,omitempty"`
	AvatarKey     *string `json:"-"`
	AvatarURL     *string `json:"avatarUrl,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UpdateProfileParams holds the fields that can be updated via PATCH /users/me.
// Nil pointers mean "leave unchanged".
type UpdateProfileParams struct {
	Username      *string
	FullName      *string
	Bio           *string
	BusinessPhone *string
	Address       *string
}

// ErrNotFound is returned when a user does not exist.
var ErrNotFound = errors.New("user not found")

// ErrAlreadyExists is returned when a phone number is already registered.
var ErrAlreadyExists = errors.New("user already exists")

// ErrUsernameTaken is returned when the chosen username is already in use.
var ErrUsernameTaken = errors.New("username already taken")

// Repository handles all user database operations.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository with the given connection pool.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// scanUser scans a full user row into a User value.
func scanUser(row pgx.Row, u *User) error {
	return row.Scan(
		&u.ID, &u.Phone, &u.AccountType,
		&u.Username, &u.FullName, &u.Bio,
		&u.BusinessPhone, &u.Address, &u.AvatarKey,
		&u.CreatedAt, &u.UpdatedAt,
	)
}

const selectCols = `id, phone, account_type, username, full_name, bio, business_phone, address, avatar_key, created_at, updated_at`

// Create inserts a new user and returns the created record.
func (r *Repository) Create(ctx context.Context, phone, accountType string) (*User, error) {
	u := &User{}
	err := scanUser(r.db.QueryRow(ctx,
		`INSERT INTO users (phone, account_type)
		 VALUES ($1, $2)
		 RETURNING `+selectCols,
		phone, accountType,
	), u)
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
	err := scanUser(r.db.QueryRow(ctx,
		`SELECT `+selectCols+` FROM users WHERE id = $1`, id,
	), u)
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
	err := scanUser(r.db.QueryRow(ctx,
		`SELECT `+selectCols+` FROM users WHERE phone = $1`, phone,
	), u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by phone: %w", err)
	}
	return u, nil
}

// UpdateProfile applies partial profile updates. Nil fields are left unchanged.
func (r *Repository) UpdateProfile(ctx context.Context, id string, p UpdateProfileParams) (*User, error) {
	u := &User{}
	err := scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET
		    username       = COALESCE($2, username),
		    full_name      = COALESCE($3, full_name),
		    bio            = COALESCE($4, bio),
		    business_phone = COALESCE($5, business_phone),
		    address        = COALESCE($6, address)
		 WHERE id = $1
		 RETURNING `+selectCols,
		id, p.Username, p.FullName, p.Bio, p.BusinessPhone, p.Address,
	), u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return u, nil
}

// UsernameExists returns true when the username is already taken by any user.
func (r *Repository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, username,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}
	return exists, nil
}

// UpdateAvatarKey saves a new avatar object key for the user and returns the updated record.
func (r *Repository) UpdateAvatarKey(ctx context.Context, id, key string) (*User, error) {
	u := &User{}
	err := scanUser(r.db.QueryRow(ctx,
		`UPDATE users SET avatar_key = $2 WHERE id = $1 RETURNING `+selectCols,
		id, key,
	), u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update avatar key: %w", err)
	}
	return u, nil
}

// isUniqueViolation checks whether an error is a PostgreSQL unique_violation (code 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
