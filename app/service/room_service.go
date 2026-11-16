package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
)

var (
	// ErrCodeRequired menandakan kode ruangan kosong atau hanya spasi.
	ErrCodeRequired = errors.New("room code is required and cannot be empty")

	// ErrNameRequired menandakan nama ruangan kosong atau hanya spasi.
	ErrNameRequired = errors.New("room name is required and cannot be empty")

	// ErrBuildingRequired menandakan nama gedung kosong atau hanya spasi.
	ErrBuildingRequired = errors.New("room building is required and cannot be empty")

	// ErrInvalidCapacity menandakan kapasitas ruangan tidak lebih besar dari 0.
	ErrInvalidCapacity = errors.New("room capacity must be greater than 0")

	// ErrInvalidID menandakan ID ruangan tidak valid (<= 0).
	ErrInvalidID = errors.New("invalid room id")

	// ErrNilRequest menandakan request payload bernilai nil.
	ErrNilRequest = errors.New("request payload cannot be nil")
)

// RoomService mendefinisikan kontrak logika bisnis untuk modul ruangan.
type RoomService interface {
	CreateRoom(ctx context.Context, req *model.CreateRoomRequest) (*model.Room, error)
	GetAllRooms(ctx context.Context) ([]model.Room, error)
	GetRoomByID(ctx context.Context, id int64) (*model.Room, error)
	UpdateRoom(ctx context.Context, id int64, req *model.UpdateRoomRequest) (*model.Room, error)
	DeactivateRoom(ctx context.Context, id int64) error
}

type roomService struct {
	repo repository.RoomRepository
}

// NewRoomService membuat instance baru RoomService dengan dependency repository.
func NewRoomService(repo repository.RoomRepository) RoomService {
	return &roomService{repo: repo}
}

// validateCreateInput memvalidasi business rules untuk pembuatan ruangan (fungsi murni Go).
func validateCreateInput(req *model.CreateRoomRequest) error {
	if req == nil {
		return ErrNilRequest
	}
	if strings.TrimSpace(req.Code) == "" {
		return ErrCodeRequired
	}
	if strings.TrimSpace(req.Name) == "" {
		return ErrNameRequired
	}
	if strings.TrimSpace(req.Building) == "" {
		return ErrBuildingRequired
	}
	if req.Capacity <= 0 {
		return ErrInvalidCapacity
	}
	return nil
}

// validateUpdateInput memvalidasi business rules untuk pembaruan ruangan (fungsi murni Go).
func validateUpdateInput(req *model.UpdateRoomRequest) error {
	if req == nil {
		return ErrNilRequest
	}
	if strings.TrimSpace(req.Code) == "" {
		return ErrCodeRequired
	}
	if strings.TrimSpace(req.Name) == "" {
		return ErrNameRequired
	}
	if strings.TrimSpace(req.Building) == "" {
		return ErrBuildingRequired
	}
	if req.Capacity <= 0 {
		return ErrInvalidCapacity
	}
	return nil
}

func (s *roomService) CreateRoom(ctx context.Context, req *model.CreateRoomRequest) (*model.Room, error) {
	if err := validateCreateInput(req); err != nil {
		return nil, err
	}

	room := &model.Room{
		Code:     strings.TrimSpace(req.Code),
		Name:     strings.TrimSpace(req.Name),
		Building: strings.TrimSpace(req.Building),
		Capacity: req.Capacity,
		IsActive: true,
	}

	if err := s.repo.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *roomService) GetAllRooms(ctx context.Context) ([]model.Room, error) {
	return s.repo.GetAll(ctx)
}

func (s *roomService) GetRoomByID(ctx context.Context, id int64) (*model.Room, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *roomService) UpdateRoom(ctx context.Context, id int64, req *model.UpdateRoomRequest) (*model.Room, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}

	if err := validateUpdateInput(req); err != nil {
		return nil, err
	}

	// Pastikan data ruangan ada sebelum di-update
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Code = strings.TrimSpace(req.Code)
	existing.Name = strings.TrimSpace(req.Name)
	existing.Building = strings.TrimSpace(req.Building)
	existing.Capacity = req.Capacity
	existing.IsActive = req.IsActive

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *roomService) DeactivateRoom(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}
	return s.repo.Deactivate(ctx, id)
}
