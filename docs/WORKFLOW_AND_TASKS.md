
# WORKFLOW_AND_TASKS.md — Pembagian Tugas dan Kolaborasi GitHub

## 1. Informasi repository dan aturan main

- Repository: https://github.com/Arvvvii/CampusSpace-Backend
- Visibility: Public.
- Branch utama: `main`.
- Branch protection yang telah disiapkan: perubahan melalui Pull Request, minimal satu approval, larangan deletion dan force push, serta conversation resolution sesuai konfigurasi repository.
- Arvi adalah Git Master dan satu-satunya anggota yang melakukan merge ke `main`.

Setiap anggota wajib bekerja pada branch sendiri, membuat micro commits, mengirim Pull Request, dan tidak push langsung ke `main`.

## 2. Pembagian peran dan file

| Anggota                                | Peran                                     | File/tanggung jawab                                                                                                                                                                                                                                                             |
| -------------------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M. Arviansyah Desta Andini (434241003) | Git Master & Core Setup                   | Repository, branch protection, collaborator, README, AGENTS, docs, `.gitignore`, `.env.example`, `go.mod`, `config/*`, `database/postgres.go`, `main.go`, `helper/response.go` *(target)*, `route/route.go` *(target)*, integrasi, review, audit, dan merge |
| Marsha Adhia Camilla (434241005)       | Database & Repository                     | `migrations/*` (`rooms`, `users`, `bookings`, `booking_logs`, `approver_rooms`), `app/repository/*` (interface, query PostgreSQL, row-level locking, transaksi, error mapping DB), serta integration test repository                                              |
| Robithoh Ahmad (434241014)             | Model, Service, Route & Pendamping Teknis | `app/model/*`, `app/service/*`, `route/room.go`, `route/booking.go`, unit test business rules/service, dan bantuan teknis                                                                                                                                               |

Jika suatu perubahan membutuhkan file milik anggota lain, koordinasikan dahulu. Pembagian ini adalah batas kerja untuk mengurangi konflik, bukan larangan berdiskusi atau saling review.

**Catatan status struktur & implementasi saat ini:**

- **Rooms CRUD (Implemented & Terverifikasi):** telah diintegrasikan ke `main` dan teruji end-to-end dengan PostgreSQL. Response envelope saat ini berada sebagai helper lokal di `route/room.go`, dan registrasi route dilakukan di `main.go`; pemisahan ke `helper/response.go` dan `route/route.go` belum dilakukan.
- **Booking & User (Dalam Pengembangan / Perbaikan di `fix/booking-review-improvements`):** migration dasar `002`/`003`, model, repository, service, dan route awal sudah dibuat, namun masih menjalani audit dan perbaikan. Pekerjaan mencakup penambahan `users.is_active` dan `bookings.updated_at` pada migration yang ada, pembuatan migration baru `004_create_approver_rooms_table.sql`, otorisasi role, validasi transisi status, pemetaan HTTP error, wiring ke `main.go`, serta kelengkapan unit & integration test.
- **Future Scope (Planned):** Autentikasi/authorization middleware (`middleware/`), `Idempotency-Key`, Transactional Outbox, Outbox Relay, Redis, Asynq, dan Notification Worker.

## 3. Urutan pekerjaan proyek

### Tahap 1 — Fondasi & MVP Rooms CRUD *(Selesai & Terintegrasi di `main`)*

1. Setup repository, branch protection, dan dokumen kontrak (`docs/*`, `AGENTS.md`).
2. `config/config.go` dan `database/postgres.go` menggunakan `database/sql` + `github.com/jackc/pgx/v5/stdlib`.
3. `migrations/001_create_rooms_table.sql` dan `RoomRepository` + integration tests.
4. `Room` model, `RoomService`, `route/room.go`, dan unit tests.
5. Dependency wiring dan penyelarasan health check `GET /` di `main.go`.

### Tahap 2 — Pengembangan Modul Users & Booking *(Sedang Berjalan)*

1. Penyelarasan migration `002` dan `003` untuk `users.is_active` serta `bookings.updated_at`, dan penambahan migration `004_create_approver_rooms_table.sql` untuk relasi `approver_rooms`.
2. Implementasi & perbaikan `UserRepository`, `BookingRepository` (row-level locking, atomic log, status transitions, `approver_rooms`), dan integration test PostgreSQL.
3. Implementasi & perbaikan `BookingService` (validasi user aktif, role `student`/`approver`/`admin`, aturan kepemilikan & penugasan ruangan) beserta unit test.
4. Implementasi & perbaikan `route/booking.go` (pemetaan HTTP `400`/`403`/`404`/`409`/`422`, validasi query param) dan wiring ke `main.go`.
5. Review integrasi, validasi seluruh unit test + integration test + build, lalu merge via PR ke `main`.

### Tahap 3 — Pengembangan Lanjutan *(Future Scope / Planned)*

1. Autentikasi (login/register, JWT/session) dan authorization middleware yang mengisi identitas aktor ke request context.
2. `Idempotency-Key` pada `POST /bookings/`.
3. Transactional Outbox, Outbox Relay, Redis, Asynq, dan Notification Worker.

## 4. Branch yang digunakan

| Anggota  | Branch Tahap Rooms (Merged)                                                                                                | Branch Tahap Booking & Perbaikan                                                                      |
| -------- | -------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Arvi     | `chore/project-setup`, `feat/app-config`, `feat/database-connection`, `feat/main-wiring`, `docs/audit-alignment` | `fix/booking-review-improvements`                                                                   |
| Marsha   | `feat/room-migration`, `feat/room-repository`                                                                          | `feat/user-booking-migration`, `feat/booking-repository`, `test/booking-repository-integration` |
| Robithoh | `feat/room-model`, `feat/room-service`, `feat/room-route`                                                            | `feat/booking-model`, `feat/booking-service`, `feat/booking-route`                              |

Untuk pekerjaan baru, selalu buat branch dari `main` terbaru. Jangan bekerja langsung di `main`.

Penting: jika dua anggota perlu mengerjakan kontrak yang saling bergantung, sepakati file/interface terlebih dahulu. Jangan menganggap branch fitur berikutnya sudah berisi perubahan PR yang belum di-merge. Sinkronisasi dari `main` dilakukan setelah PR terkait masuk.

## 5. Setup lokal anggota

Setelah undangan collaborator diterima:

```bash
git clone https://github.com/Arvvvii/CampusSpace-Backend.git
cd CampusSpace-Backend

git config user.name "Nama GitHub masing-masing"
git config user.email "Email GitHub masing-masing"

go version
git --version

git switch main
git pull origin main
git branch
```

Jangan menjalankan `git init` atau `git remote add origin` lagi di folder hasil clone. Jangan mengirim password atau token ke chat kelompok.

Buat branch kerja:

```bash
git switch main
git pull origin main
git switch -c feat/nama-tugas
git branch
```

Tanda `*` pada output `git branch` menunjukkan branch aktif.

## 6. Siklus kerja micro commits

1. Pastikan branch aktif adalah branch tugas sendiri.
2. Kerjakan satu perubahan kecil.
3. Jalankan formatter/test yang relevan.
4. Periksa file yang berubah:

   ```bash
   git status
   git diff
   ```
5. Stage file yang memang terkait; hindari `git add .` bila ada file lokal/rahasia.
6. Buat commit yang menjelaskan satu perubahan:

   ```bash
   git add path/file-yang-diubah
   git commit -m "feat(scope): deskripsi perubahan"
   ```
7. Push branch:

   ```bash
   git push -u origin nama-branch
   ```

   Untuk push berikutnya pada branch yang sudah punya upstream:

   ```bash
   git push
   ```

Contoh pesan commit:

- `feat(migration): add rooms table schema`
- `feat(repository): define RoomRepository interface`
- `feat(model): add Booking and User entities`
- `feat(service): add booking authorization and validation rules`
- `test(repository): add booking concurrency integration test`
- `feat(route): register booking endpoints`
- `fix(booking): enforce status transition and role rules`

## 7. Pull Request

Setelah push:

1. Buka GitHub → Pull requests → New pull request.
2. Pilih `base: main` dan `compare: branch-kamu`.
3. Isi judul dan ringkasan perubahan.
4. Jelaskan file yang diubah dan cara pengujian.
5. Minta anggota lain melakukan review/approval sesuai branch protection.
6. Tanggapi komentar review dan push perbaikan ke branch PR yang sama.

Jangan merge PR sendiri. Arvi melakukan review integrasi dan merge setelah syarat branch protection terpenuhi.

## 8. Checklist review Git Master

- Perubahan sesuai pembagian package Modul 4.
- Tidak ada folder/layer `controller`, `handler`, `usecase`, atau `delivery`.
- Tidak ada dependency Redis/Asynq, worker, atau outbox prematur sebelum disetujui masuk scope implementasi.
- Tidak ada `.env` asli, password (`password_hash` tidak diekspos JSON), token, atau credential.
- Model, interface repository, API, dan migration konsisten.
- Kode sudah diformat dengan `gofmt`.
- `go build ./...`, unit test (`go test ./...`), dan integration test (`go test -tags=integration ./app/repository -v`) dijalankan bila kode sudah siap.
- Dokumentasi (`README.md`, `docs/*`) diperbarui jika kontrak/desain berubah.
- Hasil test yang dilaporkan benar-benar dijalankan.

Metode merge yang disarankan: **Create a merge commit** agar micro commits tetap terlihat.

## 9. Setelah PR di-merge

Setiap anggota memperbarui branch utama lokal:

```bash
git switch main
git pull origin main
```

Untuk pekerjaan berikutnya, buat branch baru dari `main` terbaru. Jika masih ada perubahan lokal yang belum di-commit, jangan melakukan switch/pull secara membabi buta; periksa dan amankan perubahan terlebih dahulu.

## 10. Kontrak koordinasi sebelum coding paralel

Sebelum implementasi fitur berjalan paralel, anggota harus menyepakati:

- Entity (`Room`, `User`, `Booking`, `BookingLog`, `ApproverRoom`) dan field beserta tag JSON/DB-nya; `ApproverRoom` memerlukan migration `004`.
- Primary key `int64`.
- Nama dan signature method interface repository dan service.
- Matriks transisi status booking dan hak akses tiap role (`student`, `approver`, `admin`).
- Format response envelope dan pemetaan HTTP status code (`400`, `403`, `404`, `409`, `422`, `500`).
- Perilaku soft delete `rooms` (`is_active = false`) dan schema migration SQL terbaru.

Kontrak teknis berada di `docs/CODING_STANDARDS.md` dan `docs/ARCHITECTURE.md`; endpoint berada di `docs/API_SPEC.md`.

## 11. Hal yang belum boleh diasumsikan

- Jangan menganggap database lokal/test setiap anggota otomatis sudah menerapkan perubahan migration terbaru sebelum dijalankan secara manual.
- Jangan menganggap fitur Booking sudah aktif di aplikasi sebelum wiring di `main.go` selesai dan seluruh unit serta integration test lulus.
- Jangan mengklaim sistem sudah memiliki proteksi autentikasi atau authorization middleware jika middleware belum diimplementasikan; penggunaan `user_id`/`actor_id` saat ini hanya untuk pengujian sebelum integrasi auth context.
- Jangan mengimplementasikan atau mengklaim Redis, Asynq, Transactional Outbox, atau `Idempotency-Key` sudah berjalan sebelum masuk scope implementasi dan teruji.
- Jangan membuat username, email, credential, atau hasil pengujian fiktif.
