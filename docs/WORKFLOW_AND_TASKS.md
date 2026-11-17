# WORKFLOW_AND_TASKS.md — Pembagian Tugas dan Kolaborasi GitHub

## 1. Informasi repository dan aturan main

- Repository: https://github.com/Arvvvii/CampusSpace-Backend
- Visibility: Public.
- Branch utama: `main`.
- Branch protection yang telah disiapkan: perubahan melalui Pull Request, minimal satu approval, larangan deletion dan force push, serta conversation resolution sesuai konfigurasi repository.
- Arvi adalah Git Master dan satu-satunya anggota yang melakukan merge ke `main`.

Setiap anggota wajib bekerja pada branch sendiri, membuat micro commits, mengirim Pull Request, dan tidak push langsung ke `main`.

## 2. Pembagian peran dan file

| Anggota                                | Peran                                     | File/tanggung jawab                                                                                                                                                                                                                                                                            |
| -------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M. Arviansyah Desta Andini (434241003) | Git Master & Core Setup                   | Repository, branch protection, collaborator, README, AGENTS, docs, `.gitignore`, `.env.example`, `go.mod`, `config/*`, `database/postgres.go`, `helper/response.go` (belum dipisahkan pada MVP), `main.go`, `route/route.go` (belum dibuat pada MVP), integrasi, review, merge |
| Marsha Adhia Camilla (434241005)       | Database & Repository                     | `migrations/*`, `app/repository/*`; migration rooms, interface dan query PostgreSQL CRUD, penanganan error database                                                                                                                                                                        |
| Robithoh Ahmad (434241014)             | Model, Service, Route & Pendamping Teknis | `app/model/*`, `app/service/*`, `route/room.go`, unit test terkait; model Room, rules, service, route, bantuan teknis                                                                                                                                                                    |

Jika suatu perubahan membutuhkan file milik anggota lain, koordinasikan dahulu. Pembagian ini adalah batas kerja untuk mengurangi konflik, bukan larangan berdiskusi atau saling review.

**Catatan status saat ini:** CRUD `rooms` telah diintegrasikan ke `main` melalui PR. Implementasi response envelope berada di `route/room.go`, sedangkan wiring route Rooms berada di `main.go`; `helper/response.go` dan `route/route.go` masih merupakan target pemisahan struktur, bukan file yang sudah tersedia. PostgreSQL menggunakan `database/sql` dengan driver `pgx/v5/stdlib`.

## 3. Urutan pekerjaan MVP

Prioritas:

1. Pastikan seluruh anggota dapat mengakses repository dan branch `main` terbaru.
2. Sepakati kontrak Room pada `docs/CODING_STANDARDS.md`.
3. Marsha menyiapkan migration `rooms` dan repository.
4. Robithoh menyiapkan model, business rules, service, route, dan unit test.
5. Arvi menyiapkan konfigurasi, koneksi PostgreSQL, helper response, wiring dependency, dan integrasi.
6. Integrasikan melalui PR secara bertahap.
7. Jalankan aplikasi, migration, CRUD endpoint, dan test.
8. Simpan bukti aktual untuk laporan. Untuk status proyek saat ini, CRUD Rooms telah selesai diintegrasikan; langkah tersisa adalah audit, dokumentasi, validasi ulang setelah perubahan, dan pengumpulan bukti laporan.

Migration tabel `users`, `bookings`, dan `booking_logs` tercantum dalam rancangan sistem, tetapi bukan prasyarat untuk menyelesaikan CRUD `rooms`. Prioritaskan migration `rooms`; perluasan migration harus disepakati dan tidak menghambat MVP.

## 4. Branch yang disarankan

| Anggota  | Branch                                                          |
| -------- | --------------------------------------------------------------- |
| Arvi     | `chore/project-setup`, `feat/app-config`                    |
| Marsha   | `feat/room-migration`, `feat/room-repository`               |
| Robithoh | `feat/room-model`, `feat/room-service`, `feat/room-route` |

Branch `chore/project-setup` sudah pernah dibuat dan di-merge ke `main`. Untuk pekerjaan baru, buat branch dari `main` terbaru.

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
- `feat(repository): implement room create query`
- `feat(model): add Room entity`
- `feat(service): add room validation rules`
- `test(service): add room validation tests`
- `feat(route): register room endpoints`
- `feat(config): load application environment`

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
- Tidak ada dependency Redis/Asynq, worker, atau outbox prematur selama fokus CRUD rooms.
- Tidak ada `.env` asli, password, token, atau credential.
- Model, interface repository, API, dan migration konsisten.
- Kode sudah diformat dengan `gofmt`.
- `go build ./...` dan `go test ./...` dijalankan bila kode sudah siap.
- README dan dokumen diperbarui jika desain berubah.
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

Sebelum implementasi CRUD berjalan paralel, anggota harus menyepakati:

- Entity `Room` dan field yang dipakai.
- Primary key `int64`.
- Nama dan signature method repository.
- Format response dan error.
- Perilaku DELETE sebagai soft delete (`is_active = false`).
- Migration `rooms` sesuai field pada `ARCHITECTURE.md`.

Kontrak teknis berada di `docs/CODING_STANDARDS.md`; endpoint berada di `docs/API_SPEC.md`.

## 11. Hal yang belum boleh diasumsikan

- Jangan menganggap Marsha/Robithoh sudah menerima invitation sampai mereka mengonfirmasi.
- Jangan menganggap dependency atau database lokal setiap anggota sudah siap sebelum dicek.
- Jangan menganggap test integrasi PostgreSQL berhasil jika tidak dijalankan.
- Jangan mengimplementasikan Redis/Asynq, booking, auth, atau outbox hanya karena tercantum di rancangan.
- Jangan membuat username, email, credential, atau hasil pengujian fikti

# WORKFLOW_AND_TASKS.md — Pembagian Tugas dan Kolaborasi GitHub

## 1. Informasi repository dan aturan main

- Repository: https://github.com/Arvvvii/CampusSpace-Backend
- Visibility: Public.
- Branch utama: `main`.
- Branch protection yang telah disiapkan: perubahan melalui Pull Request, minimal satu approval, larangan deletion dan force push, serta conversation resolution sesuai konfigurasi repository.
- Arvi adalah Git Master dan satu-satunya anggota yang melakukan merge ke `main`.

Setiap anggota wajib bekerja pada branch sendiri, membuat micro commits, mengirim Pull Request, dan tidak push langsung ke `main`.

## 2. Pembagian peran dan file

| Anggota                                | Peran                                     | File/tanggung jawab                                                                                                                                                                                                                        |
| -------------------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| M. Arviansyah Desta Andini (434241003) | Git Master & Core Setup                   | Repository, branch protection, collaborator, README, AGENTS, docs, `.gitignore`, `.env.example`, `go.mod`, `config/*`, `database/postgres.go`, `helper/response.go`, `main.go`, `route/route.go`, integrasi, review, merge |
| Marsha Adhia Camilla (434241005)       | Database & Repository                     | `migrations/*`, `app/repository/*`; migration rooms, interface dan query PostgreSQL CRUD, penanganan error database                                                                                                                    |
| Robithoh Ahmad (434241014)             | Model, Service, Route & Pendamping Teknis | `app/model/*`, `app/service/*`, `route/room.go`, unit test terkait; model Room, rules, service, route, bantuan teknis                                                                                                                |

Jika suatu perubahan membutuhkan file milik anggota lain, koordinasikan dahulu. Pembagian ini adalah batas kerja untuk mengurangi konflik, bukan larangan berdiskusi atau saling review.

## 3. Urutan pekerjaan MVP

Prioritas:

1. Pastikan seluruh anggota dapat mengakses repository dan branch `main` terbaru.
2. Sepakati kontrak Room pada `docs/CODING_STANDARDS.md`.
3. Marsha menyiapkan migration `rooms` dan repository.
4. Robithoh menyiapkan model, business rules, service, route, dan unit test.
5. Arvi menyiapkan konfigurasi, koneksi PostgreSQL, helper response, wiring dependency, dan integrasi.
6. Integrasikan melalui PR secara bertahap.
7. Jalankan aplikasi, migration, CRUD endpoint, dan test.
8. Simpan bukti aktual untuk laporan.

Migration tabel `users`, `bookings`, dan `booking_logs` tercantum dalam rancangan sistem, tetapi bukan prasyarat untuk menyelesaikan CRUD `rooms`. Prioritaskan migration `rooms`; perluasan migration harus disepakati dan tidak menghambat MVP.

## 4. Branch yang disarankan

| Anggota  | Branch                                                          |
| -------- | --------------------------------------------------------------- |
| Arvi     | `chore/project-setup`, `feat/app-config`                    |
| Marsha   | `feat/room-migration`, `feat/room-repository`               |
| Robithoh | `feat/room-model`, `feat/room-service`, `feat/room-route` |

Branch `chore/project-setup` sudah pernah dibuat dan di-merge ke `main`. Untuk pekerjaan baru, buat branch dari `main` terbaru.

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
- `feat(repository): implement room create query`
- `feat(model): add Room entity`
- `feat(service): add room validation rules`
- `test(service): add room validation tests`
- `feat(route): register room endpoints`
- `feat(config): load application environment`

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
- Tidak ada dependency Redis/Asynq, worker, atau outbox prematur selama fokus CRUD rooms.
- Tidak ada `.env` asli, password, token, atau credential.
- Model, interface repository, API, dan migration konsisten.
- Kode sudah diformat dengan `gofmt`.
- `go build ./...` dan `go test ./...` dijalankan bila kode sudah siap.
- README dan dokumen diperbarui jika desain berubah.
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

Sebelum implementasi CRUD berjalan paralel, anggota harus menyepakati:

- Entity `Room` dan field yang dipakai.
- Primary key `int64`.
- Nama dan signature method repository.
- Format response dan error.
- Perilaku DELETE sebagai soft delete (`is_active = false`).
- Migration `rooms` sesuai field pada `ARCHITECTURE.md`.

Kontrak teknis berada di `docs/CODING_STANDARDS.md`; endpoint berada di `docs/API_SPEC.md`.

## 11. Hal yang belum boleh diasumsikan

- Jangan menganggap Marsha/Robithoh sudah menerima invitation sampai mereka mengonfirmasi.
- Jangan menganggap dependency atau database lokal setiap anggota sudah siap sebelum dicek.
- Jangan menganggap test integrasi PostgreSQL berhasil jika tidak dijalankan.
- Jangan mengimplementasikan Redis/Asynq, booking, auth, atau outbox hanya karena tercantum di rancangan.
- Jangan membuat username, email, credential, atau hasil pengujian fiktif.
