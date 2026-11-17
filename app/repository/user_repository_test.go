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

func setupUserTestDB(t *testing.T) (*sql.DB, repository.UserRepository) {
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

	repo := repository.NewUserRepository(db)
	return db, repo
}

func uniqueUserEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@test.campus.ac.id", prefix, time.Now().UnixNano())
}

// 1. Uji Create User — data tersimpan, ID digenerate, CreatedAt terisi
func TestUserRepository_Create(t *testing.T) {
	db, repo := setupUserTestDB(t)
	ctx := context.Background()

	email := uniqueUserEmail("budi")
	user := &model.User{
		Name:         "Budi Prasetyo",
		Email:        email,
		PasswordHash: "$2a$10$abcdef1234567890",
		Role:         "student",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("repo.Create gagal: %v", err)
	}

	t.Cleanup(func() {
		if user.ID > 0 {
			_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
		}
	})

	if user.ID <= 0 {
		t.Errorf("ID user harus lebih besar dari 0, didapat: %d", user.ID)
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt user tidak boleh bernilai zero time")
	}
}

// 2. Uji GetByID User — data yang diambil sesuai dengan yang disimpan
func TestUserRepository_GetByID(t *testing.T) {
	db, repo := setupUserTestDB(t)
	ctx := context.Background()

	email := uniqueUserEmail("siti")
	user := &model.User{
		Name:         "Siti Aminah",
		Email:        email,
		PasswordHash: "$2a$10$passwordhashapprover",
		Role:         "approver",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("repo.Create gagal: %v", err)
	}

	t.Cleanup(func() {
		if user.ID > 0 {
			_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
		}
	})

	found, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("repo.GetByID gagal: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("ID tidak cocok: ekspektasi %d, didapat %d", user.ID, found.ID)
	}
	if found.Name != user.Name {
		t.Errorf("Name tidak cocok: ekspektasi %q, didapat %q", user.Name, found.Name)
	}
	if found.Email != user.Email {
		t.Errorf("Email tidak cocok: ekspektasi %q, didapat %q", user.Email, found.Email)
	}
	if found.PasswordHash != user.PasswordHash {
		t.Errorf("PasswordHash tidak cocok: ekspektasi %q, didapat %q", user.PasswordHash, found.PasswordHash)
	}
	if found.Role != user.Role {
		t.Errorf("Role tidak cocok: ekspektasi %q, didapat %q", user.Role, found.Role)
	}
}

// 3. Uji GetByID User NotFound — ID tidak terdaftar mengembalikan ErrUserNotFound
func TestUserRepository_GetByID_NotFound(t *testing.T) {
	_, repo := setupUserTestDB(t)
	ctx := context.Background()

	found, err := repo.GetByID(ctx, 99999999)
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("Ekspektasi ErrUserNotFound, didapat error: %v", err)
	}
	if found != nil {
		t.Fatalf("Ekspektasi user nil saat not found, didapat: %+v", found)
	}
}

// 4. Uji GetByEmail User — data yang diambil berdasarkan email sesuai
func TestUserRepository_GetByEmail(t *testing.T) {
	db, repo := setupUserTestDB(t)
	ctx := context.Background()

	email := uniqueUserEmail("admin")
	user := &model.User{
		Name:         "Admin Kampus",
		Email:        email,
		PasswordHash: "$2a$10$adminhashpassword",
		Role:         "admin",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("repo.Create gagal: %v", err)
	}

	t.Cleanup(func() {
		if user.ID > 0 {
			_, _ = db.Exec("DELETE FROM users WHERE id = $1", user.ID)
		}
	})

	found, err := repo.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("repo.GetByEmail gagal: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("ID tidak cocok: ekspektasi %d, didapat %d", user.ID, found.ID)
	}
	if found.Email != user.Email {
		t.Errorf("Email tidak cocok: ekspektasi %q, didapat %q", user.Email, found.Email)
	}
}

// 5. Uji GetByEmail User NotFound — email tidak terdaftar mengembalikan ErrUserNotFound
func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	_, repo := setupUserTestDB(t)
	ctx := context.Background()

	found, err := repo.GetByEmail(ctx, "nonexistent_email_1234567@campus.ac.id")
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("Ekspektasi ErrUserNotFound, didapat error: %v", err)
	}
	if found != nil {
		t.Fatalf("Ekspektasi user nil saat not found, didapat: %+v", found)
	}
}

// 6. Uji Duplicate Email — constraint unique email mengembalikan ErrDuplicateEmail
func TestUserRepository_DuplicateEmail(t *testing.T) {
	db, repo := setupUserTestDB(t)
	ctx := context.Background()

	email := uniqueUserEmail("dup")
	user1 := &model.User{
		Name:         "User Pertama",
		Email:        email,
		PasswordHash: "hash1",
		Role:         "student",
	}

	err := repo.Create(ctx, user1)
	if err != nil {
		t.Fatalf("repo.Create user1 gagal: %v", err)
	}

	t.Cleanup(func() {
		if user1.ID > 0 {
			_, _ = db.Exec("DELETE FROM users WHERE id = $1", user1.ID)
		}
	})

	user2 := &model.User{
		Name:         "User Kedua",
		Email:        email, // Email yang sama persis
		PasswordHash: "hash2",
		Role:         "student",
	}

	err = repo.Create(ctx, user2)
	if !errors.Is(err, repository.ErrDuplicateEmail) {
		t.Fatalf("Ekspektasi ErrDuplicateEmail, didapat error: %v", err)
	}
}
