package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
)

// mockBookingRepository adalah mock implementasi interface BookingRepository untuk unit test murni Go.
type mockBookingRepository struct {
	createWithLockFunc     func(ctx context.Context, booking *model.Booking, initialLog *model.BookingLog) error
	getByIDFunc            func(ctx context.Context, id int64) (*model.Booking, error)
	getAllFunc             func(ctx context.Context) ([]model.Booking, error)
	getByUserIDFunc        func(ctx context.Context, userID int64) ([]model.Booking, error)
	getByRoomIDFunc        func(ctx context.Context, roomID int64) ([]model.Booking, error)
	checkAvailabilityFunc  func(ctx context.Context, roomID int64, startTime, endTime time.Time, excludeID ...int64) (bool, error)
	updateStatusFunc       func(ctx context.Context, bookingID int64, toStatus string, log *model.BookingLog) error
	getLogsByBookingIDFunc func(ctx context.Context, bookingID int64) ([]model.BookingLog, error)
}

func (m *mockBookingRepository) CreateWithLock(ctx context.Context, booking *model.Booking, initialLog *model.BookingLog) error {
	if m.createWithLockFunc != nil {
		return m.createWithLockFunc(ctx, booking, initialLog)
	}
	return nil
}

func (m *mockBookingRepository) GetByID(ctx context.Context, id int64) (*model.Booking, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockBookingRepository) GetAll(ctx context.Context) ([]model.Booking, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockBookingRepository) GetByUserID(ctx context.Context, userID int64) ([]model.Booking, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockBookingRepository) GetByRoomID(ctx context.Context, roomID int64) ([]model.Booking, error) {
	if m.getByRoomIDFunc != nil {
		return m.getByRoomIDFunc(ctx, roomID)
	}
	return nil, nil
}

func (m *mockBookingRepository) CheckAvailability(ctx context.Context, roomID int64, startTime, endTime time.Time, excludeID ...int64) (bool, error) {
	if m.checkAvailabilityFunc != nil {
		return m.checkAvailabilityFunc(ctx, roomID, startTime, endTime, excludeID...)
	}
	return true, nil
}

func (m *mockBookingRepository) UpdateStatus(ctx context.Context, bookingID int64, toStatus string, log *model.BookingLog) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, bookingID, toStatus, log)
	}
	return nil
}

func (m *mockBookingRepository) GetLogsByBookingID(ctx context.Context, bookingID int64) ([]model.BookingLog, error) {
	if m.getLogsByBookingIDFunc != nil {
		return m.getLogsByBookingIDFunc(ctx, bookingID)
	}
	return nil, nil
}

func TestCreateBooking_ValidationRules(t *testing.T) {
	svc := NewBookingService(&mockBookingRepository{})
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name        string
		req         *model.CreateBookingRequest
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: ErrNilBookingRequest,
		},
		{
			name: "invalid user id",
			req: &model.CreateBookingRequest{
				UserID:    0,
				RoomID:    1,
				Purpose:   "Kuliah Umum",
				StartTime: now,
				EndTime:   now.Add(2 * time.Hour),
			},
			expectedErr: ErrInvalidUserID,
		},
		{
			name: "invalid room id",
			req: &model.CreateBookingRequest{
				UserID:    1,
				RoomID:    0,
				Purpose:   "Kuliah Umum",
				StartTime: now,
				EndTime:   now.Add(2 * time.Hour),
			},
			expectedErr: ErrInvalidRoomID,
		},
		{
			name: "empty purpose",
			req: &model.CreateBookingRequest{
				UserID:    1,
				RoomID:    1,
				Purpose:   "   ",
				StartTime: now,
				EndTime:   now.Add(2 * time.Hour),
			},
			expectedErr: ErrPurposeRequired,
		},
		{
			name: "end time before start time",
			req: &model.CreateBookingRequest{
				UserID:    1,
				RoomID:    1,
				Purpose:   "Kuliah Umum",
				StartTime: now.Add(2 * time.Hour),
				EndTime:   now,
			},
			expectedErr: ErrInvalidTimeRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateBooking(ctx, tt.req)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestCreateBooking_Success(t *testing.T) {
	mockRepo := &mockBookingRepository{
		createWithLockFunc: func(ctx context.Context, booking *model.Booking, initialLog *model.BookingLog) error {
			booking.ID = 10
			booking.CreatedAt = time.Now()
			return nil
		},
	}
	svc := NewBookingService(mockRepo)

	now := time.Now()
	req := &model.CreateBookingRequest{
		UserID:    1,
		RoomID:    2,
		Purpose:   "Seminar Riset",
		StartTime: now,
		EndTime:   now.Add(3 * time.Hour),
	}

	booking, err := svc.CreateBooking(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if booking.ID != 10 {
		t.Errorf("expected booking ID 10, got %d", booking.ID)
	}
	if booking.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", booking.Status)
	}
}

func TestCreateBooking_ConflictError(t *testing.T) {
	mockRepo := &mockBookingRepository{
		createWithLockFunc: func(ctx context.Context, booking *model.Booking, initialLog *model.BookingLog) error {
			return repository.ErrBookingConflict
		},
	}
	svc := NewBookingService(mockRepo)

	now := time.Now()
	req := &model.CreateBookingRequest{
		UserID:    1,
		RoomID:    2,
		Purpose:   "Rapat Himpunan",
		StartTime: now,
		EndTime:   now.Add(time.Hour),
	}

	_, err := svc.CreateBooking(context.Background(), req)
	if !errors.Is(err, repository.ErrBookingConflict) {
		t.Errorf("expected ErrBookingConflict, got %v", err)
	}
}

func TestUpdateBookingStatus_Validation(t *testing.T) {
	svc := NewBookingService(&mockBookingRepository{})
	ctx := context.Background()

	t.Run("invalid booking id", func(t *testing.T) {
		err := svc.UpdateBookingStatus(ctx, 0, &model.UpdateBookingStatusRequest{ActorID: 1, Status: "approved"})
		if !errors.Is(err, ErrInvalidBookingID) {
			t.Errorf("expected ErrInvalidBookingID, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		err := svc.UpdateBookingStatus(ctx, 1, &model.UpdateBookingStatusRequest{ActorID: 1, Status: "unknown_status"})
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("expected ErrInvalidStatus, got %v", err)
		}
	})

	t.Run("actor id missing", func(t *testing.T) {
		err := svc.UpdateBookingStatus(ctx, 1, &model.UpdateBookingStatusRequest{ActorID: 0, Status: "approved"})
		if !errors.Is(err, ErrActorRequired) {
			t.Errorf("expected ErrActorRequired, got %v", err)
		}
	})
}
