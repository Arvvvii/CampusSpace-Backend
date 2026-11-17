package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
)

var (
	// ErrBookingNotFound dikembalikan ketika data booking tidak ditemukan.
	ErrBookingNotFound = errors.New("booking not found")

	// ErrBookingConflict dikembalikan ketika jadwal reservasi bertabrakan dengan booking lain yang aktif.
	ErrBookingConflict = errors.New("room is already booked for the requested time slot")

	// ErrRoomNotActive dikembalikan ketika ruangan berstatus nonaktif saat mencoba melakukan reservasi.
	ErrRoomNotActive = errors.New("room is not active")

	// ErrInvalidStatusTransition dikembalikan ketika transisi status booking tidak valid atau melanggar aturan status.
	ErrInvalidStatusTransition = errors.New("invalid booking status transition")
)

// BookingRepository mendefinisikan kontrak operasi akses data tabel bookings dan booking_logs pada PostgreSQL.
type BookingRepository interface {
	CreateWithLock(ctx context.Context, booking *model.Booking, initialLog *model.BookingLog) error
	GetByID(ctx context.Context, id int64) (*model.Booking, error)
	GetAll(ctx context.Context) ([]model.Booking, error)
	GetByUserID(ctx context.Context, userID int64) ([]model.Booking, error)
	GetByRoomID(ctx context.Context, roomID int64) ([]model.Booking, error)
	CheckAvailability(ctx context.Context, roomID int64, startTime, endTime time.Time, excludeBookingID ...int64) (bool, error)
	UpdateStatus(ctx context.Context, bookingID int64, toStatus string, log *model.BookingLog) error
	GetLogsByBookingID(ctx context.Context, bookingID int64) ([]model.BookingLog, error)
}

type bookingRepository struct {
	db *sql.DB
}

// NewBookingRepository membuat instance baru BookingRepository dengan dependency *sql.DB.
func NewBookingRepository(db *sql.DB) BookingRepository {
	return &bookingRepository{db: db}
}

// CreateWithLock mengeksekusi transaksi atomic pencegahan double-booking dengan row-level lock pada ruangan.
func (r *bookingRepository) CreateWithLock(ctx context.Context, booking *model.Booking, initialLog *model.BookingLog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Kunci baris ruangan dengan SELECT ... FOR UPDATE untuk mencegah race condition
	var roomID int64
	var isActive bool
	lockRoomQuery := `
		SELECT id, is_active
		FROM rooms
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, lockRoomQuery, booking.RoomID).Scan(&roomID, &isActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoomNotFound
		}
		return err
	}

	if !isActive {
		return ErrRoomNotActive
	}

	// 2. Periksa apakah ada booking berstatus pending atau approved yang bertabrakan waktu
	conflictQuery := `
		SELECT id
		FROM bookings
		WHERE room_id = $1
		  AND status IN ('pending', 'approved')
		  AND start_time < $3
		  AND end_time > $2
		LIMIT 1
	`
	var existingID int64
	err = tx.QueryRowContext(ctx, conflictQuery, booking.RoomID, booking.StartTime, booking.EndTime).Scan(&existingID)
	if err == nil {
		return ErrBookingConflict
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// 3. Simpan booking baru
	if booking.Status == "" {
		booking.Status = "pending"
	}
	insertBookingQuery := `
		INSERT INTO bookings (user_id, room_id, purpose, start_time, end_time, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err = tx.QueryRowContext(
		ctx,
		insertBookingQuery,
		booking.UserID,
		booking.RoomID,
		booking.Purpose,
		booking.StartTime,
		booking.EndTime,
		booking.Status,
	).Scan(&booking.ID, &booking.CreatedAt)
	if err != nil {
		return err
	}

	// 4. Simpan initial log pada tabel booking_logs
	if initialLog != nil {
		initialLog.BookingID = booking.ID
		if initialLog.ToStatus == "" {
			initialLog.ToStatus = booking.Status
		}
		insertLogQuery := `
			INSERT INTO booking_logs (booking_id, actor_id, from_status, to_status, note)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`
		err = tx.QueryRowContext(
			ctx,
			insertLogQuery,
			initialLog.BookingID,
			initialLog.ActorID,
			initialLog.FromStatus,
			initialLog.ToStatus,
			initialLog.Note,
		).Scan(&initialLog.ID, &initialLog.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID mengambil satu data booking berdasarkan ID.
func (r *bookingRepository) GetByID(ctx context.Context, id int64) (*model.Booking, error) {
	query := `
		SELECT id, user_id, room_id, purpose, start_time, end_time, status, created_at
		FROM bookings
		WHERE id = $1
	`

	var booking model.Booking
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.RoomID,
		&booking.Purpose,
		&booking.StartTime,
		&booking.EndTime,
		&booking.Status,
		&booking.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}

	return &booking, nil
}

// GetAll mengambil seluruh daftar booking diurutkan berdasarkan ID ascending.
func (r *bookingRepository) GetAll(ctx context.Context) ([]model.Booking, error) {
	query := `
		SELECT id, user_id, room_id, purpose, start_time, end_time, status, created_at
		FROM bookings
		ORDER BY id ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.RoomID,
			&booking.Purpose,
			&booking.StartTime,
			&booking.EndTime,
			&booking.Status,
			&booking.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookings, nil
}

// GetByUserID mengambil seluruh daftar booking milik pengguna tertentu.
func (r *bookingRepository) GetByUserID(ctx context.Context, userID int64) ([]model.Booking, error) {
	query := `
		SELECT id, user_id, room_id, purpose, start_time, end_time, status, created_at
		FROM bookings
		WHERE user_id = $1
		ORDER BY id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.RoomID,
			&booking.Purpose,
			&booking.StartTime,
			&booking.EndTime,
			&booking.Status,
			&booking.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookings, nil
}

// GetByRoomID mengambil daftar booking untuk suatu ruangan diurutkan berdasarkan start_time ascending.
func (r *bookingRepository) GetByRoomID(ctx context.Context, roomID int64) ([]model.Booking, error) {
	query := `
		SELECT id, user_id, room_id, purpose, start_time, end_time, status, created_at
		FROM bookings
		WHERE room_id = $1
		ORDER BY start_time ASC
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		var booking model.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.RoomID,
			&booking.Purpose,
			&booking.StartTime,
			&booking.EndTime,
			&booking.Status,
			&booking.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookings, nil
}

// CheckAvailability memeriksa apakah suatu ruangan tersedia (tidak ada bentrok jadwal dengan status pending/approved).
func (r *bookingRepository) CheckAvailability(ctx context.Context, roomID int64, startTime, endTime time.Time, excludeBookingID ...int64) (bool, error) {
	var excludeID *int64
	if len(excludeBookingID) > 0 && excludeBookingID[0] > 0 {
		excludeID = &excludeBookingID[0]
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM bookings
			WHERE room_id = $1
			  AND status IN ('pending', 'approved')
			  AND start_time < $3
			  AND end_time > $2
			  AND ($4::bigint IS NULL OR id != $4)
		)
	`

	var conflictExists bool
	err := r.db.QueryRowContext(ctx, query, roomID, startTime, endTime, excludeID).Scan(&conflictExists)
	if err != nil {
		return false, err
	}

	return !conflictExists, nil
}

// UpdateStatus mengubah status booking dan mencatat log audit ke booking_logs secara atomik dalam satu transaksi.
func (r *bookingRepository) UpdateStatus(ctx context.Context, bookingID int64, toStatus string, log *model.BookingLog) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Kunci baris booking dengan SELECT ... FOR UPDATE untuk mencegah race condition (TOCTOU)
	var currentStatus string
	lockBookingQuery := `
		SELECT status
		FROM bookings
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, lockBookingQuery, bookingID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBookingNotFound
		}
		return err
	}

	// 2. Validasi konsistensi status dan proteksi status terminal
	// Status tidak boleh berubah ke status yang sama
	if currentStatus == toStatus {
		return ErrInvalidStatusTransition
	}

	// Status terminal (rejected, cancelled) bersifat final dan tidak boleh diubah kembali
	if currentStatus == "rejected" || currentStatus == "cancelled" {
		return ErrInvalidStatusTransition
	}

	// Evaluasi seluruh matriks transisi resmi
	switch currentStatus {
	case "pending":
		if toStatus != "approved" && toStatus != "rejected" && toStatus != "cancelled" {
			return ErrInvalidStatusTransition
		}
	case "approved":
		if toStatus != "cancelled" {
			return ErrInvalidStatusTransition
		}
	default:
		return ErrInvalidStatusTransition
	}

	// Jika log menyertakan status asal yang diharapkan (optimistic check), verifikasi kesesuaiannya dengan status aktual di basis data
	if log != nil && log.FromStatus != nil && *log.FromStatus != "" && *log.FromStatus != currentStatus {
		return ErrInvalidStatusTransition
	}

	// 3. Perbarui status booking
	updateQuery := `
		UPDATE bookings
		SET status = $1
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, updateQuery, toStatus, bookingID)
	if err != nil {
		return err
	}

	// 4. Catat audit trail ke booking_logs
	if log != nil {
		log.BookingID = bookingID
		log.FromStatus = &currentStatus
		log.ToStatus = toStatus
		insertLogQuery := `
			INSERT INTO booking_logs (booking_id, actor_id, from_status, to_status, note)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`
		err = tx.QueryRowContext(
			ctx,
			insertLogQuery,
			log.BookingID,
			log.ActorID,
			log.FromStatus,
			log.ToStatus,
			log.Note,
		).Scan(&log.ID, &log.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetLogsByBookingID mengambil riwayat kronologis perubahan status booking.
func (r *bookingRepository) GetLogsByBookingID(ctx context.Context, bookingID int64) ([]model.BookingLog, error) {
	query := `
		SELECT id, booking_id, actor_id, from_status, to_status, note, created_at
		FROM booking_logs
		WHERE booking_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]model.BookingLog, 0)
	for rows.Next() {
		var logEntry model.BookingLog
		err := rows.Scan(
			&logEntry.ID,
			&logEntry.BookingID,
			&logEntry.ActorID,
			&logEntry.FromStatus,
			&logEntry.ToStatus,
			&logEntry.Note,
			&logEntry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}
