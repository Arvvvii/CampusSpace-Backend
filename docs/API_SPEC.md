# API_SPEC.md — Spesifikasi REST API CampusSpace

## 1. Konvensi umum

- Base URL pengembangan: `http://localhost:3000`
- Content-Type request/response: `application/json`
- Endpoint yang wajib diprioritaskan pada MVP: health check dan CRUD `rooms`.
- Modul booking di bawah adalah rancangan pengembangan lanjutan, bukan endpoint yang dinyatakan sudah tersedia.

Semua endpoint yang sudah diimplementasikan harus menggunakan response envelope yang ditetapkan di `CODING_STANDARDS.md`:

```json
{
  "success": true,
  "message": "Pesan deskriptif",
  "data": {},
  "errors": null
}
```

Jangan menganggap contoh response di dokumen ini sebagai bukti bahwa endpoint telah diimplementasikan.

## 2. Health check

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

## 3. Modul Rooms — MVP wajib

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

| HTTP status | Penggunaan |
|---|---|
| 200 | Berhasil mengambil, memperbarui, atau menonaktifkan |
| 201 | Berhasil membuat data |
| 400 | JSON atau parameter tidak valid |
| 404 | Data tidak ditemukan |
| 409 | Unique conflict atau konflik jadwal |
| 422 | Business validation gagal |
| 500 | Kesalahan internal/database yang tidak terduga |

Jangan mengirim detail credential, SQL mentah, stack trace, atau informasi sensitif dalam response error.

## 5. Modul Bookings — rancangan pengembangan lanjutan

Endpoint berikut belum menjadi target implementasi MVP:

| Method | Endpoint | Tujuan/aturan rancangan |
|---|---|---|
| GET | `/bookings/availability?room_id=1&date=2026-11-20` | Cek ketersediaan ruangan |
| POST | `/bookings` | Ajukan reservasi; rancangan memakai `Idempotency-Key` dan locking |
| PATCH | `/bookings/:id/approve` | Approver menyetujui reservasi |
| PATCH | `/bookings/:id/reject` | Approver menolak dengan alasan |
| PATCH | `/bookings/:id/cancel` | Student membatalkan pengajuan sendiri |

Aturan booking:
- Waktu mulai harus lebih awal daripada waktu selesai.
- Hanya ruangan aktif yang dapat dipesan.
- `pending` dan `approved` menahan slot; `rejected` dan `cancelled` tidak.
- Overlap: `existing.start_time < requested.end_time AND existing.end_time > requested.start_time`.
- Pembuatan booking direncanakan memakai transaksi dan `SELECT ... FOR UPDATE` pada row room sebelum pengecekan konflik.
- Pembuatan booking dan booking log harus atomik.
- `Idempotency-Key` melindungi request client dari pembuatan booking ganda saat retry.

Detail autentikasi, authorization middleware, pagination, format error validation, dan skema request/response booking perlu ditetapkan sebelum endpoint booking diimplementasikan.

## 6. Notifikasi asynchronous — rancangan, bukan endpoint MVP

Rancangan notifikasi:
`Booking Service → PostgreSQL Transaction (booking + outbox event) → Outbox Relay → Asynq Queue (Redis) → Notification Worker`.

Transactional Outbox, Redis, Asynq, worker, retry policy, dan consumer idempotency adalah pengembangan lanjutan. Tidak ada endpoint notifikasi publik yang ditetapkan dalam spesifikasi saat ini.
