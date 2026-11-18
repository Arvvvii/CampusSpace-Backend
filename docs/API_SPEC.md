
API_SPEC.md — Spesifikasi REST API CampusSpace

## 1. Konvensi umum

- Base URL pengembangan: `http://localhost:3000`
- Content-Type request/response: `application/json`
- Status ruang lingkup endpoint saat ini:
  - **Implemented & Aktif:** Health check (`GET /`) dan CRUD `rooms` (`/rooms/*`).
  - **Dalam Pengembangan / Perbaikan:** Modul `bookings` (`/bookings/*`). Endpoint booking belum di-wire di `main.go` dan masih dalam tahap perbaikan serta pengujian; jangan menganggapnya sudah aktif di server utama sampai integrasi dan seluruh test selesai.
  - **Planned (Future Scope):** Autentikasi & authorization middleware, `Idempotency-Key`, dan notifikasi asynchronous (Transactional Outbox, Redis, Asynq).

Semua endpoint yang diimplementasikan harus menggunakan response envelope yang ditetapkan di `CODING_STANDARDS.md`:

```json
{
  "success": true,
  "message": "Pesan deskriptif",
  "data": {},
  "errors": null
}
```

Jangan menganggap contoh response di dokumen ini sebagai bukti bahwa endpoint telah diimplementasikan atau diuji.

## 2. Health check *(Implemented)*

### `GET /`

Tujuan: memeriksa apakah API berjalan.

Contoh response `200 OK`:

```json
{
  "success": true,
  "message": "CampusSpace Backend API is running",
  "data": {
    "service": "CampusSpace-Backend",
    "version": "1.0.0"
  },
  "errors": null
}
```

Implementasi aktual `GET /` di `main.go` telah menggunakan response envelope standar. Endpoint ini mengonfirmasi bahwa proses API merespons, tetapi tidak melakukan pemeriksaan koneksi database secara terpisah.

## 3. Modul Rooms *(Implemented & Terverifikasi)*

### 3.1 `GET /rooms` — daftar ruangan

Tujuan: mengambil daftar ruangan.

Contoh response `200 OK`:

```json
{
  "success": true,
  "message": "Daftar ruangan berhasil diambil",
  "data": [
    {
      "id": 1,
      "code": "LAB-KOM-01",
      "name": "Laboratorium Pemrograman 1",
      "building": "Gedung Teknik Lt. 2",
      "capacity": 40,
      "is_active": true,
      "created_at": "2026-11-16T10:00:00Z"
    }
  ],
  "errors": null
}
```

Pagination, filtering, sorting, dan apakah daftar secara default hanya menampilkan ruangan aktif belum ditentukan. Jangan menambahkan perilaku tersebut tanpa keputusan tim.

### 3.2 `GET /rooms/:id` — detail ruangan

Tujuan: mengambil satu ruangan berdasarkan ID.

- `200 OK`: objek Room pada `data`.
- `400 Bad Request`: format ID tidak valid.
- `404 Not Found`: ruangan tidak ditemukan.

Contoh `404`:

```json
{
  "success": false,
  "message": "Ruangan tidak ditemukan",
  "data": null,
  "errors": null
}
```

### 3.3 `POST /rooms` — tambah ruangan

Request body:

```json
{
  "code": "AULA-UTAMA",
  "name": "Aula Gedung Rektorat",
  "building": "Gedung Rektorat Lt. 1",
  "capacity": 200
}
```

Validasi:

- `code`, `name`, dan `building` wajib diisi dan tidak boleh hanya berisi spasi.
- `capacity` harus lebih besar dari 0.
- `code` harus unik di database.

Response:

- `201 Created`: objek Room yang dibuat, dengan `is_active: true`.
- `400 Bad Request`: JSON tidak valid.
- `409 Conflict`: kode ruangan sudah digunakan.
- `422 Unprocessable Entity`: business validation gagal.

### 3.4 `PUT /rooms/:id` — perbarui ruangan

Request body:

```json
{
  "code": "AULA-UTAMA",
  "name": "Aula Utama Kampus",
  "building": "Gedung Rektorat Lt. 1",
  "capacity": 250,
  "is_active": true
}
```

Response:

- `200 OK`: objek Room setelah diperbarui.
- `400 Bad Request`: ID atau JSON tidak valid.
- `404 Not Found`: ruangan tidak ditemukan.
- `409 Conflict`: kode ruangan dipakai oleh record lain.
- `422 Unprocessable Entity`: validasi business rules gagal.

Semantik PUT adalah penggantian field sesuai request body contoh. Jika tim ingin mendukung partial update, perlu spesifikasi PATCH tersendiri.

### 3.5 `DELETE /rooms/:id` — nonaktifkan ruangan

Perilaku: soft delete dengan menjalankan perubahan setara:

```sql
UPDATE rooms
SET is_active = FALSE
WHERE id = $1;
```

Response contoh `200 OK`:

```json
{
  "success": true,
  "message": "Ruangan berhasil dinonaktifkan",
  "data": null,
  "errors": null
}
```

- `400 Bad Request`: ID tidak valid.
- `404 Not Found`: ruangan tidak ditemukan.
- Operasi ini tidak menghapus row secara fisik.

## 4. Response envelope dan HTTP status

Semua endpoint harus mengikuti envelope di `CODING_STANDARDS.md`.

| HTTP status | Penggunaan                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------- |
| 200         | Berhasil mengambil, memperbarui, atau menonaktifkan data                                                   |
| 201         | Berhasil membuat data                                                                                      |
| 400         | JSON, parameter path, atau query parameter tidak valid                                                     |
| 403         | Pelanggaran hak akses bisnis (role tidak berwenang, user nonaktif, bukan pemilik/penanggung jawab ruangan) |
| 404         | Data tidak ditemukan (`room`, `user`, `booking`)                                                     |
| 409         | Unique conflict, konflik jadwal, ruangan nonaktif, atau transisi status booking ilegal                     |
| 422         | Business validation gagal                                                                                  |
| 500         | Kesalahan internal/database yang tidak terduga                                                             |

Jangan mengirim detail credential, `password_hash`, SQL mentah, stack trace, atau informasi sensitif dalam response error.

## 5. Modul Bookings *(Dalam Pengembangan / Perbaikan)*

*Status: Route dan service booking sedang disempurnakan pada branch perbaikan, belum di-wire ke `main.go`, dan belum memiliki authentication/authorization middleware. Field `user_id`/`actor_id` pada contoh request hanya untuk kebutuhan pengujian sementara; nilainya tidak boleh dianggap sebagai identitas tepercaya untuk penggunaan produksi. Setelah auth tersedia, identitas aktor harus diambil dari authenticated request context.*

### 5.1 Daftar Endpoint Booking

| Method    | Endpoint                                                         | Tujuan                                                                                                          |
| --------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `GET`   | `/bookings/`                                                   | Mengambil daftar booking; mendukung filter opsional `?user_id=` atau `?room_id=` (tidak keduanya sekaligus) |
| `GET`   | `/bookings/:id`                                                | Mengambil detail satu booking berdasarkan ID                                                                    |
| `GET`   | `/bookings/:id/logs`                                           | Mengambil riwayat kronologis perubahan status booking (`booking_logs`)                                        |
| `GET`   | `/bookings/availability?room_id=1&start_time=...&end_time=...` | Memeriksa ketersediaan ruangan pada rentang waktu RFC3339                                                       |
| `POST`  | `/bookings/`                                                   | Mengajukan reservasi baru (hanya `student` aktif; status awal `pending`)                                    |
| `PATCH` | `/bookings/:id/status`                                         | Memperbarui status booking (`approved`, `rejected`, `cancelled`) dan mencatat audit log                   |

### 5.2 Spesifikasi Ringkas per Endpoint Booking

1. **`GET /bookings/`**

   - Tanpa query parameter: mengembalikan seluruh booking (untuk pemantauan).
   - Dengan `?user_id=<id>` atau `?room_id=<id>`: memfilter sesuai ID valid (`> 0`).
   - Query parameter yang dikirim tetapi invalid (bukan angka positif, atau `user_id` dan `room_id` dikirim bersamaan): wajib mengembalikan `400 Bad Request` (tidak boleh fallback mengambil seluruh data).
2. **`GET /bookings/:id`**

   - `200 OK`: objek Booking (`id`, `user_id`, `room_id`, `purpose`, `start_time`, `end_time`, `status`, `created_at`, `updated_at`).
   - `400 Bad Request`: ID tidak valid.
   - `404 Not Found`: booking tidak ditemukan.
3. **`GET /bookings/:id/logs`**

   - `200 OK`: array kronologis `BookingLog`.
   - `400 Bad Request`: ID tidak valid.
   - `404 Not Found`: booking dengan `:id` tersebut tidak ditemukan.
4. **`GET /bookings/availability`**

   - Query parameter wajib: `room_id` (integer `> 0`), `start_time` (RFC3339), `end_time` (RFC3339).
   - `200 OK`: `{ "room_id": 1, "available": true | false }`.
   - `400 Bad Request`: parameter hilang atau format tidak valid.
   - `404 Not Found`: ruangan tidak ditemukan.
   - `409 Conflict`: ruangan berstatus nonaktif (`is_active = false`).
   - `422 Unprocessable Entity`: `start_time` tidak lebih awal dari `end_time`.
5. **`POST /bookings/`**

   - Contoh request body (catatan: pada desain akhir dengan auth middleware, `user_id` berasal dari authenticated request context):
     ```json
     {
       "user_id": 1,
       "room_id": 1,
       "purpose": "Praktikum Basis Data",
       "start_time": "2026-11-20T09:00:00Z",
       "end_time": "2026-11-20T11:00:00Z"
     }
     ```
   - Aturan:
     - Hanya `student` aktif (`is_active = TRUE`) yang boleh mengajukan booking; `admin` dan `approver` ditolak (`403 Forbidden`).
     - Hanya ruangan aktif (`is_active = TRUE`) yang dapat dipesan.
     - Status awal selalu `pending` dan transaksi wajib membuat initial log di `booking_logs` secara atomik dengan locking baris ruangan (`SELECT ... FOR UPDATE`).
     - `pending` dan `approved` menahan slot; `rejected` dan `cancelled` tidak menahan slot.
     - Overlap: `existing.start_time < requested.end_time AND existing.end_time > requested.start_time`. Interval bersebelahan diperbolehkan.
   - Pemetaan HTTP error:
     - `201 Created`: booking berhasil dibuat dengan status `pending`.
     - `400 Bad Request`: JSON tidak valid.
     - `403 Forbidden`: user nonaktif atau role bukan `student` (akses ditolak).
     - `404 Not Found`: user atau ruangan tidak ditemukan.
     - `409 Conflict`: ruangan nonaktif atau jadwal bertabrakan.
     - `422 Unprocessable Entity`: validasi payload gagal (`user_id`/`room_id` invalid, `purpose` kosong, rentang waktu tidak valid).
6. **`PATCH /bookings/:id/status`**

   - Contoh request body (catatan: pada desain akhir dengan auth middleware, `actor_id` berasal dari authenticated request context):
     ```json
     {
       "actor_id": 2,
       "status": "approved",
       "note": "Disetujui untuk kegiatan akademik"
     }
     ```
   - Aturan transisi & hak akses:
     - Transisi valid: `pending → approved`, `pending → rejected`, `pending → cancelled`, `approved → cancelled`.
     - `rejected` dan `cancelled` bersifat terminal (tidak boleh diubah kembali).
     - `student` aktif: hanya boleh mengubah ke `cancelled` untuk booking miliknya sendiri.
     - `approver` aktif: hanya boleh mengubah `pending` ke `approved` atau `rejected` untuk ruangan yang ditugaskan kepadanya di `approver_rooms` (tidak boleh `cancelled`).
     - `admin`: tidak boleh melakukan `approved`, `rejected`, maupun `cancelled`.
     - Pembaruan status, pembaruan `updated_at`, dan pencatatan `booking_logs` wajib atomik dalam satu transaksi.
   - Pemetaan HTTP error:
     - `200 OK`: status booking berhasil diperbarui.
     - `400 Bad Request`: ID atau JSON tidak valid.
     - `403 Forbidden`: aktor nonaktif, role tidak berwenang (`admin` / aksi terlarang), `student` membatalkan booking orang lain, atau `approver` bukan penanggung jawab ruangan.
     - `404 Not Found`: booking atau aktor tidak ditemukan.
     - `409 Conflict`: transisi status ilegal (termasuk mencoba mengubah status terminal `rejected`/`cancelled`).
     - `422 Unprocessable Entity`: `actor_id` atau nilai `status` tidak valid.

### 5.3 Fitur Future Scope Modul Bookings *(Belum Diimplementasikan)*

- **Authentication & Authorization Middleware:** Endpoint registrasi/login dan middleware pembaca token/session yang menyuntikkan identitas aktor ke request context.
- **Idempotency-Key:** Header `Idempotency-Key` pada `POST /bookings/` beserta tabel `idempotency_keys` untuk melindungi request client dari duplikasi saat retry.

## 6. Notifikasi asynchronous — rancangan, bukan endpoint aktif *(Future Scope)*

Rancangan notifikasi:
`Booking Service → PostgreSQL Transaction (booking + outbox event) → Outbox Relay → Asynq Queue (Redis) → Notification Worker`.

Transactional Outbox, Redis, Asynq, worker, retry policy, dan consumer idempotency adalah pengembangan lanjutan (future scope). Tidak ada endpoint notifikasi publik yang ditetapkan dalam spesifikasi saat ini.
