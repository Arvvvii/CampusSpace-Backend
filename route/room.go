package route

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
	"github.com/Arvvvii/CampusSpace-Backend/app/service"
)

// responseEnvelope adalah struktur respon standar sesuai CODING_STANDARDS.md.
type responseEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Errors  any    `json:"errors"`
}

func jsonSuccess(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(responseEnvelope{
		Success: true,
		Message: message,
		Data:    data,
		Errors:  nil,
	})
}

func jsonError(c *fiber.Ctx, status int, message string, errDetail any) error {
	return c.Status(status).JSON(responseEnvelope{
		Success: false,
		Message: message,
		Data:    nil,
		Errors:  errDetail,
	})
}

// RoomHandler menangani permintaan HTTP terkait entitas ruangan.
type RoomHandler struct {
	service service.RoomService
}

// NewRoomHandler membuat instance RoomHandler baru.
func NewRoomHandler(svc service.RoomService) *RoomHandler {
	return &RoomHandler{service: svc}
}

// RegisterRoomRoutes mendaftarkan endpoint ruangan pada router Fiber.
func RegisterRoomRoutes(router fiber.Router, svc service.RoomService) {
	h := NewRoomHandler(svc)
	rooms := router.Group("/rooms")

	rooms.Get("/", h.GetAllRooms)
	rooms.Get("/:id", h.GetRoomByID)
	rooms.Post("/", h.CreateRoom)
	rooms.Put("/:id", h.UpdateRoom)
	rooms.Delete("/:id", h.DeactivateRoom)
}

// GetAllRooms mengambil daftar seluruh ruangan (GET /rooms).
func (h *RoomHandler) GetAllRooms(c *fiber.Ctx) error {
	rooms, err := h.service.GetAllRooms(c.Context())
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar ruangan", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Daftar ruangan berhasil diambil", rooms)
}

// GetRoomByID mengambil satu ruangan berdasarkan ID (GET /rooms/:id).
func (h *RoomHandler) GetRoomByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "ID ruangan tidak valid", nil)
	}

	room, err := h.service.GetRoomByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return jsonError(c, fiber.StatusNotFound, "Ruangan tidak ditemukan", nil)
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data ruangan", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Detail ruangan berhasil diambil", room)
}

// CreateRoom menambahkan ruangan baru (POST /rooms).
func (h *RoomHandler) CreateRoom(c *fiber.Ctx) error {
	var req model.CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "Format JSON tidak valid", nil)
	}

	room, err := h.service.CreateRoom(c.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrCodeRequired) ||
			errors.Is(err, service.ErrNameRequired) ||
			errors.Is(err, service.ErrBuildingRequired) ||
			errors.Is(err, service.ErrInvalidCapacity) ||
			errors.Is(err, service.ErrNilRequest) {
			return jsonError(c, fiber.StatusUnprocessableEntity, "Validasi data gagal", err.Error())
		}
		if errors.Is(err, repository.ErrDuplicateRoomCode) {
			return jsonError(c, fiber.StatusConflict, "Kode ruangan sudah digunakan", nil)
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan ruangan", nil)
	}

	return jsonSuccess(c, fiber.StatusCreated, "Ruangan berhasil ditambahkan", room)
}

// UpdateRoom memperbarui data ruangan (PUT /rooms/:id).
func (h *RoomHandler) UpdateRoom(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "ID ruangan tidak valid", nil)
	}

	var req model.UpdateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "Format JSON tidak valid", nil)
	}

	room, err := h.service.UpdateRoom(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return jsonError(c, fiber.StatusNotFound, "Ruangan tidak ditemukan", nil)
		}
		if errors.Is(err, repository.ErrDuplicateRoomCode) {
			return jsonError(c, fiber.StatusConflict, "Kode ruangan sudah digunakan oleh ruangan lain", nil)
		}
		if errors.Is(err, service.ErrCodeRequired) ||
			errors.Is(err, service.ErrNameRequired) ||
			errors.Is(err, service.ErrBuildingRequired) ||
			errors.Is(err, service.ErrInvalidCapacity) ||
			errors.Is(err, service.ErrNilRequest) {
			return jsonError(c, fiber.StatusUnprocessableEntity, "Validasi data gagal", err.Error())
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui ruangan", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Ruangan berhasil diperbarui", room)
}

// DeactivateRoom menonaktifkan ruangan / soft delete (DELETE /rooms/:id).
func (h *RoomHandler) DeactivateRoom(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "ID ruangan tidak valid", nil)
	}

	if err := h.service.DeactivateRoom(c.Context(), id); err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return jsonError(c, fiber.StatusNotFound, "Ruangan tidak ditemukan", nil)
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal menonaktifkan ruangan", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Ruangan berhasil dinonaktifkan", nil)
}
