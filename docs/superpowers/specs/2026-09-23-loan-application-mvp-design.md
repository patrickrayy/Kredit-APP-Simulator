# Sistem Pengajuan Pinjaman (Loan Application) — Fase 1 (MVP) Design

## Tujuan

Proyek latihan fullstack untuk persiapan melamar kerja sebagai corporate
application developer (target: PT BRI Indonesia). Dibangun sebagai simulasi
alur bisnis pinjaman/kredit bank: nasabah mengajukan pinjaman, petugas
mereview dan menyetujui/menolak.

Selain sebagai portofolio, proyek ini adalah sarana belajar terbimbing:
- React 18, Next.js 14 (App Router, Server Components)
- TypeScript, Tailwind CSS 3
- REST API dengan Go (net/http + chi router)
- PostgreSQL (raw SQL via sqlx)
- Autentikasi JWT via HTTP-only cookie

Roadmap lengkap (lihat "Fase Lanjutan" di bawah) mencakup approval
bertingkat, credit scoring (simulasi), pencairan dana + angsuran, dan
notifikasi — tapi **spec ini hanya mencakup Fase 1**. Fase-fase lanjutan
akan mendapat spec terpisah setelah Fase 1 selesai.

## Arsitektur

Dua project terpisah dalam satu parent directory, meniru pola tim
frontend/backend terpisah di lingkungan corporate:

```
bri-loan-app/
├── backend/          Go (net/http + chi), REST API
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── handler/      HTTP handlers per resource
│   │   ├── middleware/   auth, logging, CORS
│   │   ├── repository/   raw SQL queries (sqlx)
│   │   ├── model/        struct definitions
│   │   └── auth/         JWT generate/verify, password hashing
│   ├── migrations/       SQL migration files
│   ├── uploads/          file dokumen yang diupload (gitignored)
│   ├── docker-compose.yml   PostgreSQL container
│   └── go.mod
│
└── frontend/         Next.js 14 (App Router)
    ├── app/
    │   ├── (auth)/login, register
    │   ├── (nasabah)/dashboard, ajukan-pinjaman
    │   ├── (petugas)/review
    ├── components/
    ├── lib/               fetch helpers ke Go API, shared types
    └── package.json
```

Next.js Server Components memanggil Go REST API langsung (server-to-server
fetch), bukan lewat Next.js API routes sebagai perantara — kecuali jika
nanti dibutuhkan BFF kecil untuk kasus khusus.

## Data Model (PostgreSQL)

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  full_name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('nasabah', 'petugas')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE loan_applications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  amount NUMERIC(15,2) NOT NULL,
  purpose TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'approved', 'rejected')),
  reviewed_by UUID REFERENCES users(id),
  review_note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE loan_documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  loan_application_id UUID NOT NULL REFERENCES loan_applications(id) ON DELETE CASCADE,
  doc_type TEXT NOT NULL,       -- 'ktp', 'slip_gaji', dst
  file_path TEXT NOT NULL,      -- path relatif di uploads/
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Catatan:
- Password disimpan sebagai bcrypt hash, tidak pernah plaintext.
- File upload disimpan di disk lokal backend (`uploads/`); hanya path yang
  disimpan di DB. Di production sungguhan biasanya pakai object storage
  (S3-compatible) — dicatat sebagai talking point interview, tidak
  diimplementasi di Fase 1.
- `reviewed_by` + `review_note` mencatat audit trail siapa yang
  approve/reject dan alasannya.

## REST API Endpoints (Fase 1)

```
POST   /api/auth/register        daftar nasabah baru
POST   /api/auth/login           login, set cookie JWT
POST   /api/auth/logout          hapus cookie

GET    /api/loans                list pengajuan (nasabah: milik sendiri, petugas: semua)
POST   /api/loans                nasabah buat pengajuan baru
GET    /api/loans/{id}           detail satu pengajuan
PATCH  /api/loans/{id}/review    petugas approve/reject (role: petugas saja)

POST   /api/loans/{id}/documents upload dokumen pendukung (multipart/form-data)
GET    /api/loans/{id}/documents list dokumen milik satu pengajuan
```

Otorisasi:
- Middleware `AuthRequired`: parse JWT dari cookie, inject user info ke
  request context.
- Middleware `RequireRole("petugas")`: dipasang di endpoint khusus petugas
  (review, list semua pengajuan).
- Kepemilikan data (nasabah hanya boleh akses pengajuan/dokumen miliknya
  sendiri) divalidasi di level handler/repository (`WHERE user_id = ?`),
  bukan hanya di frontend.

## Alur Autentikasi

1. User submit form login (Client Component).
2. Next.js meneruskan request ke Go: `POST /api/auth/login`.
3. Go verifikasi password (bcrypt), generate JWT `{ sub: user_id, role, exp }`,
   set `Set-Cookie: token=<jwt>; HttpOnly; SameSite=Lax`.
4. Browser menyimpan cookie tersebut secara otomatis.
5. Halaman yang butuh data (dashboard, dst) adalah **Server Component**:
   baca cookie via `next/headers` → `cookies().get('token')`, fetch ke Go
   API dengan cookie itu disisipkan di header, Go verifikasi JWT di
   middleware, kembalikan data sesuai role, Server Component render HTML
   langsung berisi data (tanpa loading spinner client-side).

Pembagian Server vs Client Component:
- **Server Component**: halaman list pengajuan, halaman detail (read-only),
  dashboard ringkasan.
- **Client Component**: form pengajuan pinjaman (state input), tombol
  approve/reject (onClick + refresh), upload file (progress/preview), form
  login/register.

## Testing

- **Backend (Go)**: unit test (`testing` bawaan) untuk business logic
  penting — validasi input, password hashing/verify, JWT generate/verify,
  dan authorization check (nasabah tidak bisa akses data nasabah lain).
- **Frontend (Next.js)**: manual testing per flow di browser (register →
  login → ajukan pinjaman → upload dokumen → petugas review). Automated
  frontend test bukan prioritas Fase 1.

## Environment & Tooling

- Go 1.27.1, Node 26.8.1, npm 11.19, Docker 29.8.0 — sudah terpasang.
- PostgreSQL dijalankan via `docker-compose.yml` di folder `backend/`.
- `psql` client tidak terpasang di host — migrasi dijalankan lewat tool Go
  (mis. `golang-migrate`) atau lewat container, bukan psql manual.

## Cara Kerja (Mentoring Workflow)

- User (yang dilatih) menulis kode sendiri, dipandu per langkah kecil.
- Setiap potongan kode direview seperti code review sungguhan: bug dan
  anti-pattern ditunjukkan beserta alasannya, bukan langsung diperbaikikan
  begitu saja.
- Kebiasaan React lama yang salah tempat di konteks Next.js (misal
  Server/Client Component tertukar, `useEffect` untuk data fetching yang
  seharusnya Server Component) dikoreksi secara eksplisit.
- Commit git dilakukan bertahap per fitur kecil dengan pesan commit yang
  jelas.

## Fase Lanjutan (di luar scope spec ini)

Direncanakan sebagai spec terpisah setelah Fase 1 selesai:

- **Fase 2**: Approval bertingkat (Analis Kredit → Branch Manager, routing
  berdasarkan nominal pinjaman).
- **Fase 3**: Credit scoring (simulasi/mock, bukan integrasi SLIK OJK
  sungguhan).
- **Fase 4**: Pencairan dana + tracking angsuran (entitas baru:
  disbursement, installment).
- **Fase 5**: Notifikasi (in-app dulu, email/SMS opsional belakangan).
