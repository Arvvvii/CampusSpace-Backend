package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
)

var (
	// ErrInvalidUserID menandakan ID pengguna tidak valid (<= 0).
	ErrInvalidUserID = errors.New("invalid user id")

	// ErrInvalidRoomID menandakan ID ruangan tidak valid (<= 0).
	ErrInvalidRoomID = errors.New("invalid room id")

	// ErrInvalidBookingID menandakan ID booking tidak valid (<= 0).
	ErrInvalidBookingID = errors.New("invalid booking id")

	// ErrPurposeRequired menandakan tujuan peminjaman kosong atau hanya spasi.
	ErrPurposeRequired = errors.New("purpose is required and cannot be empty")

	// ErrInvalidTimeRange menandakan waktu mulai tidak lebih awal dari waktu selesai.
	ErrInvalidTimeRange = errors.New("start time must be before end time")

	// ErrInvalidStatus menandakan status booking yang diberikan tidak valid.
	ErrInvalidStatus = errors.New("invalid booking status (must be pending, approved, rejected, or cancelled)")

	// ErrNilBookingRequest menandakan request payload booking bernilai nil.
	ErrNilBookingRequest = errors.New("booking request payload cannot be nil")

	// ErrActorRequired menandakan ID aktor yang mengubah status tidak valid.
	ErrActorRequired = errors.New("actor id is required")
)

// BookingService mendefinisikan kontrak logika bisnis reservasi ruangan.
type BookingService interface {
	CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error)
	GetBookingByID(ctx context.Context, id int64) (*model.Booking, error)
	GetAllBookings(ctx context.Context) ([]model.Booking, error)
	GetBookingsByUserID(ctx context.Context, userID int64) ([]model.Booking, error)
	GetBookingsByRoomID(ctx context.Context, roomID int64) ([]model.Booking, error)
	CheckRoomAvailability(ctx context.Context, roomID int64, startTime, endTime time.Time) (bool, error)
	UpdateBookingStatus(ctx context.Context, bookingID int64, req *model.UpdateBookingStatusRequest) error
	GetBookingLogs(ctx context.Context, bookingID int64) ([]model.BookingLog, error)
}

type bookingService struct {
	bookingRepo repository.BookingRepository
}

// NewBookingService membuat instance baru BookingService dengan dependency repository.
func NewBookingService(bookingRepo repository.BookingRepository) BookingService {
	return &bookingService{bookingRepo: bookingRepo}
}

// validateCreateBooking memvalidasi aturan bisnis pembuatan reservasi.
func validateCreateBooking(req *model.CreateBookingRequest) error {
	if req == nil {
		return ErrNilBookingRequest
	}
	if req.UserID <= 0 {
		return ErrInvalidUserID
	}
	if req.RoomID <= 0 {
		return ErrInvalidRoomID
	}
	if strings.TrimSpace(req.Purpose) == "" {
		return ErrPurposeRequired
	}
	if !req.StartTime.Before(req.EndTime) {
		return ErrInvalidTimeRange
	}
	return nil
}

// isValidBookingStatus memeriksa apakah status termasuk dalam status yang diizinkan.
func isValidBookingStatus(status string) bool {
	switch status {
	case "pending", "approved", "rejected", "cancelled":
		return true
	default:
		return false
	}
}

func (s *bookingService) CreateBooking(ctx context.Context, req *model.CreateBookingRequest) (*model.Booking, error) {
	if err := validateCreateBooking(req); err != nil {
		return nil, err
	}

	booking := &model.Booking{
		UserID:    req.UserID,
		RoomID:    req.RoomID,
		Purpose:   strings.TrimSpace(req.Purpose),
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Status:    "pending",
	}

	initialLog := &model.BookingLog{
		ActorID:  req.UserID,
		ToStatus: "pending",
		Note:     nil,
	}

	if err := s.bookingRepo.CreateWithLock(ctx, booking, initialLog); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *bookingService) GetBookingByID(ctx context.Context, id int64) (*model.Booking, error) {
	if id <= 0 {
		return nil, ErrInvalidBookingID
	}
	return s.bookingRepo.GetByID(ctx, id)
}

func (s *bookingService) GetAllBookings(ctx context.Context) ([]model.Booking, error) {
	return s.bookingRepo.GetAll(ctx)
}

func (s *bookingService) GetBookingsByUserID(ctx context.Context, userID int64) ([]model.Booking, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	return s.bookingRepo.GetByUserID(ctx, userID)
}

func (s *bookingService) GetBookingsByRoomID(ctx context.Context, roomID int64) ([]model.Booking, error) {
	if roomID <= 0 {
		return nil, ErrInvalidRoomID
	}
	return s.bookingRepo.GetByRoomID(ctx, roomID)
}

func (s *bookingService) CheckRoomAvailability(ctx context.Context, roomID int64, startTime, endTime time.Time) (bool, error) {
	if roomID <= 0 {
		return false, ErrInvalidRoomID
	}
	if !startTime.Before(endTime) {
		return false, ErrInvalidTimeRange
	}
	return s.bookingRepo.CheckAvailability(ctx, roomID, startTime, endTime)
}

func (s *bookingService) UpdateBookingStatus(ctx context.Context, bookingID int64, req *model.UpdateBookingStatusRequest) error {
	if bookingID <= 0 {
		return ErrInvalidBookingID
	}
	if req == nil {
		return ErrNilBookingRequest
	}
	if req.ActorID <= 0 {
		return ErrActorRequired
	}
	if !isValidBookingStatus(req.Status) {
		return ErrInvalidStatus
	}

	logEntry := &model.BookingLog{
		ActorID:  req.ActorID,
		ToStatus: req.Status,
		Note:     req.Note,
	}

	return s.bookingRepo.UpdateStatus(ctx, bookingID, req.Status, logEntry)
}

func (s *bookingService) GetBookingLogs(ctx context.Context, bookingID int64) ([]model.BookingLog, error) {
	if bookingID <= 0 {
		return nil, ErrInvalidBookingID
	}
	return s.bookingRepo.GetLogsByBookingID(ctx, bookingID)
}
