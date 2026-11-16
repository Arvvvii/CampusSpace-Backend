package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
)

// mockRoomRepository adalah mock murni Go tanpa koneksi ke database fisik.
type mockRoomRepository struct {
	createFunc     func(ctx context.Context, room *model.Room) error
	getAllFunc     func(ctx context.Context) ([]model.Room, error)
	getByIDFunc    func(ctx context.Context, id int64) (*model.Room, error)
	updateFunc     func(ctx context.Context, room *model.Room) error
	deactivateFunc func(ctx context.Context, id int64) error
}

func (m *mockRoomRepository) Create(ctx context.Context, room *model.Room) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, room)
	}
	return nil
}

func (m *mockRoomRepository) GetAll(ctx context.Context) ([]model.Room, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}

func (m *mockRoomRepository) GetByID(ctx context.Context, id int64) (*model.Room, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRoomRepository) Update(ctx context.Context, room *model.Room) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, room)
	}
	return nil
}

func (m *mockRoomRepository) Deactivate(ctx context.Context, id int64) error {
	if m.deactivateFunc != nil {
		return m.deactivateFunc(ctx, id)
	}
	return nil
}

func TestCreateRoom_ValidationRules(t *testing.T) {
	mockRepo := &mockRoomRepository{}
	svc := NewRoomService(mockRepo)
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *model.CreateRoomRequest
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: ErrNilRequest,
		},
		{
			name: "empty code",
			req: &model.CreateRoomRequest{
				Code:     "   ",
				Name:     "Lab Komputer",
				Building: "Gedung A",
				Capacity: 30,
			},
			expectedErr: ErrCodeRequired,
		},
		{
			name: "empty name",
			req: &model.CreateRoomRequest{
				Code:     "LAB-01",
				Name:     "",
				Building: "Gedung A",
				Capacity: 30,
			},
			expectedErr: ErrNameRequired,
		},
		{
			name: "empty building",
			req: &model.CreateRoomRequest{
				Code:     "LAB-01",
				Name:     "Lab Komputer",
				Building: "   ",
				Capacity: 30,
			},
			expectedErr: ErrBuildingRequired,
		},
		{
			name: "capacity zero or negative",
			req: &model.CreateRoomRequest{
				Code:     "LAB-01",
				Name:     "Lab Komputer",
				Building: "Gedung A",
				Capacity: 0,
			},
			expectedErr: ErrInvalidCapacity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateRoom(ctx, tt.req)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestCreateRoom_Success(t *testing.T) {
	mockRepo := &mockRoomRepository{
		createFunc: func(ctx context.Context, room *model.Room) error {
			room.ID = 1
			room.CreatedAt = time.Now()
			return nil
		},
	}
	svc := NewRoomService(mockRepo)

	req := &model.CreateRoomRequest{
		Code:     "AULA-01",
		Name:     "Aula Utama",
		Building: "Gedung Rektorat",
		Capacity: 100,
	}

	result, err := svc.CreateRoom(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 1 {
		t.Errorf("expected ID 1, got %d", result.ID)
	}
	if !result.IsActive {
		t.Errorf("expected room to be active")
	}
}

func TestGetRoomByID(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		svc := NewRoomService(&mockRoomRepository{})
		_, err := svc.GetRoomByID(context.Background(), 0)
		if !errors.Is(err, ErrInvalidID) {
			t.Errorf("expected ErrInvalidID, got %v", err)
		}
	})

	t.Run("room not found", func(t *testing.T) {
		mockRepo := &mockRoomRepository{
			getByIDFunc: func(ctx context.Context, id int64) (*model.Room, error) {
				return nil, repository.ErrRoomNotFound
			},
		}
		svc := NewRoomService(mockRepo)
		_, err := svc.GetRoomByID(context.Background(), 99)
		if !errors.Is(err, repository.ErrRoomNotFound) {
			t.Errorf("expected ErrRoomNotFound, got %v", err)
		}
	})
}

func TestDeactivateRoom(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		svc := NewRoomService(&mockRoomRepository{})
		err := svc.DeactivateRoom(context.Background(), -1)
		if !errors.Is(err, ErrInvalidID) {
			t.Errorf("expected ErrInvalidID, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockRoomRepository{
			deactivateFunc: func(ctx context.Context, id int64) error {
				return nil
			},
		}
		svc := NewRoomService(mockRepo)
		err := svc.DeactivateRoom(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
