package model

import "time"

// Booking merepresentasikan data peminjaman ruangan pada tabel bookings.
type Booking struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	RoomID    int64     `json:"room_id" db:"room_id"`
	Purpose   string    `json:"purpose" db:"purpose"`
	StartTime time.Time `json:"start_time" db:"start_time"`
	EndTime   time.Time `json:"end_time" db:"end_time"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// BookingLog merepresentasikan catatan riwayat perubahan status reservasi.
type BookingLog struct {
	ID         int64     `json:"id" db:"id"`
	BookingID  int64     `json:"booking_id" db:"booking_id"`
	ActorID    int64     `json:"actor_id" db:"actor_id"`
	FromStatus *string   `json:"from_status" db:"from_status"`
	ToStatus   string    `json:"to_status" db:"to_status"`
	Note       *string   `json:"note" db:"note"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CreateBookingRequest adalah payload DTO saat student mengajukan reservasi.
type CreateBookingRequest struct {
	UserID    int64     `json:"user_id"`
	RoomID    int64     `json:"room_id"`
	Purpose   string    `json:"purpose"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// UpdateBookingStatusRequest adalah payload DTO saat approver/student mengubah status reservasi.
type UpdateBookingStatusRequest struct {
	ActorID int64   `json:"actor_id"`
	Status  string  `json:"status"`
	Note    *string `json:"note"`
}
