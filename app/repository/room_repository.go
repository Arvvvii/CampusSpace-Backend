package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
)

var (
	// ErrRoomNotFound dikembalikan ketika ruangan dengan ID yang diminta tidak ditemukan.
	ErrRoomNotFound = errors.New("room not found")

	// ErrDuplicateRoomCode dikembalikan ketika kode ruangan yang dimasukkan sudah digunakan.
	ErrDuplicateRoomCode = errors.New("room code already exists")
)

// sqlStateGetter adalah interface untuk mengekstrak SQLSTATE dari error database tanpa bergantung pada driver tertentu.
type sqlStateGetter interface {
	SQLState() string
}

// mapDBError memetakan error database ke domain error yang sesuai tanpa string matching yang rapuh.
func mapDBError(err error) error {
	if err == nil {
		return nil
	}

	var stateErr sqlStateGetter
	if errors.As(err, &stateErr) && stateErr.SQLState() == "23505" {
		return ErrDuplicateRoomCode
	}

	return err
}

// RoomRepository mendefinisikan kontrak operasi akses data tabel rooms pada PostgreSQL.
type RoomRepository interface {
	Create(ctx context.Context, room *model.Room) error
	GetAll(ctx context.Context) ([]model.Room, error)
	GetByID(ctx context.Context, id int64) (*model.Room, error)
	Update(ctx context.Context, room *model.Room) error
	Deactivate(ctx context.Context, id int64) error
}

type roomRepository struct {
	db *sql.DB
}

// NewRoomRepository membuat instance baru RoomRepository dengan dependency database *sql.DB.
func NewRoomRepository(db *sql.DB) RoomRepository {
	return &roomRepository{db: db}
}

// Create menyimpan ruangan baru ke tabel rooms dan mengisi ID serta CreatedAt yang digenerate basis data.
func (r *roomRepository) Create(ctx context.Context, room *model.Room) error {
	query := `
		INSERT INTO rooms (code, name, building, capacity, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		room.Code,
		room.Name,
		room.Building,
		room.Capacity,
		room.IsActive,
	).Scan(&room.ID, &room.CreatedAt)
	if err != nil {
		return mapDBError(err)
	}

	return nil
}

// GetAll mengambil seluruh daftar ruangan dari tabel rooms diurutkan berdasarkan id ascending.
func (r *roomRepository) GetAll(ctx context.Context) ([]model.Room, error) {
	query := `
		SELECT id, code, name, building, capacity, is_active, created_at
		FROM rooms
		ORDER BY id ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]model.Room, 0)
	for rows.Next() {
		var room model.Room
		err := rows.Scan(
			&room.ID,
			&room.Code,
			&room.Name,
			&room.Building,
			&room.Capacity,
			&room.IsActive,
			&room.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

// GetByID mengambil satu data ruangan berdasarkan primary key id.
func (r *roomRepository) GetByID(ctx context.Context, id int64) (*model.Room, error) {
	query := `
		SELECT id, code, name, building, capacity, is_active, created_at
		FROM rooms
		WHERE id = $1
	`

	var room model.Room
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&room.ID,
		&room.Code,
		&room.Name,
		&room.Building,
		&room.Capacity,
		&room.IsActive,
		&room.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	return &room, nil
}

// Update memperbarui data ruangan (code, name, building, capacity, is_active) berdasarkan id.
func (r *roomRepository) Update(ctx context.Context, room *model.Room) error {
	query := `
		UPDATE rooms
		SET code = $1, name = $2, building = $3, capacity = $4, is_active = $5
		WHERE id = $6
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		room.Code,
		room.Name,
		room.Building,
		room.Capacity,
		room.IsActive,
		room.ID,
	)
	if err != nil {
		return mapDBError(err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRoomNotFound
	}

	return nil
}

// Deactivate menonaktifkan ruangan dengan mengubah is_active menjadi FALSE (soft delete).
func (r *roomRepository) Deactivate(ctx context.Context, id int64) error {
	query := `
		UPDATE rooms
		SET is_active = FALSE
		WHERE id = $1
	`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRoomNotFound
	}

	return nil
}
