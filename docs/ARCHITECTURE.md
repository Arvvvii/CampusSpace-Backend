# ARCHITECTURE.md — Rancangan Sistem, Database, dan Keandalan CampusSpace

## 1. Ringkasan sistem

CampusSpace adalah backend REST API untuk reservasi fasilitas kampus, seperti laboratorium komputer, ruang seminar, aula, dan ruang organisasi mahasiswa. Sistem dirancang untuk membantu civitas akademika melihat ruangan, mengajukan reservasi, dan memproses persetujuan oleh pengelola ruangan.

Tiga role yang direncanakan:

- `student`: peminjam.
- `approver`: pengelola ruangan.
- `admin`: administrator.

Hak akses rinci:

- Student: melihat ruangan, memeriksa ketersediaan, mengajukan/membatalkan reservasi sendiri, dan melihat riwayat.
- Approver: melihat pengajuan pada ruangan yang menjadi tanggung jawabnya, lalu menyetujui atau menolak dengan alasan.
- Admin: mengelola data ruangan dan akun/role pengguna serta memantau reservasi.

Autentikasi dan otorisasi termasuk pengembangan lanjutan, bukan klaim fitur MVP yang sudah berjalan.

## 2. Teknologi dan arsitektur

| Area           | Teknologi/status                                                                     |
| -------------- | ------------------------------------------------------------------------------------ |
| Bahasa         | Go                                                                                   |
| HTTP framework | Fiber v2                                                                             |
| Database       | PostgreSQL melalui `database/sql` dengan driver `github.com/jackc/pgx/v5/stdlib` |
| Arsitektur     | Clean Architecture mengikuti Modul 4                                                 |
| Queue/cache    | Redis + Asynq, rancangan pengembangan notifikasi asynchronous                        |

Struktur package:

```text
app/model       Entity dan DTO
app/repository  Interface dan implementasi akses database
app/service     Alur aplikasi dan business rules
config          Konfigurasi aplikasi, environment, logger
database        Koneksi PostgreSQL dan connection pool
helper          Fungsi bantuan (response helper belum dipisahkan pada MVP)
middleware      Middleware aplikasi
route           Endpoint Rooms; registrasi utama saat ini dilakukan di `main.go`
migrations      SQL migration
main.go         Dependency wiring dan entry point
```

Jangan menambahkan `controller`, `handler`, `usecase`, atau `delivery`; struktur mengikuti Modul 4.

**Kondisi implementasi MVP:** package `app/model`, `app/repository`, `app/service`, `config`, `database`, `route`, serta `migrations` sudah digunakan untuk CRUD `rooms`. Folder `helper/` dan `middleware/` tersedia, tetapi belum memiliki implementasi fitur pada MVP. Response envelope saat ini didefinisikan sebagai helper lokal di `route/room.go`; pendaftaran route Rooms dan dependency wiring dilakukan langsung dari `main.go`. Pemisahan ke `helper/response.go` dan `route/route.go` belum dilakukan dan tidak boleh dianggap sudah tersedia.

## 3. Skema database inti

Empat tabel inti rancangan sistem adalah `users`, `rooms`, `bookings`, dan `booking_logs`. Untuk target implementasi awal, prioritaskan migration dan CRUD `rooms`. Kolom di bawah merupakan rancangan; tipe/constraint final harus diwujudkan dalam migration dan diselaraskan dengan kode.

### 3.1 `users`

| Kolom             | Tipe rancangan | Keterangan                           |
| ----------------- | -------------- | ------------------------------------ |
| `id`            | BIGSERIAL      | Primary key                          |
| `name`          | VARCHAR(100)   | Nama pengguna, NOT NULL              |
| `email`         | VARCHAR(100)   | Email, UNIQUE dan NOT NULL           |
| `password_hash` | TEXT           | Password hash, NOT NULL              |
| `role`          | VARCHAR(20)    | `student`, `approver`, `admin` |
| `created_at`    | TIMESTAMPTZ    | Default waktu pembuatan              |

### 3.2 `rooms` — prioritas implementasi

| Kolom          | Tipe rancangan | Keterangan              |
| -------------- | -------------- | ----------------------- |
| `id`         | BIGSERIAL      | Primary key             |
| `code`       | VARCHAR(50)    | Kode unik, NOT NULL     |
| `name`       | VARCHAR(100)   | Nama ruangan, NOT NULL  |
| `building`   | VARCHAR(100)   | Gedung/lokasi, NOT NULL |
| `capacity`   | INTEGER        | NOT NULL dan harus > 0  |
| `is_active`  | BOOLEAN        | NOT NULL, default TRUE  |
| `created_at` | TIMESTAMPTZ    | Default waktu pembuatan |

`DELETE /rooms/:id` dirancang sebagai soft delete: ubah `is_active` menjadi `false`, bukan menghapus row secara fisik, agar relasi/histori booking tetap aman.

### 3.3 `bookings` — pengembangan lanjutan

| Kolom          | Tipe rancangan | Keterangan                                             |
| -------------- | -------------- | ------------------------------------------------------ |
| `id`         | BIGSERIAL      | Primary key                                            |
| `user_id`    | BIGINT         | Foreign key ke `users(id)`                           |
| `room_id`    | BIGINT         | Foreign key ke `rooms(id)`                           |
| `purpose`    | TEXT           | Tujuan reservasi                                       |
| `start_time` | TIMESTAMPTZ    | Waktu mulai                                            |
| `end_time`   | TIMESTAMPTZ    | Waktu selesai                                          |
| `status`     | VARCHAR(20)    | `pending`, `approved`, `rejected`, `cancelled` |
| `created_at` | TIMESTAMPTZ    | Waktu pengajuan                                        |

### 3.4 `booking_logs` — pengembangan lanjutan

| Kolom           | Tipe rancangan | Keterangan                                      |
| --------------- | -------------- | ----------------------------------------------- |
| `id`          | BIGSERIAL      | Primary key                                     |
| `booking_id`  | BIGINT         | Foreign key ke `bookings(id)`                 |
| `actor_id`    | BIGINT         | Foreign key ke `users(id)`                    |
| `from_status` | VARCHAR(20)    | Status sebelumnya; dapat null untuk status awal |
| `to_status`   | VARCHAR(20)    | Status setelah perubahan                        |
| `note`        | TEXT           | Catatan/alasan                                  |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan                                |

Relasi:

- `users.id` → `bookings.user_id`
- `rooms.id` → `bookings.room_id`
- `bookings.id` → `booking_logs.booking_id`
- `users.id` → `booking_logs.actor_id`

## 4. Aturan bisnis reservasi

Aturan yang direncanakan untuk modul booking:

1. `start_time` harus lebih awal daripada `end_time`.
2. Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
3. Status `pending` dan `approved` menahan slot waktu.
4. Status `rejected` dan `cancelled` tidak menahan slot waktu.
5. Approver hanya dapat memutuskan reservasi untuk ruangan yang ditugaskan kepadanya.
6. Perubahan status dicatat di `booking_logs`.

Dua rentang waktu bertabrakan jika:

```text
existing.start_time < requested.end_time
AND existing.end_time > requested.start_time
```

Rentang yang tepat bersentuhan pada batas waktu (misalnya 09.00–10.00 dan 10.00–11.00) tidak dianggap overlap menurut rumus tersebut.

## 5. Transaksi, race condition, dan idempotency booking

Bagian ini adalah rancangan pengembangan lanjutan. Jangan mengklaim telah diimplementasikan sebelum kode dan test tersedia.

### 5.1 Pencegahan double booking

Race condition dapat terjadi jika dua request memeriksa slot yang sama sebelum salah satunya menyimpan booking. Rancangan yang dipilih adalah transaksi PostgreSQL dengan pessimistic locking pada row ruangan:

```sql
BEGIN;

SELECT id
FROM rooms
WHERE id = $1 AND is_active = TRUE
FOR UPDATE;

-- Periksa booking berstatus pending/approved
-- dengan rentang waktu yang overlap.

-- Jika tidak ada konflik, INSERT booking.

COMMIT;
```

Semua jalur pembuatan booking untuk `room_id` yang sama harus mengunci row ruangan yang sama sebelum memeriksa konflik. Pengecekan di luar transaksi tidak memenuhi rancangan ini.

### 5.2 ACID

- Atomicity: insert booking dan booking log dalam satu transaksi.
- Consistency: validasi, foreign key, dan constraint menjaga aturan data.
- Isolation: transaksi dan locking mencegah dua request mengambil slot yang sama.
- Durability: perubahan yang berhasil di-commit disimpan oleh PostgreSQL.

Jika insert booking log gagal, seluruh transaksi booking harus rollback.

### 5.3 Idempotency-Key

Untuk `POST /bookings`, client direncanakan mengirim header `Idempotency-Key`. Key diikat ke user dan request. Rancangan tabel `idempotency_keys` memiliki kolom utama:

- `id`
- `user_id`
- `idempotency_key`
- `request_hash`
- `response_status`
- `response_body`
- `created_at`

Constraint yang direncanakan: `UNIQUE(user_id, idempotency_key)`. Key sama dengan payload sama mengembalikan hasil yang tersimpan; key sama dengan payload berbeda ditolak sebagai konflik. Penanganan request bersamaan harus mengandalkan unique constraint dan transaksi, bukan hanya pola SELECT lalu INSERT.

## 6. Rancangan notifikasi asynchronous: Transactional Outbox + Redis/Asynq

Notifikasi asynchronous adalah pengembangan lanjutan yang dikaitkan dengan materi Modul 11.

Komponen:

- **Transactional Outbox:** menyimpan event notifikasi bersama perubahan booking dalam transaksi PostgreSQL yang sama.
- **Outbox Relay:** membaca event outbox yang belum diteruskan dan membuat job antrean.
- **Redis:** backend antrean yang digunakan Asynq.
- **Asynq:** library/task queue Go untuk mengelola job asynchronous dan retry.
- **Notification Worker:** mengambil job dan menjalankan pengiriman notifikasi.
- **Consumer idempotency:** mencegah efek samping ganda saat job diproses ulang.

Rancangan tabel `outbox_events` masih perlu difinalisasi sebelum implementasi. Kolom awal yang pernah direncanakan: `id`, `event_type`, `payload`, `created_at`, `processed_at`. Detail status, retry, locking relay, dan kebijakan retensi belum ditetapkan dan tidak boleh ditebak oleh AI.

Alur:

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

Jangan memasang dependency Redis/Asynq, membuat worker, atau menambahkan tabel outbox pada tahap CRUD rooms kecuali tim menyetujui perubahan scope.

## 7. Batas ruang lingkup dan hal yang belum diputuskan

Hal berikut belum ditentukan secara lengkap dalam dokumen sumber dan harus disepakati sebelum implementasi terkait:

- Driver PostgreSQL dan library akses database yang akan digunakan.
- Strategi migration runner dan perintah menjalankan migration.
- Detail konfigurasi environment selain variabel yang sudah ada di `.env.example`.
- Pagination, filtering, dan sorting untuk `GET /rooms`.
- Kebijakan validasi/format waktu dan timezone API secara menyeluruh.
- Bentuk final error detail untuk semua kondisi.
- Provider/channel pengiriman notifikasi.
- Detail schema, retry, dan retensi `outbox_events`.
- Kebijakan penugasan approver ke ruangan dan skema tabel penghubungnya.

AI dan anggota tim tidak boleh mengisi keputusan tersebut secara diam-diam. Catat keputusan tim di dokumen ini sebelum membuat implementasi yang bergantung padany

# ARCHITECTURE.md — Rancangan Sistem, Database, dan Keandalan CampusSpace

## 1. Ringkasan sistem

CampusSpace adalah backend REST API untuk reservasi fasilitas kampus, seperti laboratorium komputer, ruang seminar, aula, dan ruang organisasi mahasiswa. Sistem dirancang untuk membantu civitas akademika melihat ruangan, mengajukan reservasi, dan memproses persetujuan oleh pengelola ruangan.

Tiga role yang direncanakan:

- `student`: peminjam.
- `approver`: pengelola ruangan.
- `admin`: administrator.

Hak akses rinci:

- Student: melihat ruangan, memeriksa ketersediaan, mengajukan/membatalkan reservasi sendiri, dan melihat riwayat.
- Approver: melihat pengajuan pada ruangan yang menjadi tanggung jawabnya, lalu menyetujui atau menolak dengan alasan.
- Admin: mengelola data ruangan dan akun/role pengguna serta memantau reservasi.

Autentikasi dan otorisasi termasuk pengembangan lanjutan, bukan klaim fitur MVP yang sudah berjalan.

## 2. Teknologi dan arsitektur

| Area           | Teknologi/status                                              |
| -------------- | ------------------------------------------------------------- |
| Bahasa         | Go                                                            |
| HTTP framework | Fiber v2                                                      |
| Database       | PostgreSQL                                                    |
| Arsitektur     | Clean Architecture mengikuti Modul 4                          |
| Queue/cache    | Redis + Asynq, rancangan pengembangan notifikasi asynchronous |

Struktur package:

```text
app/model       Entity dan DTO
app/repository  Interface dan implementasi akses database
app/service     Alur aplikasi dan business rules
config          Konfigurasi aplikasi, environment, logger
database        Koneksi PostgreSQL dan connection pool
helper          Response envelope dan fungsi bantuan
middleware      Middleware aplikasi
route           Registrasi endpoint
migrations      SQL migration
main.go         Dependency wiring dan entry point
```

Jangan menambahkan `controller`, `handler`, `usecase`, atau `delivery`; struktur mengikuti Modul 4.

## 3. Skema database inti

Empat tabel inti rancangan sistem adalah `users`, `rooms`, `bookings`, dan `booking_logs`. Untuk target implementasi awal, prioritaskan migration dan CRUD `rooms`. Kolom di bawah merupakan rancangan; tipe/constraint final harus diwujudkan dalam migration dan diselaraskan dengan kode.

### 3.1 `users`

| Kolom             | Tipe rancangan | Keterangan                           |
| ----------------- | -------------- | ------------------------------------ |
| `id`            | BIGSERIAL      | Primary key                          |
| `name`          | VARCHAR(100)   | Nama pengguna, NOT NULL              |
| `email`         | VARCHAR(100)   | Email, UNIQUE dan NOT NULL           |
| `password_hash` | TEXT           | Password hash, NOT NULL              |
| `role`          | VARCHAR(20)    | `student`, `approver`, `admin` |
| `created_at`    | TIMESTAMPTZ    | Default waktu pembuatan              |

### 3.2 `rooms` — prioritas implementasi

| Kolom          | Tipe rancangan | Keterangan              |
| -------------- | -------------- | ----------------------- |
| `id`         | BIGSERIAL      | Primary key             |
| `code`       | VARCHAR(50)    | Kode unik, NOT NULL     |
| `name`       | VARCHAR(100)   | Nama ruangan, NOT NULL  |
| `building`   | VARCHAR(100)   | Gedung/lokasi, NOT NULL |
| `capacity`   | INTEGER        | NOT NULL dan harus > 0  |
| `is_active`  | BOOLEAN        | NOT NULL, default TRUE  |
| `created_at` | TIMESTAMPTZ    | Default waktu pembuatan |

`DELETE /rooms/:id` dirancang sebagai soft delete: ubah `is_active` menjadi `false`, bukan menghapus row secara fisik, agar relasi/histori booking tetap aman.

### 3.3 `bookings` — pengembangan lanjutan

| Kolom          | Tipe rancangan | Keterangan                                             |
| -------------- | -------------- | ------------------------------------------------------ |
| `id`         | BIGSERIAL      | Primary key                                            |
| `user_id`    | BIGINT         | Foreign key ke `users(id)`                           |
| `room_id`    | BIGINT         | Foreign key ke `rooms(id)`                           |
| `purpose`    | TEXT           | Tujuan reservasi                                       |
| `start_time` | TIMESTAMPTZ    | Waktu mulai                                            |
| `end_time`   | TIMESTAMPTZ    | Waktu selesai                                          |
| `status`     | VARCHAR(20)    | `pending`, `approved`, `rejected`, `cancelled` |
| `created_at` | TIMESTAMPTZ    | Waktu pengajuan                                        |

### 3.4 `booking_logs` — pengembangan lanjutan

| Kolom           | Tipe rancangan | Keterangan                                      |
| --------------- | -------------- | ----------------------------------------------- |
| `id`          | BIGSERIAL      | Primary key                                     |
| `booking_id`  | BIGINT         | Foreign key ke `bookings(id)`                 |
| `actor_id`    | BIGINT         | Foreign key ke `users(id)`                    |
| `from_status` | VARCHAR(20)    | Status sebelumnya; dapat null untuk status awal |
| `to_status`   | VARCHAR(20)    | Status setelah perubahan                        |
| `note`        | TEXT           | Catatan/alasan                                  |
| `created_at`  | TIMESTAMPTZ    | Waktu pencatatan                                |

Relasi:

- `users.id` → `bookings.user_id`
- `rooms.id` → `bookings.room_id`
- `bookings.id` → `booking_logs.booking_id`
- `users.id` → `booking_logs.actor_id`

## 4. Aturan bisnis reservasi

Aturan yang direncanakan untuk modul booking:

1. `start_time` harus lebih awal daripada `end_time`.
2. Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
3. Status `pending` dan `approved` menahan slot waktu.
4. Status `rejected` dan `cancelled` tidak menahan slot waktu.
5. Approver hanya dapat memutuskan reservasi untuk ruangan yang ditugaskan kepadanya.
6. Perubahan status dicatat di `booking_logs`.

Dua rentang waktu bertabrakan jika:

```text
existing.start_time < requested.end_time
AND existing.end_time > requested.start_time
```

Rentang yang tepat bersentuhan pada batas waktu (misalnya 09.00–10.00 dan 10.00–11.00) tidak dianggap overlap menurut rumus tersebut.

## 5. Transaksi, race condition, dan idempotency booking

Bagian ini adalah rancangan pengembangan lanjutan. Jangan mengklaim telah diimplementasikan sebelum kode dan test tersedia.

### 5.1 Pencegahan double booking

Race condition dapat terjadi jika dua request memeriksa slot yang sama sebelum salah satunya menyimpan booking. Rancangan yang dipilih adalah transaksi PostgreSQL dengan pessimistic locking pada row ruangan:

```sql
BEGIN;

SELECT id
FROM rooms
WHERE id = $1 AND is_active = TRUE
FOR UPDATE;

-- Periksa booking berstatus pending/approved
-- dengan rentang waktu yang overlap.

-- Jika tidak ada konflik, INSERT booking.

COMMIT;
```

Semua jalur pembuatan booking untuk `room_id` yang sama harus mengunci row ruangan yang sama sebelum memeriksa konflik. Pengecekan di luar transaksi tidak memenuhi rancangan ini.

### 5.2 ACID

- Atomicity: insert booking dan booking log dalam satu transaksi.
- Consistency: validasi, foreign key, dan constraint menjaga aturan data.
- Isolation: transaksi dan locking mencegah dua request mengambil slot yang sama.
- Durability: perubahan yang berhasil di-commit disimpan oleh PostgreSQL.

Jika insert booking log gagal, seluruh transaksi booking harus rollback.

### 5.3 Idempotency-Key

Untuk `POST /bookings`, client direncanakan mengirim header `Idempotency-Key`. Key diikat ke user dan request. Rancangan tabel `idempotency_keys` memiliki kolom utama:

- `id`
- `user_id`
- `idempotency_key`
- `request_hash`
- `response_status`
- `response_body`
- `created_at`

Constraint yang direncanakan: `UNIQUE(user_id, idempotency_key)`. Key sama dengan payload sama mengembalikan hasil yang tersimpan; key sama dengan payload berbeda ditolak sebagai konflik. Penanganan request bersamaan harus mengandalkan unique constraint dan transaksi, bukan hanya pola SELECT lalu INSERT.

## 6. Rancangan notifikasi asynchronous: Transactional Outbox + Redis/Asynq

Notifikasi asynchronous adalah pengembangan lanjutan yang dikaitkan dengan materi Modul 11.

Komponen:

- **Transactional Outbox:** menyimpan event notifikasi bersama perubahan booking dalam transaksi PostgreSQL yang sama.
- **Outbox Relay:** membaca event outbox yang belum diteruskan dan membuat job antrean.
- **Redis:** backend antrean yang digunakan Asynq.
- **Asynq:** library/task queue Go untuk mengelola job asynchronous dan retry.
- **Notification Worker:** mengambil job dan menjalankan pengiriman notifikasi.
- **Consumer idempotency:** mencegah efek samping ganda saat job diproses ulang.

Rancangan tabel `outbox_events` masih perlu difinalisasi sebelum implementasi. Kolom awal yang pernah direncanakan: `id`, `event_type`, `payload`, `created_at`, `processed_at`. Detail status, retry, locking relay, dan kebijakan retensi belum ditetapkan dan tidak boleh ditebak oleh AI.

Alur:

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

Jangan memasang dependency Redis/Asynq, membuat worker, atau menambahkan tabel outbox pada tahap CRUD rooms kecuali tim menyetujui perubahan scope.

## 7. Batas ruang lingkup dan hal yang belum diputuskan

Hal berikut belum ditentukan secara lengkap dalam dokumen sumber dan harus disepakati sebelum implementasi terkait:

- Driver PostgreSQL dan library akses database yang akan digunakan.
- Strategi migration runner dan perintah menjalankan migration.
- Detail konfigurasi environment selain variabel yang sudah ada di `.env.example`.
- Pagination, filtering, dan sorting untuk `GET /rooms`.
- Kebijakan validasi/format waktu dan timezone API secara menyeluruh.
- Bentuk final error detail untuk semua kondisi.
- Provider/channel pengiriman notifikasi.
- Detail schema, retry, dan retensi `outbox_events`.
- Kebijakan penugasan approver ke ruangan dan skema tabel penghubungnya.

AI dan anggota tim tidak boleh mengisi keputusan tersebut secara diam-diam. Catat keputusan tim di dokumen ini sebelum membuat implementasi yang bergantung padanya.
