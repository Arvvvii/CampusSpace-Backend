//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Arvvvii/CampusSpace-Backend/app/model"
	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
	"github.com/Arvvvii/CampusSpace-Backend/config"
	"github.com/Arvvvii/CampusSpace-Backend/database"
)

func setupTestDB(t *testing.T) (*sql.DB, repository.RoomRepository) {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Gagal memuat konfigurasi: %v", err)
	}

	// Memungkinkan penentuan database uji terpisah melalui environment variable TEST_DB_NAME
	if testDB := os.Getenv("TEST_DB_NAME"); testDB != "" {
		cfg.Database.Name = testDB
	}

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		t.Fatalf("Gagal terhubung ke database PostgreSQL (%s:%d/%s): %v",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, err)
	}

	repo := repository.NewRoomRepository(db)
	return db, repo
}

func uniqueCode(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// 1. Uji Create Room — data tersimpan, ID digenerate, CreatedAt terisi, dan default is_active
func TestRoomRepository_Create(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	code := uniqueCode("TEST-CR")
	room := &model.Room{
		Code:     code,
		Name:     "Laboratorium Uji Create",
		Building: "Gedung Uji Lt. 1",
		Capacity: 35,
		IsActive: true,
	}

	err := repo.Create(ctx, room)
	if err != nil {
		t.Fatalf("repo.Create gagal: %v", err)
	}

	// Daftarkan pembersihan data spesifik hanya berdasarkan ID yang tercatat
	t.Cleanup(func() {
		if room.ID > 0 {
			_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room.ID)
		}
	})

	if room.ID <= 0 {
		t.Errorf("ID harus > 0, didapat: %d", room.ID)
	}
	if room.CreatedAt.IsZero() {
		t.Errorf("CreatedAt tidak boleh bernilai zero time")
	}
	if !room.IsActive {
		t.Errorf("IsActive harus true")
	}
}

// 2. Uji GetByID — membaca ruangan berdasarkan ID yang tersimpan
func TestRoomRepository_GetByID(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	code := uniqueCode("TEST-GET")
	room := &model.Room{
		Code:     code,
		Name:     "Laboratorium Uji GetByID",
		Building: "Gedung Uji Lt. 2",
		Capacity: 45,
		IsActive: true,
	}

	if err := repo.Create(ctx, room); err != nil {
		t.Fatalf("Persiapan data gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room.ID)
	})

	fetched, err := repo.GetByID(ctx, room.ID)
	if err != nil {
		t.Fatalf("repo.GetByID gagal: %v", err)
	}

	if fetched == nil {
		t.Fatal("Hasil ruangan tidak boleh nil")
	}
	if fetched.ID != room.ID {
		t.Errorf("ID tidak cocok: expected %d, got %d", room.ID, fetched.ID)
	}
	if fetched.Code != room.Code {
		t.Errorf("Code tidak cocok: expected %s, got %s", room.Code, fetched.Code)
	}
	if fetched.Name != room.Name {
		t.Errorf("Name tidak cocok: expected %s, got %s", room.Name, fetched.Name)
	}
	if fetched.Building != room.Building {
		t.Errorf("Building tidak cocok: expected %s, got %s", room.Building, fetched.Building)
	}
	if fetched.Capacity != room.Capacity {
		t.Errorf("Capacity tidak cocok: expected %d, got %d", room.Capacity, fetched.Capacity)
	}
	if fetched.IsActive != room.IsActive {
		t.Errorf("IsActive tidak cocok: expected %v, got %v", room.IsActive, fetched.IsActive)
	}
}

// 3. Uji GetByID NotFound — ID yang tidak ada harus mengembalikan ErrRoomNotFound
func TestRoomRepository_GetByID_NotFound(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	const nonExistentID = 999999999
	fetched, err := repo.GetByID(ctx, nonExistentID)
	if err == nil {
		t.Fatalf("Harus mengembalikan error untuk ID yang tidak ada")
	}
	if !errors.Is(err, repository.ErrRoomNotFound) {
		t.Errorf("Error harus ErrRoomNotFound, didapat: %v", err)
	}
	if fetched != nil {
		t.Errorf("Hasil ruangan harus nil saat not found")
	}
}

// 4. Uji GetAll — data yang baru dibuat harus muncul dalam daftar seluruh ruangan
func TestRoomRepository_GetAll(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	code := uniqueCode("TEST-ALL")
	room := &model.Room{
		Code:     code,
		Name:     "Laboratorium Uji GetAll",
		Building: "Gedung Uji Lt. 3",
		Capacity: 20,
		IsActive: true,
	}

	if err := repo.Create(ctx, room); err != nil {
		t.Fatalf("Persiapan data gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room.ID)
	})

	rooms, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("repo.GetAll gagal: %v", err)
	}

	found := false
	for _, r := range rooms {
		if r.ID == room.ID && r.Code == room.Code {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Ruangan dengan ID %d tidak ditemukan pada hasil GetAll", room.ID)
	}
}

// 5. Uji Update — perubahan atribut tersimpan dengan benar di database
func TestRoomRepository_Update(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	code := uniqueCode("TEST-UPD")
	room := &model.Room{
		Code:     code,
		Name:     "Nama Sebelum Update",
		Building: "Gedung Lama",
		Capacity: 30,
		IsActive: true,
	}

	if err := repo.Create(ctx, room); err != nil {
		t.Fatalf("Persiapan data gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room.ID)
	})

	room.Name = "Nama Setelah Update"
	room.Building = "Gedung Baru"
	room.Capacity = 60

	if err := repo.Update(ctx, room); err != nil {
		t.Fatalf("repo.Update gagal: %v", err)
	}

	updated, err := repo.GetByID(ctx, room.ID)
	if err != nil {
		t.Fatalf("repo.GetByID gagal: %v", err)
	}

	if updated.Name != "Nama Setelah Update" {
		t.Errorf("Nama tidak terbarui: got %s", updated.Name)
	}
	if updated.Building != "Gedung Baru" {
		t.Errorf("Building tidak terbarui: got %s", updated.Building)
	}
	if updated.Capacity != 60 {
		t.Errorf("Capacity tidak terbarui: got %d", updated.Capacity)
	}
}

// 6. Uji Deactivate — soft delete mengubah status is_active menjadi false
func TestRoomRepository_Deactivate(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	code := uniqueCode("TEST-DEACT")
	room := &model.Room{
		Code:     code,
		Name:     "Laboratorium Uji Deactivate",
		Building: "Gedung Uji",
		Capacity: 25,
		IsActive: true,
	}

	if err := repo.Create(ctx, room); err != nil {
		t.Fatalf("Persiapan data gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room.ID)
	})

	if err := repo.Deactivate(ctx, room.ID); err != nil {
		t.Fatalf("repo.Deactivate gagal: %v", err)
	}

	deactivated, err := repo.GetByID(ctx, room.ID)
	if err != nil {
		t.Fatalf("repo.GetByID gagal: %v", err)
	}

	if deactivated.IsActive {
		t.Errorf("IsActive harus bernilai false setelah Deactivate")
	}
}

// 7. Uji DuplicateCode — kode ruangan yang sama harus menghasilkan ErrDuplicateRoomCode
func TestRoomRepository_DuplicateCode(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	code := uniqueCode("TEST-DUP")
	room1 := &model.Room{
		Code:     code,
		Name:     "Ruangan Pertama",
		Building: "Gedung A",
		Capacity: 30,
		IsActive: true,
	}

	if err := repo.Create(ctx, room1); err != nil {
		t.Fatalf("Insert ruangan pertama gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room1.ID)
	})

	room2 := &model.Room{
		Code:     code, // Kode sama persis
		Name:     "Ruangan Kedua Duplikat",
		Building: "Gedung B",
		Capacity: 40,
		IsActive: true,
	}

	err := repo.Create(ctx, room2)
	if err == nil {
		t.Cleanup(func() {
			if room2.ID > 0 {
				_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", room2.ID)
			}
		})
		t.Fatalf("Seharusnya gagal saat memasukkan kode ruangan duplikat")
	}

	if !errors.Is(err, repository.ErrDuplicateRoomCode) {
		t.Errorf("Error harus ErrDuplicateRoomCode, didapat: %v", err)
	}
}

// 8. Uji CapacityConstraint — kapasitas 0 dan negatif harus ditolak oleh CHECK constraint database
func TestRoomRepository_CapacityConstraint(t *testing.T) {
	db, repo := setupTestDB(t)
	ctx := context.Background()

	// Kasus A: Kapasitas 0
	roomZero := &model.Room{
		Code:     uniqueCode("TEST-CAP0"),
		Name:     "Ruang Kapasitas Nol",
		Building: "Gedung C",
		Capacity: 0,
		IsActive: true,
	}

	errZero := repo.Create(ctx, roomZero)
	if errZero == nil {
		t.Cleanup(func() {
			if roomZero.ID > 0 {
				_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", roomZero.ID)
			}
		})
		t.Errorf("Kapasitas 0 harus ditolak oleh basis data")
	}

	// Kasus B: Kapasitas negatif
	roomNeg := &model.Room{
		Code:     uniqueCode("TEST-CAPNEG"),
		Name:     "Ruang Kapasitas Negatif",
		Building: "Gedung D",
		Capacity: -15,
		IsActive: true,
	}

	errNeg := repo.Create(ctx, roomNeg)
	if errNeg == nil {
		t.Cleanup(func() {
			if roomNeg.ID > 0 {
				_, _ = db.Exec("DELETE FROM rooms WHERE id = $1", roomNeg.ID)
			}
		})
		t.Errorf("Kapasitas negatif harus ditolak oleh basis data")
	}
}

// 9. Uji Update & Deactivate pada ID yang tidak ada
func TestRoomRepository_NotFound_Operations(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	const nonExistentID = 999999999

	// Update pada nonExistentID
	errUpdate := repo.Update(ctx, &model.Room{
		ID:       nonExistentID,
		Code:     uniqueCode("TEST-NOUPD"),
		Name:     "Non Existent",
		Building: "Gedung X",
		Capacity: 10,
		IsActive: true,
	})
	if !errors.Is(errUpdate, repository.ErrRoomNotFound) {
		t.Errorf("Update non-existent ID harus menghasilkan ErrRoomNotFound, didapat: %v", errUpdate)
	}

	// Deactivate pada nonExistentID
	errDeact := repo.Deactivate(ctx, nonExistentID)
	if !errors.Is(errDeact, repository.ErrRoomNotFound) {
		t.Errorf("Deactivate non-existent ID harus menghasilkan ErrRoomNotFound, didapat: %v", errDeact)
	}
}
