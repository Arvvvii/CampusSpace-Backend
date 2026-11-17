package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
)

var (
	// ErrUserNotFound dikembalikan ketika user dengan ID atau email yang diminta tidak ditemukan.
	ErrUserNotFound = errors.New("user not found")

	// ErrDuplicateEmail dikembalikan ketika alamat email sudah terdaftar di database.
	ErrDuplicateEmail = errors.New("email already exists")
)

// UserRepository mendefinisikan kontrak operasi akses data tabel users pada PostgreSQL.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type userRepository struct {
	db *sql.DB
}

// NewUserRepository membuat instance baru UserRepository dengan dependency *sql.DB.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// Create menyimpan pengguna baru ke tabel users dan mengisi ID serta CreatedAt yang digenerate database.
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var stateErr sqlStateGetter
		if errors.As(err, &stateErr) && stateErr.SQLState() == "23505" {
			return ErrDuplicateEmail
		}
		return err
	}

	return nil
}

// GetByID mengambil satu data pengguna berdasarkan ID.
func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, created_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// GetByEmail mengambil data pengguna berdasarkan alamat email unik.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, created_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
