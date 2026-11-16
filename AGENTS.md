# AGENTS.md — Aturan AI Coding Assistant CampusSpace

Dokumen ini adalah instruksi kerja untuk AI coding assistant yang digunakan oleh Arvi, Marsha, dan Robithoh pada repository `CampusSpace-Backend`.

## 1. Identitas dan sumber kebenaran proyek

- Nama proyek: CampusSpace Backend.
- Tujuan: REST API reservasi ruangan dan laboratorium kampus.
- Repository: https://github.com/Arvvvii/CampusSpace-Backend
- Bahasa: Go.
- Framework: Fiber v2.
- Database: PostgreSQL.
- Arsitektur: Clean Architecture mengikuti struktur Modul 4 praktikum.
- Target implementasi utama tugas: CRUD `rooms` dari HTTP request sampai PostgreSQL, termasuk validasi dan pengujian yang relevan.

Baca dokumen berikut sebelum mengubah kode:
1. `docs/ARCHITECTURE.md`
2. `docs/API_SPEC.md`
3. `docs/CODING_STANDARDS.md`
4. `docs/WORKFLOW_AND_TASKS.md`

Jika ada konflik antar dokumen, jangan menebak. Jelaskan konflik dan minta keputusan tim. Keputusan yang sudah disepakati harus diperbarui pada dokumen terkait agar tidak ada dua spesifikasi aktif.

## 2. Status fitur

### Ruang lingkup MVP yang diprioritaskan
- Go + Fiber v2.
- Koneksi PostgreSQL.
- Migration tabel `rooms`.
- CRUD `rooms` lengkap.
- Validasi business rules dan unit test yang relevan.
- Build dan pengujian aplikasi.

### Rancangan/pengembangan lanjutan
- Registrasi, autentikasi, dan otorisasi.
- Booking, approval, rejection, cancellation, dan booking history.
- Transaksi ACID dan pencegahan double booking.
- `Idempotency-Key` untuk pembuatan booking.
- Transactional Outbox, Outbox Relay, Redis, Asynq, Notification Worker, dan consumer idempotency.

**Larangan dependency prematur:** selama target utama masih CRUD `rooms`, jangan menambahkan dependency Redis/Asynq, konfigurasi Redis, worker, outbox relay, atau tabel outbox ke implementasi aktif kecuali Git Master dan tim secara eksplisit menyetujui perubahan scope.

Jangan menyebut fitur sebagai "selesai", "berjalan", atau "sudah diuji" tanpa bukti implementasi dan pengujian aktual.

## 3. Struktur arsitektur yang wajib diikuti

Gunakan struktur package berikut sesuai Modul 4:

```text
CampusSpace-Backend/
├── app/
│   ├── model/
│   ├── repository/
│   └── service/
├── config/
├── database/
├── helper/
├── middleware/
├── route/
├── migrations/
├── docs/
├── AGENTS.md
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── README.md
└── main.go
```

Jangan membuat layer/folder `controller`, `handler`, `usecase`, atau `delivery`.

Aturan pemisahan:
- `app/model`: entity dan DTO.
- `app/repository`: interface dan implementasi akses PostgreSQL.
- `app/service`: alur aplikasi dan business rules.
- `config`: konfigurasi aplikasi, environment, dan logger.
- `database`: koneksi PostgreSQL dan connection pool.
- `helper`: response envelope dan fungsi bantuan lintas package.
- `middleware`: middleware aplikasi; autentikasi/otorisasi merupakan tahap lanjutan.
- `route`: pendaftaran endpoint Fiber dan pemetaan ke service.
- `migrations`: file SQL migration.
- `main.go`: merakit dependency dan menjalankan server.

`app/repository` tidak boleh mengimpor Fiber. Business rules harus dapat diuji tanpa Fiber context.

## 4. Kontrak kerja AI

Sebelum menulis atau mengubah file:
1. Identifikasi anggota yang sedang bekerja dan tanggung jawab file-nya dari `docs/WORKFLOW_AND_TASKS.md`.
2. Baca spesifikasi yang relevan.
3. Periksa file dan pola kode yang sudah ada; jangan mengasumsikan file kosong atau belum dibuat.
4. Pastikan perubahan tidak melanggar kontrak model, repository, API, maupun struktur Modul 4.

Batas kepemilikan file:
- Arvi: dokumentasi, konfigurasi, database connection, helper, `main.go`, `route/route.go`, integrasi.
- Marsha: `migrations/*` dan `app/repository/*`.
- Robithoh: `app/model/*`, `app/service/*`, `route/room.go`, dan unit test terkait.
- Jika tugas memerlukan perubahan lintas kepemilikan, jelaskan file yang perlu disentuh dan minta koordinasi; jangan diam-diam mengambil alih.

AI harus:
- Menghasilkan perubahan sekecil mungkin sesuai tugas.
- Menjelaskan file yang dibuat/diubah dan alasan teknisnya.
- Menandai asumsi atau keputusan yang belum ditentukan.
- Tidak mengarang dependency, endpoint, kolom, aturan bisnis, atau hasil test.
- Tidak mengubah spesifikasi atau kontrak bersama tanpa persetujuan.
- Tidak menghapus atau menimpa pekerjaan yang sudah ada.
- Tidak membuat credential, token, password, atau `.env` asli.
- Tidak menyatakan test berhasil kecuali perintah benar-benar dijalankan dan hasilnya tersedia.

Jika informasi penting belum ada, berhenti pada bagian yang belum jelas dan ajukan pertanyaan paling spesifik. Jangan mengisi kekosongan dengan tebakan.

## 5. Aturan teknis dan keamanan

- Ikuti `docs/CODING_STANDARDS.md`.
- Gunakan `gofmt`.
- Gunakan parameterized query; jangan merangkai input pengguna langsung ke SQL.
- Jangan hardcode credential.
- File `.env` asli tidak boleh di-commit; hanya `.env.example` yang boleh dibagikan.
- Jangan menambahkan dependency tanpa alasan dan persetujuan scope.
- Jangan mengubah module path atau dependency inti tanpa koordinasi Git Master.

## 6. Pemeriksaan sebelum menyerahkan pekerjaan

Jalankan pemeriksaan yang relevan dengan perubahan:
```bash
gofmt -w <file-go-yang-diubah>
go test ./...
go build ./...
git status
git diff --check
```

Jangan mengklaim perintah berhasil jika belum dijalankan. Jika database diperlukan untuk test integrasi tetapi belum tersedia, jelaskan bahwa test tersebut belum dijalankan.

## 7. Alur Git

- Jangan bekerja langsung di `main`.
- Buat branch dari `main` terbaru.
- Lakukan micro commits.
- Push branch sendiri dan buat Pull Request dengan base `main`.
- Jangan merge PR; Arvi sebagai Git Master yang melakukan merge setelah review dan approval sesuai branch protection.
- Setelah merge, sinkronkan `main` lokal sebelum pekerjaan berikutnya.

Detail branch dan pembagian tugas ada di `docs/WORKFLOW_AND_TASKS.md`.
