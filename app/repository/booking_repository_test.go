//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
	"github.com/Arvvvii/CampusSpace-Backend/config"
	"github.com/Arvvvii/CampusSpace-Backend/database"
)

type bookingTestContext struct {
	db          *sql.DB
	bookingRepo repository.BookingRepository
	roomRepo    repository.RoomRepository
	userRepo    repository.UserRepository
}

func setupBookingTestDB(t *testing.T) *bookingTestContext {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Gagal memuat konfigurasi: %v", err)
	}

	if testDB := os.Getenv("TEST_DB_NAME"); testDB != "" {
		cfg.Database.Name = testDB
	} else {
		cfg.Database.Name = "campus_space_test"
	}

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		t.Fatalf("Gagal terhubung ke database PostgreSQL (%s:%d/%s): %v",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, err)
	}

	return &bookingTestContext{
		db:          db,
		bookingRepo: repository.NewBookingRepository(db),
		roomRepo:    repository.NewRoomRepository(db),
		userRepo:    repository.NewUserRepository(db),
	}
}

func createTestUserHelper(t *testing.T, tc *bookingTestContext, role string) *model.User {
	t.Helper()
	ctx := context.Background()

	user := &model.User{
		Name:         "User Test " + role,
		Email:        fmt.Sprintf("user_%s_%d@test.campus.ac.id", role, time.Now().UnixNano()),
		PasswordHash: "hashpassword",
		Role:         role,
	}

	err := tc.userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Gagal membuat user uji: %v", err)
	}

	t.Cleanup(func() {
		if user.ID > 0 {
			_, _ = tc.db.Exec("DELETE FROM users WHERE id = $1", user.ID)
		}
	})

	return user
}

func createTestRoomHelper(t *testing.T, tc *bookingTestContext, isActive bool) *model.Room {
	t.Helper()
	ctx := context.Background()

	room := &model.Room{
		Code:     fmt.Sprintf("RM-%d", time.Now().UnixNano()),
		Name:     "Ruang Uji Booking",
		Building: "Gedung Uji Lt. 2",
		Capacity: 50,
		IsActive: isActive,
	}

	err := tc.roomRepo.Create(ctx, room)
	if err != nil {
		t.Fatalf("Gagal membuat ruangan uji: %v", err)
	}

	t.Cleanup(func() {
		if room.ID > 0 {
			_, _ = tc.db.Exec("DELETE FROM rooms WHERE id = $1", room.ID)
		}
	})

	return room
}

// 1. Uji CreateWithLock berhasil dan menghasilkan initial booking log
func TestBookingRepository_CreateWithLock(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	startTime := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	endTime := startTime.Add(2 * time.Hour)

	booking := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Kegiatan Uji CreateWithLock",
		StartTime: startTime,
		EndTime:   endTime,
		Status:    "pending",
	}

	initialLog := &model.BookingLog{
		ActorID: user.ID,
		Note:    nil,
	}

	err := tc.bookingRepo.CreateWithLock(ctx, booking, initialLog)
	if err != nil {
		t.Fatalf("CreateWithLock gagal: %v", err)
	}

	t.Cleanup(func() {
		if booking.ID > 0 {
			_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
			_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
		}
	})

	if booking.ID <= 0 {
		t.Errorf("ID booking harus > 0, didapat: %d", booking.ID)
	}
	if initialLog.ID <= 0 {
		t.Errorf("ID initial log harus > 0, didapat: %d", initialLog.ID)
	}
	if initialLog.BookingID != booking.ID {
		t.Errorf("BookingID pada initial log harus %d, didapat %d", booking.ID, initialLog.BookingID)
	}
	if initialLog.ToStatus != "pending" {
		t.Errorf("ToStatus pada initial log harus 'pending', didapat %q", initialLog.ToStatus)
	}
	if initialLog.FromStatus != nil {
		t.Errorf("FromStatus pada initial log harus nil, didapat %v", initialLog.FromStatus)
	}
}

// 2. Uji GetByID berhasil dan penanganan Not Found
func TestBookingRepository_GetByID(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	startTime := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	endTime := startTime.Add(3 * time.Hour)

	booking := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Uji GetByID",
		StartTime: startTime,
		EndTime:   endTime,
		Status:    "pending",
	}

	err := tc.bookingRepo.CreateWithLock(ctx, booking, nil)
	if err != nil {
		t.Fatalf("CreateWithLock gagal: %v", err)
	}

	t.Cleanup(func() {
		if booking.ID > 0 {
			_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
			_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
		}
	})

	found, err := tc.bookingRepo.GetByID(ctx, booking.ID)
	if err != nil {
		t.Fatalf("GetByID gagal: %v", err)
	}

	if found.ID != booking.ID {
		t.Errorf("ID tidak cocok: ekspektasi %d, didapat %d", booking.ID, found.ID)
	}
	if found.Purpose != booking.Purpose {
		t.Errorf("Purpose tidak cocok: ekspektasi %q, didapat %q", booking.Purpose, found.Purpose)
	}

	// Uji NotFound
	nonExistent, err := tc.bookingRepo.GetByID(ctx, 99999999)
	if !errors.Is(err, repository.ErrBookingNotFound) {
		t.Fatalf("Ekspektasi ErrBookingNotFound, didapat: %v", err)
	}
	if nonExistent != nil {
		t.Fatalf("Ekspektasi nil saat not found, didapat: %+v", nonExistent)
	}
}

// 3. Uji GetAll, GetByUserID, dan GetByRoomID
func TestBookingRepository_ListQueries(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user1 := createTestUserHelper(t, tc, "student")
	user2 := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	start1 := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	end1 := start1.Add(2 * time.Hour)

	start2 := end1.Add(1 * time.Hour)
	end2 := start2.Add(2 * time.Hour)

	b1 := &model.Booking{
		UserID:    user1.ID,
		RoomID:    room.ID,
		Purpose:   "Booking User 1",
		StartTime: start1,
		EndTime:   end1,
		Status:    "pending",
	}
	b2 := &model.Booking{
		UserID:    user2.ID,
		RoomID:    room.ID,
		Purpose:   "Booking User 2",
		StartTime: start2,
		EndTime:   end2,
		Status:    "pending",
	}

	if err := tc.bookingRepo.CreateWithLock(ctx, b1, nil); err != nil {
		t.Fatalf("Create b1 gagal: %v", err)
	}
	if err := tc.bookingRepo.CreateWithLock(ctx, b2, nil); err != nil {
		t.Fatalf("Create b2 gagal: %v", err)
	}

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id IN ($1, $2)", b1.ID, b2.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id IN ($1, $2)", b1.ID, b2.ID)
	})

	// GetAll
	all, err := tc.bookingRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll gagal: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("GetAll harus mengembalikan minimal 2 booking, didapat: %d", len(all))
	}

	// GetByUserID
	user1Bookings, err := tc.bookingRepo.GetByUserID(ctx, user1.ID)
	if err != nil {
		t.Fatalf("GetByUserID gagal: %v", err)
	}
	if len(user1Bookings) != 1 || user1Bookings[0].ID != b1.ID {
		t.Errorf("GetByUserID tidak mengembalikan booking milik user1")
	}

	// GetByRoomID
	roomBookings, err := tc.bookingRepo.GetByRoomID(ctx, room.ID)
	if err != nil {
		t.Fatalf("GetByRoomID gagal: %v", err)
	}
	if len(roomBookings) != 2 {
		t.Errorf("GetByRoomID harus mengembalikan 2 booking, didapat: %d", len(roomBookings))
	}
}

// 4. Uji CheckAvailability — slot tersedia dan tidak tersedia
func TestBookingRepository_CheckAvailability(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(100 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	// Sebelum booking dibuat, slot harus tersedia
	avail, err := tc.bookingRepo.CheckAvailability(ctx, room.ID, start, end)
	if err != nil {
		t.Fatalf("CheckAvailability gagal: %v", err)
	}
	if !avail {
		t.Error("Ekspektasi slot tersedia sebelum booking dibuat")
	}

	// Buat booking pada slot tersebut
	booking := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Uji Availability",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}
	if err := tc.bookingRepo.CreateWithLock(ctx, booking, nil); err != nil {
		t.Fatalf("Create booking gagal: %v", err)
	}

	t.Cleanup(func() {
		if booking.ID > 0 {
			_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
			_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
		}
	})

	// Setelah booking pending dibuat, slot harus menjadi tidak tersedia
	avail, err = tc.bookingRepo.CheckAvailability(ctx, room.ID, start, end)
	if err != nil {
		t.Fatalf("CheckAvailability kedua gagal: %v", err)
	}
	if avail {
		t.Error("Ekspektasi slot tidak tersedia setelah booking pending tersimpan")
	}

	// Pengecualian ID sendiri (misal saat validasi update) harus mengembalikan true
	avail, err = tc.bookingRepo.CheckAvailability(ctx, room.ID, start, end, booking.ID)
	if err != nil {
		t.Fatalf("CheckAvailability dengan exclude ID gagal: %v", err)
	}
	if !avail {
		t.Error("Ekspektasi slot tersedia jika mengecualikan ID booking sendiri")
	}
}

// 5. Uji Batas Waktu Bersinggungan (Adjacent Slots) — 09:00-10:00 dan 10:00-11:00 diperbolehkan
func TestBookingRepository_AdjacentSlotsAllowed(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	t1 := time.Now().Add(120 * time.Hour).Truncate(time.Second)
	t2 := t1.Add(1 * time.Hour) // t1 - t2 (09:00 - 10:00)
	t3 := t2.Add(1 * time.Hour) // t2 - t3 (10:00 - 11:00)

	b1 := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Slot Pertama",
		StartTime: t1,
		EndTime:   t2,
		Status:    "approved",
	}
	if err := tc.bookingRepo.CreateWithLock(ctx, b1, nil); err != nil {
		t.Fatalf("Booking slot pertama gagal: %v", err)
	}

	b2 := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Slot Kedua Bersinggungan",
		StartTime: t2, // Mulai tepat saat slot pertama selesai
		EndTime:   t3,
		Status:    "pending",
	}

	err := tc.bookingRepo.CreateWithLock(ctx, b2, nil)
	if err != nil {
		t.Fatalf("Slot bersinggungan seharusnya diizinkan, tetapi ditolak: %v", err)
	}

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id IN ($1, $2)", b1.ID, b2.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id IN ($1, $2)", b1.ID, b2.ID)
	})
}

// 6. Uji Overlap Ditolak — ErrBookingConflict
func TestBookingRepository_OverlapConflict(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	t1 := time.Now().Add(140 * time.Hour).Truncate(time.Second)
	t2 := t1.Add(2 * time.Hour) // 10:00 - 12:00

	b1 := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Booking Pertama",
		StartTime: t1,
		EndTime:   t2,
		Status:    "pending",
	}
	if err := tc.bookingRepo.CreateWithLock(ctx, b1, nil); err != nil {
		t.Fatalf("Create b1 gagal: %v", err)
	}

	t.Cleanup(func() {
		if b1.ID > 0 {
			_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", b1.ID)
			_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", b1.ID)
		}
	})

	// Coba buat booking yang bertabrakan di tengah: t1 + 30m s/d t2 + 30m
	bOverlap := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Booking Tabrakan",
		StartTime: t1.Add(30 * time.Minute),
		EndTime:   t2.Add(30 * time.Minute),
		Status:    "pending",
	}

	err := tc.bookingRepo.CreateWithLock(ctx, bOverlap, nil)
	if !errors.Is(err, repository.ErrBookingConflict) {
		t.Fatalf("Ekspektasi ErrBookingConflict, didapat: %v", err)
	}
}

// 7. Uji Ruangan Non-Aktif Ditolak — ErrRoomNotActive
func TestBookingRepository_InactiveRoomRejected(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	inactiveRoom := createTestRoomHelper(t, tc, false) // is_active = FALSE

	start := time.Now().Add(160 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	booking := &model.Booking{
		UserID:    user.ID,
		RoomID:    inactiveRoom.ID,
		Purpose:   "Coba Booking Ruang Nonaktif",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}

	err := tc.bookingRepo.CreateWithLock(ctx, booking, nil)
	if !errors.Is(err, repository.ErrRoomNotActive) {
		t.Fatalf("Ekspektasi ErrRoomNotActive, didapat: %v", err)
	}
}

// 8. Uji UpdateStatus Sah: pending -> approved, pending -> rejected, pending -> cancelled
func TestBookingRepository_UpdateStatus_PendingTransitions(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	approver := createTestUserHelper(t, tc, "approver")
	student := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	transitions := []string{"approved", "rejected", "cancelled"}

	for i, targetStatus := range transitions {
		t.Run("pending_to_"+targetStatus, func(t *testing.T) {
			start := time.Now().Add(time.Duration(180+i*10) * time.Hour).Truncate(time.Second)
			end := start.Add(2 * time.Hour)

			booking := &model.Booking{
				UserID:    student.ID,
				RoomID:    room.ID,
				Purpose:   "Uji transisi " + targetStatus,
				StartTime: start,
				EndTime:   end,
				Status:    "pending",
			}
			if err := tc.bookingRepo.CreateWithLock(ctx, booking, nil); err != nil {
				t.Fatalf("Create booking gagal: %v", err)
			}

			t.Cleanup(func() {
				_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
				_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
			})

			note := "Catatan untuk " + targetStatus
			logEntry := &model.BookingLog{
				ActorID: approver.ID,
				Note:    &note,
			}

			err := tc.bookingRepo.UpdateStatus(ctx, booking.ID, targetStatus, logEntry)
			if err != nil {
				t.Fatalf("UpdateStatus ke %q gagal: %v", targetStatus, err)
			}

			// Verifikasi status booking diperbarui
			updated, err := tc.bookingRepo.GetByID(ctx, booking.ID)
			if err != nil {
				t.Fatalf("GetByID gagal: %v", err)
			}
			if updated.Status != targetStatus {
				t.Errorf("Status aktual %q tidak cocok dengan ekspektasi %q", updated.Status, targetStatus)
			}
		})
	}
}

// 9. Uji UpdateStatus: approved -> cancelled diperbolehkan
func TestBookingRepository_UpdateStatus_ApprovedToCancelled(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	approver := createTestUserHelper(t, tc, "approver")
	student := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(220 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	booking := &model.Booking{
		UserID:    student.ID,
		RoomID:    room.ID,
		Purpose:   "Uji approved to cancelled",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}
	if err := tc.bookingRepo.CreateWithLock(ctx, booking, nil); err != nil {
		t.Fatalf("Create booking gagal: %v", err)
	}

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
	})

	// Ubah pending -> approved
	approveNote := "Disetujui oleh approver"
	approveLog := &model.BookingLog{
		ActorID: approver.ID,
		Note:    &approveNote,
	}
	if err := tc.bookingRepo.UpdateStatus(ctx, booking.ID, "approved", approveLog); err != nil {
		t.Fatalf("Approve booking gagal: %v", err)
	}

	// Ubah approved -> cancelled
	cancelNote := "Dibatalkan oleh mahasiswa"
	cancelLog := &model.BookingLog{
		ActorID: student.ID,
		Note:    &cancelNote,
	}

	err := tc.bookingRepo.UpdateStatus(ctx, booking.ID, "cancelled", cancelLog)
	if err != nil {
		t.Fatalf("approved to cancelled seharusnya diizinkan, tetapi gagal: %v", err)
	}

	updated, _ := tc.bookingRepo.GetByID(ctx, booking.ID)
	if updated.Status != "cancelled" {
		t.Errorf("Ekspektasi status 'cancelled', didapat: %q", updated.Status)
	}
}

// 10. Uji Transisi Status Ilegal Ditolak — ErrInvalidStatusTransition
func TestBookingRepository_UpdateStatus_InvalidTransitions(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	student := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(240 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	booking := &model.Booking{
		UserID:    student.ID,
		RoomID:    room.ID,
		Purpose:   "Uji invalid transitions",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}
	if err := tc.bookingRepo.CreateWithLock(ctx, booking, nil); err != nil {
		t.Fatalf("Create booking gagal: %v", err)
	}

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
	})

	// 10a. pending -> pending (status sama ditolak)
	err := tc.bookingRepo.UpdateStatus(ctx, booking.ID, "pending", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition untuk pending->pending, didapat: %v", err)
	}

	// 10b. pending -> target status tidak valid (misal: "completed")
	err = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "completed", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition untuk pending->completed, didapat: %v", err)
	}

	// Ubah ke approved
	if err := tc.bookingRepo.UpdateStatus(ctx, booking.ID, "approved", nil); err != nil {
		t.Fatalf("Approve gagal: %v", err)
	}

	// 10c. approved -> approved (status sama ditolak)
	err = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "approved", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition untuk approved->approved, didapat: %v", err)
	}

	// 10d. approved -> rejected (dilarang)
	err = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "rejected", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition untuk approved->rejected, didapat: %v", err)
	}

	// 10e. approved -> pending (dilarang)
	err = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "pending", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition untuk approved->pending, didapat: %v", err)
	}
}

// 11. Uji Status Terminal Tidak Dapat Diaktifkan Kembali (rejected / cancelled -> anything)
func TestBookingRepository_TerminalStatesImmutable(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	student := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	// Kasus Rejected
	start1 := time.Now().Add(260 * time.Hour).Truncate(time.Second)
	end1 := start1.Add(2 * time.Hour)

	bRejected := &model.Booking{
		UserID:    student.ID,
		RoomID:    room.ID,
		Purpose:   "Booking Rejected",
		StartTime: start1,
		EndTime:   end1,
		Status:    "pending",
	}
	_ = tc.bookingRepo.CreateWithLock(ctx, bRejected, nil)
	_ = tc.bookingRepo.UpdateStatus(ctx, bRejected.ID, "rejected", nil)

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", bRejected.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", bRejected.ID)
	})

	// Coba ubah rejected -> approved
	err := tc.bookingRepo.UpdateStatus(ctx, bRejected.ID, "approved", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition saat rejected->approved, didapat: %v", err)
	}

	// Coba ubah rejected -> pending
	err = tc.bookingRepo.UpdateStatus(ctx, bRejected.ID, "pending", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition saat rejected->pending, didapat: %v", err)
	}

	// Kasus Cancelled
	start2 := end1.Add(2 * time.Hour)
	end2 := start2.Add(2 * time.Hour)

	bCancelled := &model.Booking{
		UserID:    student.ID,
		RoomID:    room.ID,
		Purpose:   "Booking Cancelled",
		StartTime: start2,
		EndTime:   end2,
		Status:    "pending",
	}
	_ = tc.bookingRepo.CreateWithLock(ctx, bCancelled, nil)
	_ = tc.bookingRepo.UpdateStatus(ctx, bCancelled.ID, "cancelled", nil)

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", bCancelled.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", bCancelled.ID)
	})

	// Coba ubah cancelled -> approved
	err = tc.bookingRepo.UpdateStatus(ctx, bCancelled.ID, "approved", nil)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Errorf("Ekspektasi ErrInvalidStatusTransition saat cancelled->approved, didapat: %v", err)
	}
}

// 12. Uji GetLogsByBookingID — riwayat audit kronologis
func TestBookingRepository_GetLogsByBookingID(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	student := createTestUserHelper(t, tc, "student")
	approver := createTestUserHelper(t, tc, "approver")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(280 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	booking := &model.Booking{
		UserID:    student.ID,
		RoomID:    room.ID,
		Purpose:   "Uji Kronologi Log",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}

	initLog := &model.BookingLog{
		ActorID: student.ID,
	}

	if err := tc.bookingRepo.CreateWithLock(ctx, booking, initLog); err != nil {
		t.Fatalf("Create booking gagal: %v", err)
	}

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
	})

	// Transisi 1: pending -> approved
	noteApprove := "Disetujui untuk kegiatan akademik"
	_ = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "approved", &model.BookingLog{
		ActorID: approver.ID,
		Note:    &noteApprove,
	})

	// Transisi 2: approved -> cancelled
	noteCancel := "Dibatalkan mendadak"
	_ = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "cancelled", &model.BookingLog{
		ActorID: student.ID,
		Note:    &noteCancel,
	})

	logs, err := tc.bookingRepo.GetLogsByBookingID(ctx, booking.ID)
	if err != nil {
		t.Fatalf("GetLogsByBookingID gagal: %v", err)
	}

	if len(logs) != 3 {
		t.Fatalf("Ekspektasi 3 entri audit log, didapat: %d", len(logs))
	}

	// Verifikasi urutan log 1: initial pending
	if logs[0].ToStatus != "pending" || logs[0].FromStatus != nil {
		t.Errorf("Log 0 tidak valid: from=%v, to=%s", logs[0].FromStatus, logs[0].ToStatus)
	}

	// Verifikasi urutan log 2: pending -> approved
	if logs[1].ToStatus != "approved" || logs[1].FromStatus == nil || *logs[1].FromStatus != "pending" {
		t.Errorf("Log 1 tidak valid: from=%v, to=%s", logs[1].FromStatus, logs[1].ToStatus)
	}

	// Verifikasi urutan log 3: approved -> cancelled
	if logs[2].ToStatus != "cancelled" || logs[2].FromStatus == nil || *logs[2].FromStatus != "approved" {
		t.Errorf("Log 2 tidak valid: from=%v, to=%s", logs[2].FromStatus, logs[2].ToStatus)
	}
}

// 13. Uji Concurrency CreateWithLock — dua goroutine mencoba memesan jadwal yang sama persis
func TestBookingRepository_Concurrency_DoubleBookingPrevention(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user1 := createTestUserHelper(t, tc, "student")
	user2 := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(300 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	var wg sync.WaitGroup
	var successCount int32
	var conflictCount int32
	var mu sync.Mutex

	var createdBookingIDs []int64

	runBooking := func(user *model.User) {
		defer wg.Done()

		b := &model.Booking{
			UserID:    user.ID,
			RoomID:    room.ID,
			Purpose:   "Booking Concurrent",
			StartTime: start,
			EndTime:   end,
			Status:    "pending",
		}

		err := tc.bookingRepo.CreateWithLock(ctx, b, nil)
		mu.Lock()
		defer mu.Unlock()

		if err == nil {
			successCount++
			createdBookingIDs = append(createdBookingIDs, b.ID)
		} else if errors.Is(err, repository.ErrBookingConflict) {
			conflictCount++
		}
	}

	wg.Add(2)
	go runBooking(user1)
	go runBooking(user2)
	wg.Wait()

	t.Cleanup(func() {
		for _, id := range createdBookingIDs {
			_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", id)
			_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", id)
		}
	})

	if successCount != 1 {
		t.Errorf("Ekspektasi tepat 1 booking yang berhasil, didapat: %d", successCount)
	}
	if conflictCount != 1 {
		t.Errorf("Ekspektasi tepat 1 booking ditolak konflik, didapat: %d", conflictCount)
	}
}

// 14. Uji Concurrency UpdateStatus — verifikasi status asal yang diharapkan (optimistic check)
func TestBookingRepository_UpdateStatus_ExpectedFromStatus(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	student := createTestUserHelper(t, tc, "student")
	approver := createTestUserHelper(t, tc, "approver")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(320 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	booking := &model.Booking{
		UserID:    student.ID,
		RoomID:    room.ID,
		Purpose:   "Uji Expected FromStatus",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}
	_ = tc.bookingRepo.CreateWithLock(ctx, booking, nil)

	t.Cleanup(func() {
		_, _ = tc.db.Exec("DELETE FROM booking_logs WHERE booking_id = $1", booking.ID)
		_, _ = tc.db.Exec("DELETE FROM bookings WHERE id = $1", booking.ID)
	})

	// Transaksi lain mendahului membatalkan booking (status menjadi 'cancelled')
	_ = tc.bookingRepo.UpdateStatus(ctx, booking.ID, "cancelled", nil)

	// Approver mencoba approve dengan menyertakan ekspektasi FromStatus = 'pending'
	expectedFrom := "pending"
	logApprover := &model.BookingLog{
		ActorID:    approver.ID,
		FromStatus: &expectedFrom,
	}

	err := tc.bookingRepo.UpdateStatus(ctx, booking.ID, "approved", logApprover)
	if !errors.Is(err, repository.ErrInvalidStatusTransition) {
		t.Fatalf("Ekspektasi ErrInvalidStatusTransition saat status aktual di DB telah berubah, didapat: %v", err)
	}
}

// 15. Uji Rollback Atomik Transaksi
func TestBookingRepository_AtomicRollback(t *testing.T) {
	tc := setupBookingTestDB(t)
	ctx := context.Background()

	user := createTestUserHelper(t, tc, "student")
	room := createTestRoomHelper(t, tc, true)

	start := time.Now().Add(340 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	// Skenario: Initial log memiliki foreign key actor_id yang TIDAK ADA (99999999)
	booking := &model.Booking{
		UserID:    user.ID,
		RoomID:    room.ID,
		Purpose:   "Uji Rollback",
		StartTime: start,
		EndTime:   end,
		Status:    "pending",
	}

	invalidLog := &model.BookingLog{
		ActorID: 99999999, // User actor tidak terdaftar -> FK violation di DB
	}

	err := tc.bookingRepo.CreateWithLock(ctx, booking, invalidLog)
	if err == nil {
		t.Fatal("Ekspektasi error FK violation pada insert initial log, tetapi sukses")
	}

	// Pastikan row booking TIDAK pernah tersimpan karena transaksi di-rollback secara atomik
	var count int
	_ = tc.db.QueryRowContext(ctx, "SELECT count(*) FROM bookings WHERE room_id = $1 AND start_time = $2",
		room.ID, start).Scan(&count)

	if count != 0 {
		t.Errorf("Ekspektasi row booking di-rollback (count = 0), didapat: %d", count)
	}
}
