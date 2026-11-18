# ARCHITECTURE.md — Rancangan Sistem, Database, dan Keandalan CampusSpace

## 1. Ringkasan sistem

CampusSpace adalah backend REST API untuk reservasi fasilitas kampus, seperti laboratorium komputer, ruang seminar, aula, dan ruang organisasi mahasiswa. Sistem dirancang untuk membantu civitas akademika melihat ruangan, mengajukan reservasi, dan memproses persetujuan oleh pengelola ruangan.

Tiga role dalam rancangan dan aturan bisnis sistem:

- `student`: peminjam.
- `approver`: pengelola ruangan.
- `admin`: administrator.

Hak akses bisnis per role:

- **Student:**
  - Melihat ruangan dan memeriksa ketersediaan ruangan.
  - Mengajukan booking baru (hanya `student` berstatus aktif `is_active = TRUE`).
  - Membatalkan (`cancelled`) booking miliknya sendiri yang masih berstatus `pending` atau `approved`.
  - Melihat riwayat booking.
  - Tidak boleh menyetujui (`approved`), menolak (`rejected`), atau membatalkan booking milik user lain.
- **Approver:**
  - Melihat pengajuan booking pada ruangan yang menjadi tanggung jawabnya (terdaftar di `approver_rooms`).
  - Menyetujui (`approved`) atau menolak (`rejected`) booking berstatus `pending` pada ruangan yang ditugaskan kepadanya, disertai catatan/alasan opsional.
  - Tidak boleh membatalkan (`cancelled`) booking maupun memutuskan booking untuk ruangan di luar penugasannya.
- **Admin:**
  - Mengelola data ruangan, mengelola akun/role pengguna, dan memantau data booking.
  - Tidak boleh mengajukan booking maupun melakukan `approved`, `rejected`, atau `cancelled` terhadap booking.

**Catatan autentikasi & otorisasi:** Autentikasi (login/token/session) dan authorization middleware di layer HTTP belum diimplementasikan (`middleware/` masih belum memiliki implementasi). Aturan hak akses di atas merupakan aturan bisnis yang menjadi target implementasi pada layer service. Untuk versi produksi, identitas aktor harus diambil dari authenticated request context, bukan dipercaya secara bebas dari request body.

## 2. Teknologi dan arsitektur

| Area           | Teknologi/status                                                                     |
| -------------- | ------------------------------------------------------------------------------------ |
| Bahasa         | Go                                                                                   |
| HTTP framework | Fiber v2                                                                             |
| Database       | PostgreSQL melalui `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib` |
| Arsitektur     | Clean Architecture mengikuti Modul 4                                                 |
| Queue/cache    | Redis + Asynq (future scope / rancangan notifikasi asynchronous)                     |

Struktur package:

```text
app/model       Entity dan DTO
app/repository  Interface dan implementasi akses database
app/service     Alur aplikasi dan business rules
config          Konfigurasi aplikasi, environment, logger
database        Koneksi PostgreSQL dan connection pool
helper          Fungsi bantuan (belum diisi pada tahap saat ini)
middleware      Middleware aplikasi (belum diisi; auth adalah future scope)
route           Registrasi endpoint HTTP (room.go aktif; booking.go dalam pengembangan)
migrations      SQL migration
main.go         Dependency wiring dan entry point
```

Jangan menambahkan `controller`, `handler`, `usecase`, atau `delivery`; struktur mengikuti Modul 4.

**Status implementasi package saat ini:**

- **Sudah diimplementasikan dan terverifikasi (`main`):** `config`, `database`, serta modul Rooms (`app/model/room.go`, `app/repository/room_repository.go`, `app/service/room_service.go`, `route/room.go`) yang sudah terhubung di `main.go` dan `migrations/001_create_rooms_table.sql`.
- **Sedang dikembangkan / dalam tahap review perbaikan:** modul Users dan Bookings (`app/model/user.go`, `app/model/booking.go`, `app/repository/user_repository.go`, `app/repository/booking_repository.go`, `app/service/booking_service.go`, `route/booking.go`, `migrations/002`, `migrations/003`, serta migration `004` yang direncanakan untuk `approver_rooms`). Endpoint Booking belum di-wire di `main.go` dan masih menjalani perbaikan sebelum dianggap selesai.
- **Belum diimplementasikan (Placeholder / Future Scope):** `helper/response.go` (response envelope masih berupa helper lokal di `route/room.go`), `route/route.go`, `middleware/` (autentikasi/otorisasi), `Idempotency-Key`, Transactional Outbox, Redis, Asynq, dan Notification Worker.

## 3. Skema database inti

Skema database terdiri atas tabel inti `users`, `rooms`, `bookings`, `booking_logs`, serta tabel relasi `approver_rooms`. Tabel `approver_rooms` sudah ditetapkan sebagai desain relasi penugasan approver, tetapi migration dan implementasinya masih perlu dibuat serta diverifikasi.

### 3.1 `users` *(dalam pengembangan / perbaikan skema)*

| Kolom             | Tipe rancangan | Keterangan                                    |
| ----------------- | -------------- | --------------------------------------------- |
| `id`            | BIGSERIAL      | Primary key                                   |
| `name`          | VARCHAR(100)   | Nama pengguna, NOT NULL                       |
| `email`         | VARCHAR(100)   | Email, UNIQUE dan NOT NULL                    |
| `password_hash` | TEXT           | Password hash, NOT NULL (tidak diekspos JSON) |
| `role`          | VARCHAR(20)    | `student`, `approver`, `admin`          |
| `is_active`     | BOOLEAN        | Status aktif user, NOT NULL, default TRUE     |
| `created_at`    | TIMESTAMPTZ    | Default waktu pembuatan                       |

### 3.2 `rooms` *(sudah diimplementasikan & dimigrasi)*

| Kolom          | Tipe rancangan | Keterangan                     |
| -------------- | -------------- | ------------------------------ |
| `id`         | BIGSERIAL      | Primary key                    |
| `code`       | VARCHAR(50)    | Kode unik, UNIQUE dan NOT NULL |
| `name`       | VARCHAR(100)   | Nama ruangan, NOT NULL         |
| `building`   | VARCHAR(100)   | Gedung/lokasi, NOT NULL        |
| `capacity`   | INTEGER        | NOT NULL dan harus > 0         |
| `is_active`  | BOOLEAN        | NOT NULL, default TRUE         |
| `created_at` | TIMESTAMPTZ    | Default waktu pembuatan        |

`DELETE /rooms/:id` diimplementasikan sebagai soft delete: mengubah `is_active` menjadi `false`, bukan menghapus row secara fisik, agar relasi/histori booking tetap terjaga.

### 3.3 `bookings` *(dalam pengembangan / perbaikan skema)*

| Kolom          | Tipe rancangan | Keterangan                                                                   |
| -------------- | -------------- | ---------------------------------------------------------------------------- |
| `id`         | BIGSERIAL      | Primary key                                                                  |
| `user_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL                                       |
| `room_id`    | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                                       |
| `purpose`    | TEXT           | Tujuan reservasi, NOT NULL                                                   |
| `start_time` | TIMESTAMPTZ    | Waktu mulai, NOT NULL                                                        |
| `end_time`   | TIMESTAMPTZ    | Waktu selesai, NOT NULL (`start_time < end_time`)                          |
| `status`     | VARCHAR(20)    | `pending`, `approved`, `rejected`, `cancelled` (default `pending`) |
| `created_at` | TIMESTAMPTZ    | Waktu pengajuan, NOT NULL, default `CURRENT_TIMESTAMP`                     |
| `updated_at` | TIMESTAMPTZ    | Waktu pembaruan terakhir, NOT NULL, default `CURRENT_TIMESTAMP`            |

### 3.4 `booking_logs` *(dalam pengembangan)*

| Kolom           | Tipe rancangan | Keterangan                                       |
| --------------- | -------------- | ------------------------------------------------ |
| `id`          | BIGSERIAL      | Primary key                                      |
| `booking_id`  | BIGINT         | Foreign key ke `bookings(id)`, NOT NULL        |
| `actor_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL           |
| `from_status` | VARCHAR(20)    | Status sebelumnya; `NULL` untuk initial status |
| `to_status`   | VARCHAR(20)    | Status setelah perubahan, NOT NULL               |
| `note`        | TEXT           | Catatan/alasan perubahan                         |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL                       |

### 3.5 `approver_rooms` *(rancangan; migration belum dibuat)*

| Kolom           | Tipe rancangan | Keterangan                                                |
| --------------- | -------------- | --------------------------------------------------------- |
| `id`          | BIGSERIAL      | Primary key                                               |
| `approver_id` | BIGINT         | Foreign key ke `users(id)`, NOT NULL                    |
| `room_id`     | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                    |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL, default `CURRENT_TIMESTAMP` |

Constraint: `UNIQUE (approver_id, room_id)` untuk mencegah duplikasi penugasan approver pada ruangan yang sama. Validasi bahwa `approver_id` memiliki role `approver` dilakukan pada layer aplikasi/service. Migration yang direncanakan: `004_create_approver_rooms_table.sql`.

`from_status` dan `to_status` pada `booking_logs` dibatasi pada nilai `pending`, `approved`, `rejected`, atau `cancelled`; `from_status` boleh `NULL` hanya untuk pencatatan status awal booking.

### Relasi Tabel

- `users.id` → `bookings.user_id`
- `rooms.id` → `bookings.room_id`
- `bookings.id` → `booking_logs.booking_id`
- `users.id` → `booking_logs.actor_id`
- `users.id` → `approver_rooms.approver_id`
- `rooms.id` → `approver_rooms.room_id`

## 4. Aturan bisnis reservasi

Aturan bisnis modul booking:

1. `start_time` harus lebih awal daripada `end_time` (`start_time < end_time`), dan keduanya tidak boleh bernilai kosong/zero.
2. Hanya `student` berstatus aktif (`is_active = TRUE`) yang boleh mengajukan booking baru; `admin` dan `approver` tidak boleh mengajukan booking.
3. Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
4. Booking baru wajib dibuat dengan status awal `pending` dan selalu mencatat initial log pada `booking_logs` (`from_status = NULL`, `to_status = 'pending'`).
5. Status `pending` dan `approved` menahan slot waktu ruangan.
6. Status `rejected` dan `cancelled` tidak menahan slot waktu ruangan.
7. Setiap perubahan status yang valid wajib menghasilkan tepat satu catatan baru di `booking_logs` dan memperbarui `bookings.updated_at` secara atomik dalam satu transaksi.

### Matriks transisi status booking

Status yang diperbolehkan: `pending`, `approved`, `rejected`, `cancelled`.

| Dari Status   | Ke Status     | Diperbolehkan | Aktor yang Berwenang                                               |
| ------------- | ------------- | ------------- | ------------------------------------------------------------------ |
| `pending`   | `approved`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `rejected`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `approved`  | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `rejected`  | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| `cancelled` | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| *(lainnya)* | *(lainnya)* | Tidak         | Ditolak (`409 Conflict`)                                         |

`admin` tidak boleh melakukan `approved`, `rejected`, maupun `cancelled`. Untuk mengajukan ulang slot setelah `rejected` atau `cancelled`, pengguna harus membuat booking baru.

### Rumus konflik jadwal

Dua rentang waktu bertabrakan jika:

```text
existing.start_time < requested.end_time
AND existing.end_time > requested.start_time
```

Rentang yang tepat bersentuhan pada batas waktu (misalnya `09:00–10:00` dan `10:00–11:00`) tidak dianggap overlap menurut rumus tersebut sehingga tetap diperbolehkan.

## 5. Transaksi, race condition, dan idempotency booking

*Status bagian ini: mekanisme transaksi dan row-level locking pada repository booking sedang dalam tahap penyempurnaan dan verifikasi; endpoint Booking belum aktif di `main.go`. `Idempotency-Key` masih merupakan rancangan pengembangan lanjutan (future scope).*

### 5.1 Pencegahan double booking

Race condition dapat terjadi jika dua request memeriksa slot yang sama sebelum salah satunya menyimpan booking. Rancangan yang digunakan adalah transaksi PostgreSQL dengan pessimistic locking pada baris ruangan (`rooms`) dan pembacaan status aktif user:

```sql
BEGIN;

-- 1. Verifikasi user aktif dan role student
SELECT is_active, role
FROM users
WHERE id = $1
FOR SHARE;

-- 2. Kunci row ruangan agar request concurrent pada room yang sama berjalan serial
SELECT id, is_active
FROM rooms
WHERE id = $2
FOR UPDATE;

-- 3. Periksa booking berstatus pending/approved dengan rentang waktu overlap

-- 4. Jika ruangan aktif dan tidak ada konflik, INSERT booking (status = 'pending')
--    dan INSERT initial log ke booking_logs dalam transaksi yang sama

COMMIT;
```

Semua jalur pembuatan booking untuk `room_id` yang sama wajib mengunci baris ruangan yang sama sebelum memeriksa konflik. Pengecekan di luar transaksi tidak memenuhi rancangan ini.

Untuk perubahan status (`UpdateStatus`), transaksi mengunci baris booking target dengan `SELECT status FROM bookings WHERE id = $1 FOR UPDATE` guna mencegah lost update dan pelanggaran transisi status saat request bersamaan.

### 5.2 ACID

- **Atomicity:** Insert/update booking dan pencatatan `booking_logs` dijalankan dalam satu transaksi PostgreSQL. Jika insert log gagal, seluruh transaksi booking wajib rollback.
- **Consistency:** Validasi service, check constraint (`start_time < end_time`, `status`), unique constraint, dan foreign key menjaga integritas data.
- **Isolation:** Row-level locking (`FOR UPDATE`) pada `rooms` dan `bookings` mencegah dua transaksi mengambil slot konflik atau mengubah status yang sama secara bersamaan.
- **Durability:** Perubahan yang berhasil di-commit disimpan secara persisten oleh PostgreSQL.

### 5.3 Idempotency-Key *(Future Scope — Belum Diimplementasikan)*

Untuk `POST /bookings`, pada tahap pengembangan lanjutan client direncanakan mengirim header `Idempotency-Key` yang diikat ke user dan request. Rancangan tabel `idempotency_keys` memiliki kolom utama:

- `id`
- `user_id`
- `idempotency_key`
- `request_hash`
- `response_status`
- `response_body`
- `created_at`

Constraint yang direncanakan: `UNIQUE(user_id, idempotency_key)`. Key sama dengan payload sama mengembalikan hasil yang tersimpan; key sama dengan payload berbeda ditolak sebagai konflik. Penanganan request bersamaan harus mengandalkan unique constraint dan transaksi, bukan hanya pola SELECT lalu INSERT.

## 6. Rancangan notifikasi asynchronous: Transactional Outbox + Redis/Asynq *(Future Scope — Belum Diimplementasikan)*

Notifikasi asynchronous adalah pengembangan lanjutan yang dikaitkan dengan materi Modul 11 dan belum menjadi bagian dari implementasi aktif saat ini.

Komponen yang direncanakan:

- **Transactional Outbox:** menyimpan event notifikasi bersama perubahan booking dalam transaksi PostgreSQL yang sama.
- **Outbox Relay:** membaca event outbox yang belum diteruskan dan membuat job antrean.
- **Redis:** backend antrean yang digunakan Asynq.
- **Asynq:** library/task queue Go untuk mengelola job asynchronous dan retry.
- **Notification Worker:** mengambil job dan menjalankan pengiriman notifikasi.
- **Consumer idempotency:** mencegah efek samping ganda saat job diproses ulang.

Rancangan tabel `outbox_events` masih perlu difinalisasi sebelum implementasi. Kolom awal yang direncanakan: `id`, `event_type`, `payload`, `created_at`, `processed_at`. Detail status, retry, locking relay, dan kebijakan retensi belum ditetapkan dan tidak boleh ditebak oleh AI.

Alur yang direncanakan:

```text
Booking Service
    |
    v
PostgreSQL Transaction
  - perubahan booking
  - insert outbox event
    |
    v
Outbox Relay
    |
    v
Asynq Queue <--> Redis
    |
    v
Notification Worker
    |
    v
Notification Delivery
```

Jangan memasang dependency Redis/Asynq, membuat worker, atau menambahkan tabel outbox sebelum tim menyetujui masuk ke tahap pengembangan notifikasi asynchronous.

## 7. Batas ruang lingkup dan hal yang belum diputuskan

Keputusan yang sudah ditetapkan:

- **Driver PostgreSQL:** `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib`.
- **Desain relasi approver-ruangan:** tabel `approver_rooms` dengan kolom `approver_id`, `room_id`, dan `UNIQUE (approver_id, room_id)`. Implementasi migration `004` masih perlu dibuat dan diverifikasi.
- **Matriks transisi status booking & hak akses role (`student`, `approver`, `admin`):** ditetapkan pada Bagian 1 dan Bagian 4.

Hal berikut masih belum ditentukan/diimplementasikan dan harus disepakati sebelum implementasi terkait:

- Strategi migration runner otomatis (saat ini file SQL di `migrations/` dijalankan secara manual).
- Desain final autentikasi (JWT/session, endpoint login/register, hashing password) dan authorization middleware di `middleware/`.
- Pagination, filtering, dan sorting untuk `GET /rooms` serta perluasan filter/pagination pada `GET /bookings`.
- Kebijakan validasi/format waktu dan timezone API secara menyeluruh di luar format RFC3339 pada parameter query.
- Bentuk final objek/detail `errors` terstruktur pada response envelope untuk semua kondisi.
- Implementasi `Idempotency-Key` dan tabel `idempotency_keys`.
- Provider/channel pengiriman notifikasi serta detail schema, retry, dan retensi `outbox_events`

# ARCHITECTURE.md — Rancangan Sistem, Database, dan Keandalan CampusSpace

## 1. Ringkasan sistem

CampusSpace adalah backend REST API untuk reservasi fasilitas kampus, seperti laboratorium komputer, ruang seminar, aula, dan ruang organisasi mahasiswa. Sistem dirancang untuk membantu civitas akademika melihat ruangan, mengajukan reservasi, dan memproses persetujuan oleh pengelola ruangan.

Tiga role dalam rancangan dan aturan bisnis sistem:

- `student`: peminjam.
- `approver`: pengelola ruangan.
- `admin`: administrator.

Hak akses bisnis per role:

- **Student:**
  - Melihat ruangan dan memeriksa ketersediaan ruangan.
  - Mengajukan booking baru (hanya `student` berstatus aktif `is_active = TRUE`).
  - Membatalkan (`cancelled`) booking miliknya sendiri yang masih berstatus `pending` atau `approved`.
  - Melihat riwayat booking.
  - Tidak boleh menyetujui (`approved`), menolak (`rejected`), atau membatalkan booking milik user lain.
- **Approver:**
  - Melihat pengajuan booking pada ruangan yang menjadi tanggung jawabnya (terdaftar di `approver_rooms`).
  - Menyetujui (`approved`) atau menolak (`rejected`) booking berstatus `pending` pada ruangan yang ditugaskan kepadanya, disertai catatan/alasan opsional.
  - Tidak boleh membatalkan (`cancelled`) booking maupun memutuskan booking untuk ruangan di luar penugasannya.
- **Admin:**
  - Mengelola data ruangan, mengelola akun/role pengguna, dan memantau data booking.
  - Tidak boleh mengajukan booking maupun melakukan `approved`, `rejected`, atau `cancelled` terhadap booking.

**Catatan autentikasi & otorisasi:** Autentikasi (login/token/session) dan authorization middleware di layer HTTP belum diimplementasikan (`middleware/` masih belum memiliki implementasi). Aturan hak akses di atas merupakan aturan bisnis yang menjadi target implementasi pada layer service. Untuk versi produksi, identitas aktor harus diambil dari authenticated request context, bukan dipercaya secara bebas dari request body.

## 2. Teknologi dan arsitektur

| Area           | Teknologi/status                                                                     |
| -------------- | ------------------------------------------------------------------------------------ |
| Bahasa         | Go                                                                                   |
| HTTP framework | Fiber v2                                                                             |
| Database       | PostgreSQL melalui `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib` |
| Arsitektur     | Clean Architecture mengikuti Modul 4                                                 |
| Queue/cache    | Redis + Asynq (future scope / rancangan notifikasi asynchronous)                     |

Struktur package:

```text
app/model       Entity dan DTO
app/repository  Interface dan implementasi akses database
app/service     Alur aplikasi dan business rules
config          Konfigurasi aplikasi, environment, logger
database        Koneksi PostgreSQL dan connection pool
helper          Fungsi bantuan (belum diisi pada tahap saat ini)
middleware      Middleware aplikasi (belum diisi; auth adalah future scope)
route           Registrasi endpoint HTTP (room.go aktif; booking.go dalam pengembangan)
migrations      SQL migration
main.go         Dependency wiring dan entry point
```

Jangan menambahkan `controller`, `handler`, `usecase`, atau `delivery`; struktur mengikuti Modul 4.

**Status implementasi package saat ini:**

- **Sudah diimplementasikan dan terverifikasi (`main`):** `config`, `database`, serta modul Rooms (`app/model/room.go`, `app/repository/room_repository.go`, `app/service/room_service.go`, `route/room.go`) yang sudah terhubung di `main.go` dan `migrations/001_create_rooms_table.sql`.
- **Sedang dikembangkan / dalam tahap review perbaikan:** modul Users dan Bookings (`app/model/user.go`, `app/model/booking.go`, `app/repository/user_repository.go`, `app/repository/booking_repository.go`, `app/service/booking_service.go`, `route/booking.go`, `migrations/002`, `migrations/003`, serta migration `004` yang direncanakan untuk `approver_rooms`). Endpoint Booking belum di-wire di `main.go` dan masih menjalani perbaikan sebelum dianggap selesai.
- **Belum diimplementasikan (Placeholder / Future Scope):** `helper/response.go` (response envelope masih berupa helper lokal di `route/room.go`), `route/route.go`, `middleware/` (autentikasi/otorisasi), `Idempotency-Key`, Transactional Outbox, Redis, Asynq, dan Notification Worker.

## 3. Skema database inti

Skema database terdiri atas tabel inti `users`, `rooms`, `bookings`, `booking_logs`, serta tabel relasi `approver_rooms`. Tabel `approver_rooms` sudah ditetapkan sebagai desain relasi penugasan approver, tetapi migration dan implementasinya masih perlu dibuat serta diverifikasi.

### 3.1 `users` *(dalam pengembangan / perbaikan skema)*

| Kolom             | Tipe rancangan | Keterangan                                    |
| ----------------- | -------------- | --------------------------------------------- |
| `id`            | BIGSERIAL      | Primary key                                   |
| `name`          | VARCHAR(100)   | Nama pengguna, NOT NULL                       |
| `email`         | VARCHAR(100)   | Email, UNIQUE dan NOT NULL                    |
| `password_hash` | TEXT           | Password hash, NOT NULL (tidak diekspos JSON) |
| `role`          | VARCHAR(20)    | `student`, `approver`, `admin`          |
| `is_active`     | BOOLEAN        | Status aktif user, NOT NULL, default TRUE     |
| `created_at`    | TIMESTAMPTZ    | Default waktu pembuatan                       |

### 3.2 `rooms` *(sudah diimplementasikan & dimigrasi)*

| Kolom          | Tipe rancangan | Keterangan                     |
| -------------- | -------------- | ------------------------------ |
| `id`         | BIGSERIAL      | Primary key                    |
| `code`       | VARCHAR(50)    | Kode unik, UNIQUE dan NOT NULL |
| `name`       | VARCHAR(100)   | Nama ruangan, NOT NULL         |
| `building`   | VARCHAR(100)   | Gedung/lokasi, NOT NULL        |
| `capacity`   | INTEGER        | NOT NULL dan harus > 0         |
| `is_active`  | BOOLEAN        | NOT NULL, default TRUE         |
| `created_at` | TIMESTAMPTZ    | Default waktu pembuatan        |

`DELETE /rooms/:id` diimplementasikan sebagai soft delete: mengubah `is_active` menjadi `false`, bukan menghapus row secara fisik, agar relasi/histori booking tetap terjaga.

### 3.3 `bookings` *(dalam pengembangan / perbaikan skema)*

| Kolom          | Tipe rancangan | Keterangan                                                                   |
| -------------- | -------------- | ---------------------------------------------------------------------------- |
| `id`         | BIGSERIAL      | Primary key                                                                  |
| `user_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL                                       |
| `room_id`    | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                                       |
| `purpose`    | TEXT           | Tujuan reservasi, NOT NULL                                                   |
| `start_time` | TIMESTAMPTZ    | Waktu mulai, NOT NULL                                                        |
| `end_time`   | TIMESTAMPTZ    | Waktu selesai, NOT NULL (`start_time < end_time`)                          |
| `status`     | VARCHAR(20)    | `pending`, `approved`, `rejected`, `cancelled` (default `pending`) |
| `created_at` | TIMESTAMPTZ    | Waktu pengajuan, NOT NULL, default `CURRENT_TIMESTAMP`                     |
| `updated_at` | TIMESTAMPTZ    | Waktu pembaruan terakhir, NOT NULL, default `CURRENT_TIMESTAMP`            |

### 3.4 `booking_logs` *(dalam pengembangan)*

| Kolom           | Tipe rancangan | Keterangan                                       |
| --------------- | -------------- | ------------------------------------------------ |
| `id`          | BIGSERIAL      | Primary key                                      |
| `booking_id`  | BIGINT         | Foreign key ke `bookings(id)`, NOT NULL        |
| `actor_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL           |
| `from_status` | VARCHAR(20)    | Status sebelumnya; `NULL` untuk initial status |
| `to_status`   | VARCHAR(20)    | Status setelah perubahan, NOT NULL               |
| `note`        | TEXT           | Catatan/alasan perubahan                         |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL                       |

### 3.5 `approver_rooms` *(rancangan; migration belum dibuat)*

| Kolom           | Tipe rancangan | Keterangan                                                |
| --------------- | -------------- | --------------------------------------------------------- |
| `id`          | BIGSERIAL      | Primary key                                               |
| `approver_id` | BIGINT         | Foreign key ke `users(id)`, NOT NULL                    |
| `room_id`     | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                    |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL, default `CURRENT_TIMESTAMP` |

Constraint: `UNIQUE (approver_id, room_id)` untuk mencegah duplikasi penugasan approver pada ruangan yang sama. Validasi bahwa `approver_id` memiliki role `approver` dilakukan pada layer aplikasi/service. Migration yang direncanakan: `004_create_approver_rooms_table.sql`.

`from_status` dan `to_status` pada `booking_logs` dibatasi pada nilai `pending`, `approved`, `rejected`, atau `cancelled`; `from_status` boleh `NULL` hanya untuk pencatatan status awal booking.

### Relasi Tabel

- `users.id` → `bookings.user_id`
- `rooms.id` → `bookings.room_id`
- `bookings.id` → `booking_logs.booking_id`
- `users.id` → `booking_logs.actor_id`
- `users.id` → `approver_rooms.approver_id`
- `rooms.id` → `approver_rooms.room_id`

## 4. Aturan bisnis reservasi

Aturan bisnis modul booking:

1. `start_time` harus lebih awal daripada `end_time` (`start_time < end_time`), dan keduanya tidak boleh bernilai kosong/zero.
2. Hanya `student` berstatus aktif (`is_active = TRUE`) yang boleh mengajukan booking baru; `admin` dan `approver` tidak boleh mengajukan booking.
3. Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
4. Booking baru wajib dibuat dengan status awal `pending` dan selalu mencatat initial log pada `booking_logs` (`from_status = NULL`, `to_status = 'pending'`).
5. Status `pending` dan `approved` menahan slot waktu ruangan.
6. Status `rejected` dan `cancelled` tidak menahan slot waktu ruangan.
7. Setiap perubahan status yang valid wajib menghasilkan tepat satu catatan baru di `booking_logs` dan memperbarui `bookings.updated_at` secara atomik dalam satu transaksi.

### Matriks transisi status booking

Status yang diperbolehkan: `pending`, `approved`, `rejected`, `cancelled`.

| Dari Status   | Ke Status     | Diperbolehkan | Aktor yang Berwenang                                               |
| ------------- | ------------- | ------------- | ------------------------------------------------------------------ |
| `pending`   | `approved`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `rejected`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `approved`  | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `rejected`  | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| `cancelled` | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| *(lainnya)* | *(lainnya)* | Tidak         | Ditolak (`409 Conflict`)                                         |

`admin` tidak boleh melakukan `approved`, `rejected`, maupun `cancelled`. Untuk mengajukan ulang slot setelah `rejected` atau `cancelled`, pengguna harus membuat booking baru.

### Rumus konflik jadwal

Dua rentang waktu bertabrakan jika:

```text
existing.start_time < requested.end_time
AND existing.end_time > requested.start_time
```

Rentang yang tepat bersentuhan pada batas waktu (misalnya `09:00–10:00` dan `10:00–11:00`) tidak dianggap overlap menurut rumus tersebut sehingga tetap diperbolehkan.

## 5. Transaksi, race condition, dan idempotency booking

*Status bagian ini: mekanisme transaksi dan row-level locking pada repository booking sedang dalam tahap penyempurnaan dan verifikasi; endpoint Booking belum aktif di `main.go`. `Idempotency-Key` masih merupakan rancangan pengembangan lanjutan (future scope).*

### 5.1 Pencegahan double booking

Race condition dapat terjadi jika dua request memeriksa slot yang sama sebelum salah satunya menyimpan booking. Rancangan yang digunakan adalah transaksi PostgreSQL dengan pessimistic locking pada baris ruangan (`rooms`) dan pembacaan status aktif user:

```sql
BEGIN;

-- 1. Verifikasi user aktif dan role student
SELECT is_active, role
FROM users
WHERE id = $1
FOR SHARE;

-- 2. Kunci row ruangan agar request concurrent pada room yang sama berjalan serial
SELECT id, is_active
FROM rooms
WHERE id = $2
FOR UPDATE;

-- 3. Periksa booking berstatus pending/approved dengan rentang waktu overlap

-- 4. Jika ruangan aktif dan tidak ada konflik, INSERT booking (status = 'pending')
--    dan INSERT initial log ke booking_logs dalam transaksi yang sama

COMMIT;
```

Semua jalur pembuatan booking untuk `room_id` yang sama wajib mengunci baris ruangan yang sama sebelum memeriksa konflik. Pengecekan di luar transaksi tidak memenuhi rancangan ini.

Untuk perubahan status (`UpdateStatus`), transaksi mengunci baris booking target dengan `SELECT status FROM bookings WHERE id = $1 FOR UPDATE` guna mencegah lost update dan pelanggaran transisi status saat request bersamaan.

### 5.2 ACID

- **Atomicity:** Insert/update booking dan pencatatan `booking_logs` dijalankan dalam satu transaksi PostgreSQL. Jika insert log gagal, seluruh transaksi booking wajib rollback.
- **Consistency:** Validasi service, check constraint (`start_time < end_time`, `status`), unique constraint, dan foreign key menjaga integritas data.
- **Isolation:** Row-level locking (`FOR UPDATE`) pada `rooms` dan `bookings` mencegah dua transaksi mengambil slot konflik atau mengubah status yang sama secara bersamaan.
- **Durability:** Perubahan yang berhasil di-commit disimpan secara persisten oleh PostgreSQL.

### 5.3 Idempotency-Key *(Future Scope — Belum Diimplementasikan)*

Untuk `POST /bookings`, pada tahap pengembangan lanjutan client direncanakan mengirim header `Idempotency-Key` yang diikat ke user dan request. Rancangan tabel `idempotency_keys` memiliki kolom utama:

- `id`
- `user_id`
- `idempotency_key`
- `request_hash`
- `response_status`
- `response_body`
- `created_at`

Constraint yang direncanakan: `UNIQUE(user_id, idempotency_key)`. Key sama dengan payload sama mengembalikan hasil yang tersimpan; key sama dengan payload berbeda ditolak sebagai konflik. Penanganan request bersamaan harus mengandalkan unique constraint dan transaksi, bukan hanya pola SELECT lalu INSERT.

## 6. Rancangan notifikasi asynchronous: Transactional Outbox + Redis/Asynq *(Future Scope — Belum Diimplementasikan)*

Notifikasi asynchronous adalah pengembangan lanjutan yang dikaitkan dengan materi Modul 11 dan belum menjadi bagian dari implementasi aktif saat ini.

Komponen yang direncanakan:

- **Transactional Outbox:** menyimpan event notifikasi bersama perubahan booking dalam transaksi PostgreSQL yang sama.
- **Outbox Relay:** membaca event outbox yang belum diteruskan dan membuat job antrean.
- **Redis:** backend antrean yang digunakan Asynq.
- **Asynq:** library/task queue Go untuk mengelola job asynchronous dan retry.
- **Notification Worker:** mengambil job dan menjalankan pengiriman notifikasi.
- **Consumer idempotency:** mencegah efek samping ganda saat job diproses ulang.

Rancangan tabel `outbox_events` masih perlu difinalisasi sebelum implementasi. Kolom awal yang direncanakan: `id`, `event_type`, `payload`, `created_at`, `processed_at`. Detail status, retry, locking relay, dan kebijakan retensi belum ditetapkan dan tidak boleh ditebak oleh AI.

Alur yang direncanakan:

```text
Booking Service
    |
    v
PostgreSQL Transaction
  - perubahan booking
  - insert outbox event
    |
    v
Outbox Relay
    |
    v
Asynq Queue <--> Redis
    |
    v
Notification Worker
    |
    v
Notification Delivery
```

Jangan memasang dependency Redis/Asynq, membuat worker, atau menambahkan tabel outbox sebelum tim menyetujui masuk ke tahap pengembangan notifikasi asynchronous.

## 7. Batas ruang lingkup dan hal yang belum diputuskan

Keputusan yang sudah ditetapkan:

- **Driver PostgreSQL:** `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib`.
- **Desain relasi approver-ruangan:** tabel `approver_rooms` dengan kolom `approver_id`, `room_id`, dan `UNIQUE (approver_id, room_id)`. Implementasi migration `004` masih perlu dibuat dan diverifikasi.
- **Matriks transisi status booking & hak akses role (`student`, `approver`, `admin`):** ditetapkan pada Bagian 1 dan Bagian 4.

Hal berikut masih belum ditentukan/diimplementasikan dan harus disepakati sebelum implementasi terkait:

- Strategi migration runner otomatis (saat ini file SQL di `migrations/` dijalankan secara manual).
- Desain final autentikasi (JWT/session, endpoint login/register, hashing password) dan authorization middleware di `middleware/`.
- Pagination, filtering, dan sorting untuk `GET /rooms` serta perluasan filter/pagination pada `GET /bookings`.
- Kebijakan validasi/format waktu dan timezone API secara menyeluruh di luar format RFC3339 pada parameter query.
- Bentuk final objek/detail `errors` terstruktur pada response envelope untuk semua kondisi.
- Implementasi `Idempotency-Key` dan tabel `idempotency_keys`.
- Provider/channel pengiriman notifikasi serta detail schema, retry, dan retensi `outbox_events`

# ARCHITECTURE.md — Rancangan Sistem, Database, dan Keandalan CampusSpace

## 1. Ringkasan sistem

CampusSpace adalah backend REST API untuk reservasi fasilitas kampus, seperti laboratorium komputer, ruang seminar, aula, dan ruang organisasi mahasiswa. Sistem dirancang untuk membantu civitas akademika melihat ruangan, mengajukan reservasi, dan memproses persetujuan oleh pengelola ruangan.

Tiga role dalam rancangan dan aturan bisnis sistem:

- `student`: peminjam.
- `approver`: pengelola ruangan.
- `admin`: administrator.

Hak akses bisnis per role:

- **Student:**
  - Melihat ruangan dan memeriksa ketersediaan ruangan.
  - Mengajukan booking baru (hanya `student` berstatus aktif `is_active = TRUE`).
  - Membatalkan (`cancelled`) booking miliknya sendiri yang masih berstatus `pending` atau `approved`.
  - Melihat riwayat booking.
  - Tidak boleh menyetujui (`approved`), menolak (`rejected`), atau membatalkan booking milik user lain.
- **Approver:**
  - Melihat pengajuan booking pada ruangan yang menjadi tanggung jawabnya (terdaftar di `approver_rooms`).
  - Menyetujui (`approved`) atau menolak (`rejected`) booking berstatus `pending` pada ruangan yang ditugaskan kepadanya, disertai catatan/alasan opsional.
  - Tidak boleh membatalkan (`cancelled`) booking maupun memutuskan booking untuk ruangan di luar penugasannya.
- **Admin:**
  - Mengelola data ruangan, mengelola akun/role pengguna, dan memantau data booking.
  - Tidak boleh mengajukan booking maupun melakukan `approved`, `rejected`, atau `cancelled` terhadap booking.

**Catatan autentikasi & otorisasi:** Autentikasi (login/token/session) dan authorization middleware di layer HTTP belum diimplementasikan (`middleware/` masih belum memiliki implementasi). Aturan hak akses di atas merupakan aturan bisnis yang menjadi target implementasi pada layer service. Untuk versi produksi, identitas aktor harus diambil dari authenticated request context, bukan dipercaya secara bebas dari request body.

## 2. Teknologi dan arsitektur

| Area           | Teknologi/status                                                                     |
| -------------- | ------------------------------------------------------------------------------------ |
| Bahasa         | Go                                                                                   |
| HTTP framework | Fiber v2                                                                             |
| Database       | PostgreSQL melalui `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib` |
| Arsitektur     | Clean Architecture mengikuti Modul 4                                                 |
| Queue/cache    | Redis + Asynq (future scope / rancangan notifikasi asynchronous)                     |

Struktur package:

```text
app/model       Entity dan DTO
app/repository  Interface dan implementasi akses database
app/service     Alur aplikasi dan business rules
config          Konfigurasi aplikasi, environment, logger
database        Koneksi PostgreSQL dan connection pool
helper          Fungsi bantuan (belum diisi pada tahap saat ini)
middleware      Middleware aplikasi (belum diisi; auth adalah future scope)
route           Registrasi endpoint HTTP (room.go aktif; booking.go dalam pengembangan)
migrations      SQL migration
main.go         Dependency wiring dan entry point
```

Jangan menambahkan `controller`, `handler`, `usecase`, atau `delivery`; struktur mengikuti Modul 4.

**Status implementasi package saat ini:**

- **Sudah diimplementasikan dan terverifikasi (`main`):** `config`, `database`, serta modul Rooms (`app/model/room.go`, `app/repository/room_repository.go`, `app/service/room_service.go`, `route/room.go`) yang sudah terhubung di `main.go` dan `migrations/001_create_rooms_table.sql`.
- **Sedang dikembangkan / dalam tahap review perbaikan:** modul Users dan Bookings (`app/model/user.go`, `app/model/booking.go`, `app/repository/user_repository.go`, `app/repository/booking_repository.go`, `app/service/booking_service.go`, `route/booking.go`, `migrations/002`, `migrations/003`, serta migration `004` yang direncanakan untuk `approver_rooms`). Endpoint Booking belum di-wire di `main.go` dan masih menjalani perbaikan sebelum dianggap selesai.
- **Belum diimplementasikan (Placeholder / Future Scope):** `helper/response.go` (response envelope masih berupa helper lokal di `route/room.go`), `route/route.go`, `middleware/` (autentikasi/otorisasi), `Idempotency-Key`, Transactional Outbox, Redis, Asynq, dan Notification Worker.

## 3. Skema database inti

Skema database terdiri atas tabel inti `users`, `rooms`, `bookings`, `booking_logs`, serta tabel relasi `approver_rooms`. Tabel `approver_rooms` sudah ditetapkan sebagai desain relasi penugasan approver, tetapi migration dan implementasinya masih perlu dibuat serta diverifikasi.

### 3.1 `users` *(dalam pengembangan / perbaikan skema)*

| Kolom             | Tipe rancangan | Keterangan                                    |
| ----------------- | -------------- | --------------------------------------------- |
| `id`            | BIGSERIAL      | Primary key                                   |
| `name`          | VARCHAR(100)   | Nama pengguna, NOT NULL                       |
| `email`         | VARCHAR(100)   | Email, UNIQUE dan NOT NULL                    |
| `password_hash` | TEXT           | Password hash, NOT NULL (tidak diekspos JSON) |
| `role`          | VARCHAR(20)    | `student`, `approver`, `admin`          |
| `is_active`     | BOOLEAN        | Status aktif user, NOT NULL, default TRUE     |
| `created_at`    | TIMESTAMPTZ    | Default waktu pembuatan                       |

### 3.2 `rooms` *(sudah diimplementasikan & dimigrasi)*

| Kolom          | Tipe rancangan | Keterangan                     |
| -------------- | -------------- | ------------------------------ |
| `id`         | BIGSERIAL      | Primary key                    |
| `code`       | VARCHAR(50)    | Kode unik, UNIQUE dan NOT NULL |
| `name`       | VARCHAR(100)   | Nama ruangan, NOT NULL         |
| `building`   | VARCHAR(100)   | Gedung/lokasi, NOT NULL        |
| `capacity`   | INTEGER        | NOT NULL dan harus > 0         |
| `is_active`  | BOOLEAN        | NOT NULL, default TRUE         |
| `created_at` | TIMESTAMPTZ    | Default waktu pembuatan        |

`DELETE /rooms/:id` diimplementasikan sebagai soft delete: mengubah `is_active` menjadi `false`, bukan menghapus row secara fisik, agar relasi/histori booking tetap terjaga.

### 3.3 `bookings` *(dalam pengembangan / perbaikan skema)*

| Kolom          | Tipe rancangan | Keterangan                                                                   |
| -------------- | -------------- | ---------------------------------------------------------------------------- |
| `id`         | BIGSERIAL      | Primary key                                                                  |
| `user_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL                                       |
| `room_id`    | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                                       |
| `purpose`    | TEXT           | Tujuan reservasi, NOT NULL                                                   |
| `start_time` | TIMESTAMPTZ    | Waktu mulai, NOT NULL                                                        |
| `end_time`   | TIMESTAMPTZ    | Waktu selesai, NOT NULL (`start_time < end_time`)                          |
| `status`     | VARCHAR(20)    | `pending`, `approved`, `rejected`, `cancelled` (default `pending`) |
| `created_at` | TIMESTAMPTZ    | Waktu pengajuan, NOT NULL, default `CURRENT_TIMESTAMP`                     |
| `updated_at` | TIMESTAMPTZ    | Waktu pembaruan terakhir, NOT NULL, default `CURRENT_TIMESTAMP`            |

### 3.4 `booking_logs` *(dalam pengembangan)*

| Kolom           | Tipe rancangan | Keterangan                                       |
| --------------- | -------------- | ------------------------------------------------ |
| `id`          | BIGSERIAL      | Primary key                                      |
| `booking_id`  | BIGINT         | Foreign key ke `bookings(id)`, NOT NULL        |
| `actor_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL           |
| `from_status` | VARCHAR(20)    | Status sebelumnya; `NULL` untuk initial status |
| `to_status`   | VARCHAR(20)    | Status setelah perubahan, NOT NULL               |
| `note`        | TEXT           | Catatan/alasan perubahan                         |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL                       |

### 3.5 `approver_rooms` *(rancangan; migration belum dibuat)*

| Kolom           | Tipe rancangan | Keterangan                                                |
| --------------- | -------------- | --------------------------------------------------------- |
| `id`          | BIGSERIAL      | Primary key                                               |
| `approver_id` | BIGINT         | Foreign key ke `users(id)`, NOT NULL                    |
| `room_id`     | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                    |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL, default `CURRENT_TIMESTAMP` |

Constraint: `UNIQUE (approver_id, room_id)` untuk mencegah duplikasi penugasan approver pada ruangan yang sama. Validasi bahwa `approver_id` memiliki role `approver` dilakukan pada layer aplikasi/service. Migration yang direncanakan: `004_create_approver_rooms_table.sql`.

`from_status` dan `to_status` pada `booking_logs` dibatasi pada nilai `pending`, `approved`, `rejected`, atau `cancelled`; `from_status` boleh `NULL` hanya untuk pencatatan status awal booking.

### Relasi Tabel

- `users.id` → `bookings.user_id`
- `rooms.id` → `bookings.room_id`
- `bookings.id` → `booking_logs.booking_id`
- `users.id` → `booking_logs.actor_id`
- `users.id` → `approver_rooms.approver_id`
- `rooms.id` → `approver_rooms.room_id`

## 4. Aturan bisnis reservasi

Aturan bisnis modul booking:

1. `start_time` harus lebih awal daripada `end_time` (`start_time < end_time`), dan keduanya tidak boleh bernilai kosong/zero.
2. Hanya `student` berstatus aktif (`is_active = TRUE`) yang boleh mengajukan booking baru; `admin` dan `approver` tidak boleh mengajukan booking.
3. Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
4. Booking baru wajib dibuat dengan status awal `pending` dan selalu mencatat initial log pada `booking_logs` (`from_status = NULL`, `to_status = 'pending'`).
5. Status `pending` dan `approved` menahan slot waktu ruangan.
6. Status `rejected` dan `cancelled` tidak menahan slot waktu ruangan.
7. Setiap perubahan status yang valid wajib menghasilkan tepat satu catatan baru di `booking_logs` dan memperbarui `bookings.updated_at` secara atomik dalam satu transaksi.

### Matriks transisi status booking

Status yang diperbolehkan: `pending`, `approved`, `rejected`, `cancelled`.

| Dari Status   | Ke Status     | Diperbolehkan | Aktor yang Berwenang                                               |
| ------------- | ------------- | ------------- | ------------------------------------------------------------------ |
| `pending`   | `approved`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `rejected`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `approved`  | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `rejected`  | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| `cancelled` | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| *(lainnya)* | *(lainnya)* | Tidak         | Ditolak (`409 Conflict`)                                         |

`admin` tidak boleh melakukan `approved`, `rejected`, maupun `cancelled`. Untuk mengajukan ulang slot setelah `rejected` atau `cancelled`, pengguna harus membuat booking baru.

### Rumus konflik jadwal

Dua rentang waktu bertabrakan jika:

```text
existing.start_time < requested.end_time
AND existing.end_time > requested.start_time
```

Rentang yang tepat bersentuhan pada batas waktu (misalnya `09:00–10:00` dan `10:00–11:00`) tidak dianggap overlap menurut rumus tersebut sehingga tetap diperbolehkan.

## 5. Transaksi, race condition, dan idempotency booking

*Status bagian ini: mekanisme transaksi dan row-level locking pada repository booking sedang dalam tahap penyempurnaan dan verifikasi; endpoint Booking belum aktif di `main.go`. `Idempotency-Key` masih merupakan rancangan pengembangan lanjutan (future scope).*

### 5.1 Pencegahan double booking

Race condition dapat terjadi jika dua request memeriksa slot yang sama sebelum salah satunya menyimpan booking. Rancangan yang digunakan adalah transaksi PostgreSQL dengan pessimistic locking pada baris ruangan (`rooms`) dan pembacaan status aktif user:

```sql
BEGIN;

-- 1. Verifikasi user aktif dan role student
SELECT is_active, role
FROM users
WHERE id = $1
FOR SHARE;

-- 2. Kunci row ruangan agar request concurrent pada room yang sama berjalan serial
SELECT id, is_active
FROM rooms
WHERE id = $2
FOR UPDATE;

-- 3. Periksa booking berstatus pending/approved dengan rentang waktu overlap

-- 4. Jika ruangan aktif dan tidak ada konflik, INSERT booking (status = 'pending')
--    dan INSERT initial log ke booking_logs dalam transaksi yang sama

COMMIT;
```

Semua jalur pembuatan booking untuk `room_id` yang sama wajib mengunci baris ruangan yang sama sebelum memeriksa konflik. Pengecekan di luar transaksi tidak memenuhi rancangan ini.

Untuk perubahan status (`UpdateStatus`), transaksi mengunci baris booking target dengan `SELECT status FROM bookings WHERE id = $1 FOR UPDATE` guna mencegah lost update dan pelanggaran transisi status saat request bersamaan.

### 5.2 ACID

- **Atomicity:** Insert/update booking dan pencatatan `booking_logs` dijalankan dalam satu transaksi PostgreSQL. Jika insert log gagal, seluruh transaksi booking wajib rollback.
- **Consistency:** Validasi service, check constraint (`start_time < end_time`, `status`), unique constraint, dan foreign key menjaga integritas data.
- **Isolation:** Row-level locking (`FOR UPDATE`) pada `rooms` dan `bookings` mencegah dua transaksi mengambil slot konflik atau mengubah status yang sama secara bersamaan.
- **Durability:** Perubahan yang berhasil di-commit disimpan secara persisten oleh PostgreSQL.

### 5.3 Idempotency-Key *(Future Scope — Belum Diimplementasikan)*

Untuk `POST /bookings`, pada tahap pengembangan lanjutan client direncanakan mengirim header `Idempotency-Key` yang diikat ke user dan request. Rancangan tabel `idempotency_keys` memiliki kolom utama:

- `id`
- `user_id`
- `idempotency_key`
- `request_hash`
- `response_status`
- `response_body`
- `created_at`

Constraint yang direncanakan: `UNIQUE(user_id, idempotency_key)`. Key sama dengan payload sama mengembalikan hasil yang tersimpan; key sama dengan payload berbeda ditolak sebagai konflik. Penanganan request bersamaan harus mengandalkan unique constraint dan transaksi, bukan hanya pola SELECT lalu INSERT.

## 6. Rancangan notifikasi asynchronous: Transactional Outbox + Redis/Asynq *(Future Scope — Belum Diimplementasikan)*

Notifikasi asynchronous adalah pengembangan lanjutan yang dikaitkan dengan materi Modul 11 dan belum menjadi bagian dari implementasi aktif saat ini.

Komponen yang direncanakan:

- **Transactional Outbox:** menyimpan event notifikasi bersama perubahan booking dalam transaksi PostgreSQL yang sama.
- **Outbox Relay:** membaca event outbox yang belum diteruskan dan membuat job antrean.
- **Redis:** backend antrean yang digunakan Asynq.
- **Asynq:** library/task queue Go untuk mengelola job asynchronous dan retry.
- **Notification Worker:** mengambil job dan menjalankan pengiriman notifikasi.
- **Consumer idempotency:** mencegah efek samping ganda saat job diproses ulang.

Rancangan tabel `outbox_events` masih perlu difinalisasi sebelum implementasi. Kolom awal yang direncanakan: `id`, `event_type`, `payload`, `created_at`, `processed_at`. Detail status, retry, locking relay, dan kebijakan retensi belum ditetapkan dan tidak boleh ditebak oleh AI.

Alur yang direncanakan:

```text
Booking Service
    |
    v
PostgreSQL Transaction
  - perubahan booking
  - insert outbox event
    |
    v
Outbox Relay
    |
    v
Asynq Queue <--> Redis
    |
    v
Notification Worker
    |
    v
Notification Delivery
```

Jangan memasang dependency Redis/Asynq, membuat worker, atau menambahkan tabel outbox sebelum tim menyetujui masuk ke tahap pengembangan notifikasi asynchronous.

## 7. Batas ruang lingkup dan hal yang belum diputuskan

Keputusan yang sudah ditetapkan:

- **Driver PostgreSQL:** `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib`.
- **Desain relasi approver-ruangan:** tabel `approver_rooms` dengan kolom `approver_id`, `room_id`, dan `UNIQUE (approver_id, room_id)`. Implementasi migration `004` masih perlu dibuat dan diverifikasi.
- **Matriks transisi status booking & hak akses role (`student`, `approver`, `admin`):** ditetapkan pada Bagian 1 dan Bagian 4.

Hal berikut masih belum ditentukan/diimplementasikan dan harus disepakati sebelum implementasi terkait:

- Strategi migration runner otomatis (saat ini file SQL di `migrations/` dijalankan secara manual).
- Desain final autentikasi (JWT/session, endpoint login/register, hashing password) dan authorization middleware di `middleware/`.
- Pagination, filtering, dan sorting untuk `GET /rooms` serta perluasan filter/pagination pada `GET /bookings`.
- Kebijakan validasi/format waktu dan timezone API secara menyeluruh di luar format RFC3339 pada parameter query.
- Bentuk final objek/detail `errors` terstruktur pada response envelope untuk semua kondisi.
- Implementasi `Idempotency-Key` dan tabel `idempotency_keys`.
- Provider/channel pengiriman notifikasi serta detail schema, retry, dan retensi `outbox_events`

# ARCHITECTURE.md — Rancangan Sistem, Database, dan Keandalan CampusSpace

## 1. Ringkasan sistem

CampusSpace adalah backend REST API untuk reservasi fasilitas kampus, seperti laboratorium komputer, ruang seminar, aula, dan ruang organisasi mahasiswa. Sistem dirancang untuk membantu civitas akademika melihat ruangan, mengajukan reservasi, dan memproses persetujuan oleh pengelola ruangan.

Tiga role dalam rancangan dan aturan bisnis sistem:

- `student`: peminjam.
- `approver`: pengelola ruangan.
- `admin`: administrator.

Hak akses bisnis per role:

- **Student:**
  - Melihat ruangan dan memeriksa ketersediaan ruangan.
  - Mengajukan booking baru (hanya `student` berstatus aktif `is_active = TRUE`).
  - Membatalkan (`cancelled`) booking miliknya sendiri yang masih berstatus `pending` atau `approved`.
  - Melihat riwayat booking.
  - Tidak boleh menyetujui (`approved`), menolak (`rejected`), atau membatalkan booking milik user lain.
- **Approver:**
  - Melihat pengajuan booking pada ruangan yang menjadi tanggung jawabnya (terdaftar di `approver_rooms`).
  - Menyetujui (`approved`) atau menolak (`rejected`) booking berstatus `pending` pada ruangan yang ditugaskan kepadanya, disertai catatan/alasan opsional.
  - Tidak boleh membatalkan (`cancelled`) booking maupun memutuskan booking untuk ruangan di luar penugasannya.
- **Admin:**
  - Mengelola data ruangan, mengelola akun/role pengguna, dan memantau data booking.
  - Tidak boleh mengajukan booking maupun melakukan `approved`, `rejected`, atau `cancelled` terhadap booking.

**Catatan autentikasi & otorisasi:** Autentikasi (login/token/session) dan authorization middleware di layer HTTP belum diimplementasikan (`middleware/` masih belum memiliki implementasi). Validasi aturan hak akses bisnis disiapkan pada layer service, dengan desain agar identitas aktor nantinya diambil dari authenticated request context, bukan dipercaya secara bebas dari request body pada versi produksi.

## 2. Teknologi dan arsitektur

| Area           | Teknologi/status                                                                     |
| -------------- | ------------------------------------------------------------------------------------ |
| Bahasa         | Go                                                                                   |
| HTTP framework | Fiber v2                                                                             |
| Database       | PostgreSQL melalui `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib` |
| Arsitektur     | Clean Architecture mengikuti Modul 4                                                 |
| Queue/cache    | Redis + Asynq (future scope / rancangan notifikasi asynchronous)                     |

Struktur package:

```text
app/model       Entity dan DTO
app/repository  Interface dan implementasi akses database
app/service     Alur aplikasi dan business rules
config          Konfigurasi aplikasi, environment, logger
database        Koneksi PostgreSQL dan connection pool
helper          Fungsi bantuan (belum diisi pada tahap saat ini)
middleware      Middleware aplikasi (belum diisi; auth adalah future scope)
route           Registrasi endpoint HTTP (room.go aktif; booking.go dalam pengembangan)
migrations      SQL migration
main.go         Dependency wiring dan entry point
```

Jangan menambahkan `controller`, `handler`, `usecase`, atau `delivery`; struktur mengikuti Modul 4.

**Status implementasi package saat ini:**

- **Sudah diimplementasikan dan terverifikasi (`main`):** `config`, `database`, serta modul Rooms (`app/model/room.go`, `app/repository/room_repository.go`, `app/service/room_service.go`, `route/room.go`) yang sudah terhubung di `main.go` dan `migrations/001_create_rooms_table.sql`.
- **Sedang dikembangkan / dalam tahap review perbaikan:** modul Users dan Bookings (`app/model/user.go`, `app/model/booking.go`, `app/repository/user_repository.go`, `app/repository/booking_repository.go`, `app/service/booking_service.go`, `route/booking.go`, `migrations/002`, `migrations/003`, serta rencana `approver_rooms`). Endpoint Booking belum di-wire di `main.go` dan masih menjalani perbaikan sebelum dianggap selesai.
- **Belum diimplementasikan (Placeholder / Future Scope):** `helper/response.go` (response envelope masih berupa helper lokal di `route/room.go`), `route/route.go`, `middleware/` (autentikasi/otorisasi), `Idempotency-Key`, Transactional Outbox, Redis, Asynq, dan Notification Worker.

## 3. Skema database inti

Skema database terdiri atas tabel inti `users`, `rooms`, `bookings`, `booking_logs`, serta tabel relasi `approver_rooms`.

### 3.1 `users` *(dalam pengembangan / perbaikan skema)*

| Kolom             | Tipe rancangan | Keterangan                                    |
| ----------------- | -------------- | --------------------------------------------- |
| `id`            | BIGSERIAL      | Primary key                                   |
| `name`          | VARCHAR(100)   | Nama pengguna, NOT NULL                       |
| `email`         | VARCHAR(100)   | Email, UNIQUE dan NOT NULL                    |
| `password_hash` | TEXT           | Password hash, NOT NULL (tidak diekspos JSON) |
| `role`          | VARCHAR(20)    | `student`, `approver`, `admin`          |
| `is_active`     | BOOLEAN        | Status aktif user, NOT NULL, default TRUE     |
| `created_at`    | TIMESTAMPTZ    | Default waktu pembuatan                       |

### 3.2 `rooms` *(sudah diimplementasikan & dimigrasi)*

| Kolom          | Tipe rancangan | Keterangan                     |
| -------------- | -------------- | ------------------------------ |
| `id`         | BIGSERIAL      | Primary key                    |
| `code`       | VARCHAR(50)    | Kode unik, UNIQUE dan NOT NULL |
| `name`       | VARCHAR(100)   | Nama ruangan, NOT NULL         |
| `building`   | VARCHAR(100)   | Gedung/lokasi, NOT NULL        |
| `capacity`   | INTEGER        | NOT NULL dan harus > 0         |
| `is_active`  | BOOLEAN        | NOT NULL, default TRUE         |
| `created_at` | TIMESTAMPTZ    | Default waktu pembuatan        |

`DELETE /rooms/:id` diimplementasikan sebagai soft delete: mengubah `is_active` menjadi `false`, bukan menghapus row secara fisik, agar relasi/histori booking tetap terjaga.

### 3.3 `bookings` *(dalam pengembangan / perbaikan skema)*

| Kolom          | Tipe rancangan | Keterangan                                                                   |
| -------------- | -------------- | ---------------------------------------------------------------------------- |
| `id`         | BIGSERIAL      | Primary key                                                                  |
| `user_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL                                       |
| `room_id`    | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                                       |
| `purpose`    | TEXT           | Tujuan reservasi, NOT NULL                                                   |
| `start_time` | TIMESTAMPTZ    | Waktu mulai, NOT NULL                                                        |
| `end_time`   | TIMESTAMPTZ    | Waktu selesai, NOT NULL (`start_time < end_time`)                          |
| `status`     | VARCHAR(20)    | `pending`, `approved`, `rejected`, `cancelled` (default `pending`) |
| `created_at` | TIMESTAMPTZ    | Waktu pengajuan, NOT NULL, default `CURRENT_TIMESTAMP`                     |
| `updated_at` | TIMESTAMPTZ    | Waktu pembaruan terakhir, NOT NULL, default `CURRENT_TIMESTAMP`            |

### 3.4 `booking_logs` *(dalam pengembangan)*

| Kolom           | Tipe rancangan | Keterangan                                       |
| --------------- | -------------- | ------------------------------------------------ |
| `id`          | BIGSERIAL      | Primary key                                      |
| `booking_id`  | BIGINT         | Foreign key ke `bookings(id)`, NOT NULL        |
| `actor_id`    | BIGINT         | Foreign key ke `users(id)`, NOT NULL           |
| `from_status` | VARCHAR(20)    | Status sebelumnya; `NULL` untuk initial status |
| `to_status`   | VARCHAR(20)    | Status setelah perubahan, NOT NULL               |
| `note`        | TEXT           | Catatan/alasan perubahan                         |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL                       |

### 3.5 `approver_rooms` *(rancangan perbaikan untuk penugasan approver)*

| Kolom           | Tipe rancangan | Keterangan                                                |
| --------------- | -------------- | --------------------------------------------------------- |
| `id`          | BIGSERIAL      | Primary key                                               |
| `approver_id` | BIGINT         | Foreign key ke `users(id)`, NOT NULL                    |
| `room_id`     | BIGINT         | Foreign key ke `rooms(id)`, NOT NULL                    |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan, NOT NULL, default `CURRENT_TIMESTAMP` |

Constraint: `UNIQUE (approver_id, room_id)` untuk mencegah duplikasi penugasan approver pada ruangan yang sama. Validasi bahwa `approver_id` memiliki role `approver` dilakukan pada layer aplikasi/service.

### Relasi Tabel

- `users.id` → `bookings.user_id`
- `rooms.id` → `bookings.room_id`
- `bookings.id` → `booking_logs.booking_id`
- `users.id` → `booking_logs.actor_id`
- `users.id` → `approver_rooms.approver_id`
- `rooms.id` → `approver_rooms.room_id`

## 4. Aturan bisnis reservasi

Aturan bisnis modul booking:

1. `start_time` harus lebih awal daripada `end_time` (`start_time < end_time`), dan keduanya tidak boleh bernilai kosong/zero.
2. Hanya `student` berstatus aktif (`is_active = TRUE`) yang boleh mengajukan booking baru; `admin` dan `approver` tidak boleh mengajukan booking.
3. Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
4. Booking baru wajib dibuat dengan status awal `pending` dan selalu mencatat initial log pada `booking_logs` (`from_status = NULL`, `to_status = 'pending'`).
5. Status `pending` dan `approved` menahan slot waktu ruangan.
6. Status `rejected` dan `cancelled` tidak menahan slot waktu ruangan.
7. Setiap perubahan status yang valid wajib menghasilkan tepat satu catatan baru di `booking_logs` dan memperbarui `bookings.updated_at` secara atomik dalam satu transaksi.

### Matriks transisi status booking

Status yang diperbolehkan: `pending`, `approved`, `rejected`, `cancelled`.

| Dari Status   | Ke Status     | Diperbolehkan | Aktor yang Berwenang                                               |
| ------------- | ------------- | ------------- | ------------------------------------------------------------------ |
| `pending`   | `approved`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `rejected`  | Ya            | `approver` aktif yang ditugaskan ke ruangan (`approver_rooms`) |
| `pending`   | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `approved`  | `cancelled` | Ya            | `student` aktif pemilik booking                                  |
| `rejected`  | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| `cancelled` | *(apa pun)* | Tidak         | Terminal — tidak boleh diaktifkan kembali                         |
| *(lainnya)* | *(lainnya)* | Tidak         | Ditolak (`409 Conflict`)                                         |

`admin` tidak boleh melakukan `approved`, `rejected`, maupun `cancelled`. Untuk mengajukan ulang slot setelah `rejected` atau `cancelled`, pengguna harus membuat booking baru.

### Rumus konflik jadwal

Dua rentang waktu bertabrakan jika:

```text
existing.start_time < requested.end_time
AND existing.end_time > requested.start_time
```

Rentang yang tepat bersentuhan pada batas waktu (misalnya `09:00–10:00` dan `10:00–11:00`) tidak dianggap overlap menurut rumus tersebut sehingga tetap diperbolehkan.

## 5. Transaksi, race condition, dan idempotency booking

*Status bagian ini: mekanisme transaksi dan row-level locking pada repository booking sedang dalam tahap penyempurnaan dan verifikasi; endpoint Booking belum aktif di `main.go`. `Idempotency-Key` masih merupakan rancangan pengembangan lanjutan (future scope).*

### 5.1 Pencegahan double booking

Race condition dapat terjadi jika dua request memeriksa slot yang sama sebelum salah satunya menyimpan booking. Rancangan yang digunakan adalah transaksi PostgreSQL dengan pessimistic locking pada baris ruangan (`rooms`) dan pembacaan status aktif user:

```sql
BEGIN;

-- 1. Verifikasi user aktif dan role student
SELECT is_active, role
FROM users
WHERE id = $1
FOR SHARE;

-- 2. Kunci row ruangan agar request concurrent pada room yang sama berjalan serial
SELECT id, is_active
FROM rooms
WHERE id = $2
FOR UPDATE;

-- 3. Periksa booking berstatus pending/approved dengan rentang waktu overlap

-- 4. Jika ruangan aktif dan tidak ada konflik, INSERT booking (status = 'pending')
--    dan INSERT initial log ke booking_logs dalam transaksi yang sama

COMMIT;
```

Semua jalur pembuatan booking untuk `room_id` yang sama wajib mengunci baris ruangan yang sama sebelum memeriksa konflik. Pengecekan di luar transaksi tidak memenuhi rancangan ini.

Untuk perubahan status (`UpdateStatus`), transaksi mengunci baris booking target dengan `SELECT status FROM bookings WHERE id = $1 FOR UPDATE` guna mencegah lost update dan pelanggaran transisi status saat request bersamaan.

### 5.2 ACID

- **Atomicity:** Insert/update booking dan pencatatan `booking_logs` dijalankan dalam satu transaksi PostgreSQL. Jika insert log gagal, seluruh transaksi booking wajib rollback.
- **Consistency:** Validasi service, check constraint (`start_time < end_time`, `status`), unique constraint, dan foreign key menjaga integritas data.
- **Isolation:** Row-level locking (`FOR UPDATE`) pada `rooms` dan `bookings` mencegah dua transaksi mengambil slot konflik atau mengubah status yang sama secara bersamaan.
- **Durability:** Perubahan yang berhasil di-commit disimpan secara persisten oleh PostgreSQL.

### 5.3 Idempotency-Key *(Future Scope — Belum Diimplementasikan)*

Untuk `POST /bookings`, pada tahap pengembangan lanjutan client direncanakan mengirim header `Idempotency-Key` yang diikat ke user dan request. Rancangan tabel `idempotency_keys` memiliki kolom utama:

- `id`
- `user_id`
- `idempotency_key`
- `request_hash`
- `response_status`
- `response_body`
- `created_at`

Constraint yang direncanakan: `UNIQUE(user_id, idempotency_key)`. Key sama dengan payload sama mengembalikan hasil yang tersimpan; key sama dengan payload berbeda ditolak sebagai konflik. Penanganan request bersamaan harus mengandalkan unique constraint dan transaksi, bukan hanya pola SELECT lalu INSERT.

## 6. Rancangan notifikasi asynchronous: Transactional Outbox + Redis/Asynq *(Future Scope — Belum Diimplementasikan)*

Notifikasi asynchronous adalah pengembangan lanjutan yang dikaitkan dengan materi Modul 11 dan belum menjadi bagian dari implementasi aktif saat ini.

Komponen yang direncanakan:

- **Transactional Outbox:** menyimpan event notifikasi bersama perubahan booking dalam transaksi PostgreSQL yang sama.
- **Outbox Relay:** membaca event outbox yang belum diteruskan dan membuat job antrean.
- **Redis:** backend antrean yang digunakan Asynq.
- **Asynq:** library/task queue Go untuk mengelola job asynchronous dan retry.
- **Notification Worker:** mengambil job dan menjalankan pengiriman notifikasi.
- **Consumer idempotency:** mencegah efek samping ganda saat job diproses ulang.

Rancangan tabel `outbox_events` masih perlu difinalisasi sebelum implementasi. Kolom awal yang direncanakan: `id`, `event_type`, `payload`, `created_at`, `processed_at`. Detail status, retry, locking relay, dan kebijakan retensi belum ditetapkan dan tidak boleh ditebak oleh AI.

Alur yang direncanakan:

```text
Booking Service
    |
    v
PostgreSQL Transaction
  - perubahan booking
  - insert outbox event
    |
    v
Outbox Relay
    |
    v
Asynq Queue <--> Redis
    |
    v
Notification Worker
    |
    v
Notification Delivery
```

Jangan memasang dependency Redis/Asynq, membuat worker, atau menambahkan tabel outbox sebelum tim menyetujui masuk ke tahap pengembangan notifikasi asynchronous.

## 7. Batas ruang lingkup dan hal yang belum diputuskan

Keputusan yang sudah ditetapkan:

- **Driver PostgreSQL:** `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib`.
- **Relasi approver-ruangan:** tabel `approver_rooms` (`approver_id`, `room_id`, `UNIQUE (approver_id, room_id)`).
- **Matriks transisi status booking & hak akses role (`student`, `approver`, `admin`):** ditetapkan pada Bagian 1 dan Bagian 4.

Hal berikut masih belum ditentukan/diimplementasikan dan harus disepakati sebelum implementasi terkait:

- Strategi migration runner otomatis (saat ini file SQL di `migrations/` dijalankan secara manual).
- Desain final autentikasi (JWT/session, endpoint login/register, hashing password) dan authorization middleware di `middleware/`.
- Pagination, filtering, dan sorting untuk `GET /rooms` serta perluasan filter/pagination pada `GET /bookings`.
- Kebijakan validasi/format waktu dan timezone API secara menyeluruh di luar format RFC3339 pada parameter query.
- Bentuk final objek/detail `errors` terstruktur pada response envelope untuk semua kondisi.
- Implementasi `Idempotency-Key` dan tabel `idempotency_keys`.
- Provider/channel pengiriman notifikasi serta detail schema, retry, dan retensi `outbox_events`.
