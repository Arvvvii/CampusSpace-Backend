package route

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
	"github.com/Arvvvii/CampusSpace-Backend/app/service"
)

// BookingHandler menangani permintaan HTTP terkait reservasi ruangan.
type BookingHandler struct {
	service service.BookingService
}

// NewBookingHandler membuat instance BookingHandler baru.
func NewBookingHandler(svc service.BookingService) *BookingHandler {
	return &BookingHandler{service: svc}
}

// RegisterBookingRoutes mendaftarkan endpoint booking pada router Fiber.
func RegisterBookingRoutes(router fiber.Router, svc service.BookingService) {
	h := NewBookingHandler(svc)
	bookings := router.Group("/bookings")

	bookings.Get("/availability", h.CheckAvailability)
	bookings.Get("/", h.GetAllBookings)
	bookings.Get("/:id", h.GetBookingByID)
	bookings.Get("/:id/logs", h.GetBookingLogs)
	bookings.Post("/", h.CreateBooking)
	bookings.Patch("/:id/status", h.UpdateBookingStatus)
}

// CreateBooking menangani pengajuan reservasi baru (POST /bookings).
func (h *BookingHandler) CreateBooking(c *fiber.Ctx) error {
	var req model.CreateBookingRequest
	if err := c.BodyParser(&req); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "Format JSON tidak valid", nil)
	}

	booking, err := h.service.CreateBooking(c.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUserID) ||
			errors.Is(err, service.ErrInvalidRoomID) ||
			errors.Is(err, service.ErrPurposeRequired) ||
			errors.Is(err, service.ErrInvalidTimeRange) ||
			errors.Is(err, service.ErrNilBookingRequest) {
			return jsonError(c, fiber.StatusUnprocessableEntity, "Validasi pengajuan reservasi gagal", err.Error())
		}
		if errors.Is(err, repository.ErrRoomNotFound) {
			return jsonError(c, fiber.StatusNotFound, "Ruangan tidak ditemukan", nil)
		}
		if errors.Is(err, repository.ErrRoomNotActive) {
			return jsonError(c, fiber.StatusConflict, "Ruangan berstatus nonaktif dan tidak dapat dipesan", nil)
		}
		if errors.Is(err, repository.ErrBookingConflict) {
			return jsonError(c, fiber.StatusConflict, "Jadwal reservasi bertabrakan dengan reservasi lain", nil)
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal membuat reservasi", nil)
	}

	return jsonSuccess(c, fiber.StatusCreated, "Reservasi berhasil diajukan", booking)
}

// CheckAvailability memeriksa apakah ruangan tersedia pada rentang waktu tertentu (GET /bookings/availability).
func (h *BookingHandler) CheckAvailability(c *fiber.Ctx) error {
	roomIDStr := c.Query("room_id")
	startStr := c.Query("start_time")
	endStr := c.Query("end_time")

	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil || roomID <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "Parameter room_id tidak valid", nil)
	}

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "Format start_time harus RFC3339 (contoh: 2026-11-20T09:00:00Z)", nil)
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return jsonError(c, fiber.StatusBadRequest, "Format end_time harus RFC3339 (contoh: 2026-11-20T11:00:00Z)", nil)
	}

	available, err := h.service.CheckRoomAvailability(c.Context(), roomID, startTime, endTime)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTimeRange) {
			return jsonError(c, fiber.StatusUnprocessableEntity, "Validasi rentang waktu gagal", err.Error())
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa ketersediaan ruangan", nil)
	}

	message := "Ruangan tersedia untuk rentang waktu yang diminta"
	if !available {
		message = "Ruangan tidak tersedia (sudah dipesan pada rentang waktu tersebut)"
	}

	return jsonSuccess(c, fiber.StatusOK, message, fiber.Map{
		"room_id":   roomID,
		"available": available,
	})
}

// GetAllBookings mengambil daftar booking dengan filter opsional (GET /bookings?user_id=1 atau ?room_id=2).
func (h *BookingHandler) GetAllBookings(c *fiber.Ctx) error {
	userIDStr := c.Query("user_id")
	roomIDStr := c.Query("room_id")

	if userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err == nil && userID > 0 {
			bookings, err := h.service.GetBookingsByUserID(c.Context(), userID)
			if err != nil {
				return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar reservasi pengguna", nil)
			}
			return jsonSuccess(c, fiber.StatusOK, "Daftar reservasi pengguna berhasil diambil", bookings)
		}
	}

	if roomIDStr != "" {
		roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
		if err == nil && roomID > 0 {
			bookings, err := h.service.GetBookingsByRoomID(c.Context(), roomID)
			if err != nil {
				return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar reservasi ruangan", nil)
			}
			return jsonSuccess(c, fiber.StatusOK, "Daftar reservasi ruangan berhasil diambil", bookings)
		}
	}

	bookings, err := h.service.GetAllBookings(c.Context())
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil seluruh data reservasi", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Daftar seluruh reservasi berhasil diambil", bookings)
}

// GetBookingByID mengambil detail satu booking berdasarkan ID (GET /bookings/:id).
func (h *BookingHandler) GetBookingByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "ID booking tidak valid", nil)
	}

	booking, err := h.service.GetBookingByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrBookingNotFound) {
			return jsonError(c, fiber.StatusNotFound, "Data reservasi tidak ditemukan", nil)
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data reservasi", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Detail reservasi berhasil diambil", booking)
}

// UpdateBookingStatus memperbarui status reservasi (PATCH /bookings/:id/status).
func (h *BookingHandler) UpdateBookingStatus(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "ID booking tidak valid", nil)
	}

	var req model.UpdateBookingStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "Format JSON tidak valid", nil)
	}

	if err := h.service.UpdateBookingStatus(c.Context(), id, &req); err != nil {
		if errors.Is(err, repository.ErrBookingNotFound) {
			return jsonError(c, fiber.StatusNotFound, "Data reservasi tidak ditemukan", nil)
		}
		if errors.Is(err, service.ErrInvalidStatus) ||
			errors.Is(err, service.ErrActorRequired) ||
			errors.Is(err, service.ErrNilBookingRequest) {
			return jsonError(c, fiber.StatusUnprocessableEntity, "Validasi pembaruan status gagal", err.Error())
		}
		return jsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui status reservasi", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Status reservasi berhasil diperbarui", nil)
}

// GetBookingLogs mengambil riwayat kronologis audit log status reservasi (GET /bookings/:id/logs).
func (h *BookingHandler) GetBookingLogs(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return jsonError(c, fiber.StatusBadRequest, "ID booking tidak valid", nil)
	}

	logs, err := h.service.GetBookingLogs(c.Context(), id)
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "Gagal mengambil riwayat audit reservasi", nil)
	}

	return jsonSuccess(c, fiber.StatusOK, "Riwayat audit reservasi berhasil diambil", logs)
}
