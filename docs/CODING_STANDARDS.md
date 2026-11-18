
# CODING_STANDARDS.md — Konvensi Kode dan Kontrak Integrasi Go

Dokumen ini menetapkan kontrak bersama agar kode Arvi, Marsha, dan Robithoh dapat diintegrasikan. Jika implementasi yang sudah ada berbeda, jangan melakukan refactor besar diam-diam; laporkan perbedaannya dan sepakati perubahan.

## 1. Konvensi Go

- Nama file: `snake_case.go`, misalnya `room_repository.go`.
- Nama package: lowercase.
- Nama exported struct/interface/function: `PascalCase`.
- Nama variabel lokal: idiom Go, singkat dan jelas.
- Primary key di Go: `int64`, untuk memetakan PostgreSQL `BIGSERIAL`/`BIGINT`.
- JSON tag dan nama kolom database: `snake_case`.
- Format seluruh file Go dengan `gofmt`.
- Gunakan `context.Context` pada operasi repository yang berinteraksi dengan database.
- Jangan mengabaikan error; tangani atau teruskan dengan konteks yang memadai.

## 2. Kontrak entity Room

Lokasi: `app/model/room.go`

```go
package model

import "time"

type Room struct {
    ID        int64     `json:"id" db:"id"`
    Code      string    `json:"code" db:"code"`
    Name      string    `json:"name" db:"name"`
    Building  string    `json:"building" db:"building"`
    Capacity  int       `json:"capacity" db:"capacity"`
    IsActive  bool      `json:"is_active" db:"is_active"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

Struct request dapat dibuat terpisah bila dibutuhkan, tetapi field dan JSON name harus konsisten dengan `API_SPEC.md`. Jangan menambahkan field database yang belum disepakati.

## 3. Kontrak repository Room

Lokasi: `app/repository/room_repository.go`

```go
package repository

import (
    "context"

    "github.com/Arvvvii/CampusSpace-Backend/app/model"
)

type RoomRepository interface {
    Create(ctx context.Context, room *model.Room) error
    GetAll(ctx context.Context) ([]model.Room, error)
    GetByID(ctx context.Context, id int64) (*model.Room, error)
    Update(ctx context.Context, room *model.Room) error
    Deactivate(ctx context.Context, id int64) error
}
```

Tanggung jawab:

- Marsha membuat interface dan implementasi query PostgreSQL sesuai pembagian kerja.
- Robithoh menggunakan kontrak interface tersebut pada service.
- Jika nama method atau signature perlu diubah, koordinasikan sebelum mengubah implementasi.

Aturan:

- Repository tidak boleh mengimpor Fiber.
- Gunakan parameterized query; jangan menyisipkan input pengguna langsung ke string SQL.
- Tangani kondisi record tidak ditemukan dan unique violation agar dapat dipetakan ke error aplikasi.
- Jangan mengembalikan error database mentah ke client HTTP.

## 4. Business rules dan service

- Business rules berada di `app/service` mengikuti pola Modul 4.
- Fungsi validasi murni Go dan tidak menerima `*fiber.Ctx`.
- Validasi Room:
  - `code`, `name`, `building` tidak boleh kosong atau hanya spasi.
  - `capacity > 0`.
- Unit test business rules tidak membutuhkan koneksi database atau HTTP request.
- Service mengorkestrasi validasi dan pemanggilan repository.
- Pola constructor/dependency injection mengikuti implementasi project yang disepakati; jangan membuat pola baru tanpa kebutuhan.

## 5. Route dan batas layer

- Route Fiber berada di `route`.
- `route/room.go` memetakan HTTP request ke service.
- Secara rancangan, `route/route.go` dapat mendaftarkan route utama. Pada implementasi MVP saat ini, `main.go` mendaftarkan `route.RegisterRoomRoutes` secara langsung; `route/route.go` belum dibuat.
- Jangan memindahkan business rules ke route.
- Jangan menambahkan folder/layer `controller`, `handler`, `usecase`, atau `delivery`.
- Repository dan business rules tidak boleh bergantung pada Fiber context.

## 6. Standard JSON response envelope

Semua endpoint menggunakan bentuk konsisten. Endpoint health check `GET /` juga mengikuti envelope ini:

```json
{
  "success": true,
  "message": "Pesan deskriptif singkat",
  "data": {},
  "errors": null
}
```

Untuk error, `success` bernilai `false`; `data` bernilai `null`; `errors` dapat berisi informasi validasi yang aman untuk client. Bentuk detail `errors` belum ditetapkan secara final; jangan mengarang kontrak baru tanpa persetujuan tim.

Pemetaan status:

- `200 OK`: berhasil mengambil, memperbarui, atau menonaktifkan.
- `201 Created`: berhasil membuat (`POST /rooms`).
- `400 Bad Request`: JSON salah atau parameter ID bukan angka valid.
- `403 Forbidden`: aktor tidak memiliki hak akses atau user berstatus nonaktif pada operasi yang mensyaratkannya.
- `404 Not Found`: resource yang diminta tidak ditemukan (misalnya room, user, atau booking).
- `409 Conflict`: kode ruangan duplikat, konflik jadwal, ruangan nonaktif saat pemeriksaan/pengajuan booking, atau transisi status booking ilegal.
- `422 Unprocessable Entity`: business validation gagal, misalnya rentang waktu tidak valid atau field wajib tidak memenuhi aturan.
- `500 Internal Server Error`: kesalahan internal/database yang tidak terduga.

## 7. Database dan keamanan

- Jangan hardcode password, token, atau credential.
- Ambil konfigurasi dari environment melalui package `config`.
- File `.env` lokal wajib di-ignore; commit hanya `.env.example`.
- Jangan memasukkan secret ke log atau response.
- Gunakan query parameterized.
- Gunakan transaksi untuk operasi multi-query yang harus atomik.
- Implementasi saat ini menggunakan `database/sql` dengan driver PostgreSQL `github.com/jackc/pgx/v5/stdlib` (pgx v5). Koneksi dan connection pool dikelola di `database/postgres.go`.

## 8. Error handling

- Bedakan error validasi, record tidak ditemukan, konflik unique, dan error internal.
- Jangan membocorkan SQL, stack trace, password, atau credential ke client.
- Bungkus error internal dengan konteks yang berguna untuk log.
- Jangan mengubah error menjadi sukses atau mengabaikannya.

## 9. Pemeriksaan kualitas

Sebelum commit:

```bash
gofmt -w <file-go-yang-diubah>
go test ./...
go build ./...
git diff --check
git status
```

Jalankan perintah yang relevan. Jika test membutuhkan PostgreSQL yang belum tersedia, laporkan bahwa test integrasi belum dijalankan; jangan mengklaim semuanya lulus.

## 10. Aturan perubahan kontrak

Perubahan pada entity, nama field, tipe ID, interface repository, response envelope, endpoint, atau perilaku DELETE harus disepakati oleh anggota terkait dan Git Master. Perbarui dokumen terkait dalam PR yang sama atau PR dokumentasi yang disepakati.
