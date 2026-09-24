# Loan Application Frontend (Fase 1) Implementation Plan

> **Mode eksekusi:** Plan ini TIDAK dieksekusi oleh subagent otonom.
> Mengikuti "Cara Kerja (Mentoring Workflow)" di spec: setiap task dikerjakan
> interaktif satu per satu — konsep dijelaskan (dalam Bahasa Indonesia), user
> yang mengetik kode **dan** menjalankan semua perintah (Git Bash), lalu hasilnya
> direview seperti code review sungguhan sebelum lanjut ke task berikutnya.
> Checkbox (`- [ ]`) dipakai untuk melacak progres antar sesi.

**Goal:** Aplikasi web Next.js yang memakai Go API Fase 1 yang sudah jadi —
nasabah bisa register, login, mengajukan pinjaman, dan mengupload dokumen;
petugas bisa melihat semua pengajuan dan approve/reject — diverifikasi manual
end-to-end di browser.

**Architecture:** Browser **tidak pernah** memanggil Go API secara langsung.
Server Components mengambil data dari Go (server-to-server `fetch`) dan
Server Actions menangani semua mutasi (login, register, logout, buat
pengajuan, upload, review). Saat login, Server Action membaca JWT dari
`Set-Cookie` respons Go lalu menyimpannya sebagai cookie HttpOnly milik
Next.js; setiap request berikutnya ke Go meneruskan cookie itu. Karena semua
panggilan ke Go terjadi di server, tidak perlu CORS dan URL API tidak pernah
terekspos ke browser. Otorisasi yang sebenarnya tetap ditegakkan Go; cek role
di frontend hanya untuk UX (redirect/menyembunyikan tombol).

**Tech Stack:** Next.js 14 (App Router, Server Components, Server Actions),
React 18, TypeScript 5, Tailwind CSS 3, `server-only`, Vitest (unit test
helper murni di `lib/`). Backend: Go API dari plan
`2026-09-23-loan-application-backend.md` (+2 endpoint/field tambahan di Task
1–2 plan ini).

**Spec:** `docs/superpowers/specs/2026-09-23-loan-application-mvp-design.md`

## Global Constraints

- Next.js 14 (App Router), TypeScript, Tailwind CSS 3 — sesuai spec. Pakai API
  Next **14**: `cookies()` dan `params`/`searchParams` bersifat **sinkron**,
  `useFormState`/`useFormStatus` dari `react-dom`. Jangan upgrade ke Next 15
  (API-nya berubah jadi async). Kalau `create-next-app@14` gagal di Node 26,
  berhenti dan diskusikan dulu — jangan diam-diam ganti versi.
- "Next.js Server Components memanggil Go REST API langsung (server-to-server
  fetch), bukan lewat Next.js API routes sebagai perantara" (spec). Mutasi
  memakai Server Actions (bukan API routes).
- Server Component: halaman list pengajuan, halaman detail, dashboard. Client
  Component: form login/register, form pengajuan, upload file, form
  approve/reject (spec).
- Cookie `token` di Next.js: `HttpOnly`, `SameSite=Lax`, `Path=/`,
  `Max-Age=86400`, `Secure` hanya di production — sama dengan atribut cookie Go.
- Semua `fetch` ke Go memakai `cache: 'no-store'` (data per-user, tidak boleh
  di-cache Next.js).
- Otorisasi ditegakkan di Go (spec: "bukan hanya di frontend"). Server Actions
  adalah endpoint HTTP publik — jangan pernah mengandalkan UI yang
  menyembunyikan tombol.
- ID dari URL selalu di-`encodeURIComponent` sebelum dimasukkan ke path Go API.
- Testing frontend: "manual testing per flow di browser (register → login →
  ajukan pinjaman → upload dokumen → petugas review)" (spec). Tambahan: unit
  test Vitest **hanya** untuk fungsi murni di `lib/` (format, validasi, parsing
  cookie, routing).
- Backend harus berjalan dengan `.env` dimuat:
  `cd backend && set -a; source .env; set +a; go run ./cmd/api`.
- Akun uji di DB lokal: `siti@example.com` & `andi@example.com` (nasabah),
  `budi2@example.com` (petugas) — password semua `rahasia123`.
- Semua perintah dijalankan di Git Bash. File baru memakai line ending LF.

## File Structure

```
backend/                                   (Task 1–2, perubahan kecil)
├── cmd/api/main.go                        + route GET /api/auth/me
├── internal/handler/auth_handler.go       + AuthHandler.Me
├── internal/repository/user_repository.go + UserRepository.FindByID
├── internal/model/loan.go                 + field ApplicantName
└── internal/repository/loan_repository.go SELECT + JOIN users (applicant_name)

frontend/                                  (Task 3–12, baru)
├── .env.example                           API_URL template (di-commit)
├── .env.local                             API_URL lokal (gitignored)
├── next.config.mjs                        batas body Server Action (Task 10)
├── app/
│   ├── layout.tsx                         root layout (html lang="id")
│   ├── globals.css                        Tailwind + class komponen (.card, .input, …)
│   ├── page.tsx                           "/" → redirect sesuai role
│   ├── not-found.tsx                      404 global
│   ├── error.tsx                          error boundary global (Go mati, dll)
│   ├── actions/
│   │   ├── auth.ts                        login, register, logout
│   │   ├── loans.ts                       createLoan, reviewLoan
│   │   └── documents.ts                   uploadDocument
│   ├── (auth)/
│   │   ├── layout.tsx                     kartu di tengah; redirect kalau sudah login
│   │   ├── login/page.tsx + LoginForm.tsx
│   │   └── register/page.tsx + RegisterForm.tsx
│   └── (app)/
│       ├── layout.tsx                     wajib login + AppHeader
│       ├── loading.tsx                    skeleton saat Server Component memuat
│       ├── dashboard/page.tsx             nasabah: pengajuan saya
│       ├── ajukan-pinjaman/page.tsx + LoanForm.tsx
│       ├── review/page.tsx                petugas: semua pengajuan
│       └── pinjaman/[id]/
│           ├── page.tsx                   detail (dipakai kedua role)
│           ├── DocumentUploadForm.tsx     (Task 10)
│           └── ReviewForm.tsx             (Task 11)
├── components/
│   ├── SubmitButton.tsx                   tombol submit + state pending
│   ├── AppHeader.tsx                      nav per role + tombol keluar
│   ├── StatusBadge.tsx
│   └── LoanTable.tsx
└── lib/
    ├── types.ts                           tipe data API + DOC_TYPES + FormState
    ├── format.ts (+ .test.ts)             formatRupiah, formatDate, statusLabel, docTypeLabel
    ├── validation.ts (+ .test.ts)         parseAmount, isDocType, MAX_UPLOAD_BYTES
    ├── cookies.ts (+ .test.ts)            extractToken dari header Set-Cookie
    ├── routes.ts (+ .test.ts)             homePathFor(role)
    ├── api.ts                             apiFetch (server-only)
    ├── auth.ts                            getCurrentUser, requireUser (server-only)
    └── loans.ts                           getLoans, getLoan, getLoanDocuments (server-only)
```

Catatan deviasi dari sketsa folder di spec: spec menggambar route group
`(nasabah)` dan `(petugas)`. Plan ini memakai satu group `(app)` (layout
bersama: wajib login + header) dan cek role dilakukan di **setiap page** lewat
`requireUser(role)`. Alasannya: layout di App Router tidak di-render ulang
saat navigasi antar halaman di dalam layout yang sama, jadi cek akses yang
paling bisa diandalkan diletakkan dekat data (di page), bukan di layout.

---

## Task 1: Backend — `GET /api/auth/me`

Frontend butuh cara mengetahui "siapa yang sedang login" (nama + role) di
setiap request — misalnya setelah reload halaman. JWT di cookie tidak bisa
dibaca JavaScript (HttpOnly) dan tidak boleh dipercaya tanpa verifikasi, jadi
Go yang menjawab lewat endpoint ini.

**Files:**
- Modify: `backend/internal/repository/user_repository.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/cmd/api/main.go`
- Modify: `README.md` (tabel API)

**Interfaces:**
- Consumes: `mw.AuthRequired`, `mw.UserFromContext` (backend Task 8).
- Produces: `UserRepository.FindByID(ctx, id string) (*model.User, error)`,
  `AuthHandler.Me`. Route `GET /api/auth/me` → `200` JSON
  `{"id","full_name","email","role","created_at"}` atau `401`.

- [ ] **Step 1: Tambahkan `FindByID` di `user_repository.go`** (di bawah `FindByEmail`)

```go
func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	query := `SELECT id, full_name, email, password_hash, role, created_at FROM users WHERE id = $1`
	if err := r.db.GetContext(ctx, &u, query, id); err != nil {
		return nil, err
	}
	return &u, nil
}
```

- [ ] **Step 2: Tambahkan handler `Me` di `auth_handler.go`**

Tambahkan import `mw "loanapp/internal/middleware"`, lalu di bawah `Logout`:

```go
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authUser, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.users.FindByID(r.Context(), authUser.UserID)
	if err != nil {
		// Token valid tapi user-nya sudah tidak ada di DB.
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
```

`PasswordHash` punya tag `json:"-"`, jadi hash tidak pernah ikut terkirim.

- [ ] **Step 3: Wire route di `main.go`** (di bawah route logout)

```go
	r.With(mw.AuthRequired).Get("/api/auth/me", authHandler.Me)
```

`r.With(...)` memasang middleware untuk **satu route saja**, tanpa membuat
group baru.

- [ ] **Step 4: Verifikasi**

```bash
cd ~/bri-loan-app/backend
go build ./... && go test ./...
set -a; source .env; set +a; go run ./cmd/api
```

Di terminal kedua (`cd ~/bri-loan-app/backend`):

```bash
curl -i -b cookies3.txt http://localhost:8080/api/auth/me
```
Expected: `200` dengan data Siti, **tanpa** field `password_hash`. (Kalau `401`,
login ulang dulu ke `cookies3.txt`.)

```bash
curl -i http://localhost:8080/api/auth/me
```
Expected: `401`.

- [ ] **Step 5: Tambah baris di tabel API `README.md`** (di bawah baris logout)

```markdown
| `GET` | `/api/auth/me` | authenticated | current user (`id`, `full_name`, `email`, `role`) |
```

- [ ] **Step 6: Commit**

```bash
cd ~/bri-loan-app
git add backend README.md
git commit -m "feat(backend): add GET /api/auth/me endpoint"
```

---

## Task 2: Backend — nama pemohon di list & detail pengajuan

Petugas butuh melihat **siapa** yang mengajukan, bukan hanya UUID `user_id`.

**Files:**
- Modify: `backend/internal/model/loan.go`
- Modify: `backend/internal/repository/loan_repository.go`

**Interfaces:**
- Produces: field JSON `applicant_name` (string) di respons `GET /api/loans`
  dan `GET /api/loans/{id}`. Respons `POST /api/loans` dan
  `PATCH /api/loans/{id}/review` **tidak** memuat field ini (dihilangkan via
  `omitempty`) karena `RETURNING` tidak bisa JOIN.

- [ ] **Step 1: Tambahkan field di `model/loan.go`** (di bawah `UserID`)

```go
	ApplicantName string    `db:"applicant_name" json:"applicant_name,omitempty"`
```

- [ ] **Step 2: Refactor query SELECT di `loan_repository.go`**

Tambahkan konstanta ini di atas `type LoanRepository struct`:

```go
// loanSelect dipakai bersama oleh semua query baca supaya daftar kolom dan
// JOIN ke users hanya ditulis sekali.
const loanSelect = `
	SELECT la.id, la.user_id, u.full_name AS applicant_name, la.amount, la.purpose,
	       la.status, la.reviewed_by, la.review_note, la.created_at, la.updated_at
	FROM loan_applications la
	JOIN users u ON u.id = la.user_id`
```

Lalu ganti **isi query** di tiga fungsi baca:

`ListByUser`:
```go
	query := loanSelect + ` WHERE la.user_id = $1 ORDER BY la.created_at DESC`
```

`ListAll`:
```go
	query := loanSelect + ` ORDER BY la.created_at DESC`
```

`GetByID`:
```go
	query := loanSelect + ` WHERE la.id = $1`
```

`Create` dan `Review` tidak diubah.

Kenapa kolom harus diberi prefix `la.`/`u.`? Kedua tabel punya kolom `id` dan
`created_at` — tanpa prefix Postgres menolak query dengan error
*"column reference is ambiguous"*.

- [ ] **Step 3: Verifikasi**

```bash
cd ~/bri-loan-app/backend
go build ./... && go test ./...
```
Restart server (dengan `.env`), lalu di terminal kedua:

```bash
curl -s -b cookies.txt http://localhost:8080/api/loans
```
Expected: setiap item punya `"applicant_name"` (`"Siti"`, `"Budi Dua"`). (Kalau
`401`, login ulang budi2 ke `cookies.txt`.)

```bash
curl -s -b cookies3.txt http://localhost:8080/api/loans/2b2086f6-b2af-490f-8b31-08687bae2a50
```
Expected: `"applicant_name":"Siti"`.

- [ ] **Step 4: Commit**

```bash
git add internal
git commit -m "feat(backend): include applicant name in loan list and detail"
```

---

## Task 3: Scaffold Next.js 14 + Tailwind + Vitest

**Files:**
- Create: `frontend/` (via `create-next-app@14`)
- Modify: `frontend/app/layout.tsx`, `frontend/app/page.tsx`, `frontend/app/globals.css`, `frontend/package.json`
- Create: `frontend/.env.example`, `frontend/.env.local`
- Modify: `.gitattributes`

**Interfaces:**
- Produces: class CSS komponen `card`, `label`, `input`, `btn-primary`,
  `btn-secondary`, `alert-error`, `alert-success` (dipakai semua task
  berikutnya). Env `API_URL`. Script `npm test`.

- [ ] **Step 1: Generate project**

```bash
cd ~/bri-loan-app
npx create-next-app@14 frontend --typescript --tailwind --eslint --app --no-src-dir --import-alias "@/*" --use-npm
```

Pastikan tidak ada folder `frontend/.git` (repo git-nya tetap satu di root):
```bash
ls -a frontend | grep '^\.git$' || echo "ok: no nested git repo"
```

- [ ] **Step 2: Install dependency tambahan**

```bash
cd frontend
npm install server-only
npm install -D vitest
```

Di `package.json`, tambahkan di `"scripts"`:
```json
    "test": "vitest run"
```

- [ ] **Step 3: Ganti `app/globals.css` seluruhnya**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer components {
  .card {
    @apply rounded-lg border border-slate-200 bg-white p-6 shadow-sm;
  }
  .label {
    @apply mb-1 block text-sm font-medium text-slate-700;
  }
  .input {
    @apply block w-full rounded-md border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-blue-600 focus:outline-none focus:ring-1 focus:ring-blue-600;
  }
  .btn-primary {
    @apply inline-flex items-center justify-center rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white hover:bg-blue-800 disabled:cursor-not-allowed disabled:opacity-60;
  }
  .btn-secondary {
    @apply inline-flex items-center justify-center rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60;
  }
  .alert-error {
    @apply rounded-md bg-red-50 px-3 py-2 text-sm text-red-700;
  }
  .alert-success {
    @apply rounded-md bg-green-50 px-3 py-2 text-sm text-green-700;
  }
}
```

- [ ] **Step 4: Ganti `app/layout.tsx` seluruhnya**

```tsx
import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'Kredit APP Simulator',
  description: 'Simulasi workflow pengajuan kredit — project latihan pribadi.',
}

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="id">
      <body className={`${inter.className} bg-slate-50 text-slate-900 antialiased`}>{children}</body>
    </html>
  )
}
```

- [ ] **Step 5: Ganti `app/page.tsx` seluruhnya** (sementara; diganti di Task 5)

```tsx
export default function Home() {
  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="card text-center">
        <h1 className="text-2xl font-semibold">Kredit APP Simulator</h1>
        <p className="mt-2 text-sm text-slate-600">Frontend siap.</p>
      </div>
    </main>
  )
}
```

- [ ] **Step 6: Env API URL**

`frontend/.env.example` (di-commit):
```
# Salin ke .env.local. Dipakai di server saja (tanpa prefix NEXT_PUBLIC_),
# jadi URL Go API tidak pernah dikirim ke browser.
API_URL=http://localhost:8080
```

```bash
cp .env.example .env.local
```

- [ ] **Step 7: LF untuk file frontend**

Tambahkan baris ini ke `.gitattributes` di root repo:
```
frontend/** text=auto eol=lf
```

- [ ] **Step 8: Verifikasi**

```bash
npm run dev
```
Buka http://localhost:3000 → kartu "Kredit APP Simulator / Frontend siap."
Hentikan (`Ctrl+C`), lalu:

```bash
npm run lint && npm run build
```
Expected: lint tanpa error, build sukses.

```bash
cd ~/bri-loan-app && git status --short
```
Expected: `.gitattributes` (modified) dan `frontend/` (untracked). Pastikan
`node_modules`, `.next`, dan `.env.local` **tidak** akan ikut
(`git status --short -uall frontend | grep -E 'node_modules|\.next|\.env\.local'`
harus kosong).

- [ ] **Step 9: Commit**

```bash
git add .gitattributes frontend
git commit -m "feat(frontend): scaffold Next.js 14 app with Tailwind and Vitest"
```

---

## Task 4: Tipe data + helper murni (TDD dengan Vitest)

**Files:**
- Create: `frontend/lib/types.ts`
- Create: `frontend/lib/format.ts`, `frontend/lib/format.test.ts`
- Create: `frontend/lib/validation.ts`, `frontend/lib/validation.test.ts`
- Create: `frontend/lib/cookies.ts`, `frontend/lib/cookies.test.ts`
- Create: `frontend/lib/routes.ts`, `frontend/lib/routes.test.ts`

**Interfaces:**
- Produces (semua murni, aman dipakai di server maupun client):
  - `types.ts`: `Role`, `LoanStatus`, `DOC_TYPES`, `DocType`, `User`,
    `LoanApplication`, `LoanDocument`, `FormState`
  - `format.ts`: `formatRupiah(amount: number): string`,
    `formatDate(iso: string): string`, `statusLabel(status: LoanStatus): string`,
    `docTypeLabel(docType: DocType): string`
  - `validation.ts`: `MAX_UPLOAD_BYTES`, `parseAmount(raw: FormDataEntryValue | null): number | null`,
    `isDocType(value: unknown): value is DocType`
  - `cookies.ts`: `extractToken(setCookieHeaders: string[]): string | null`
  - `routes.ts`: `homePathFor(role: Role): string`

Import di dalam `lib/` memakai path relatif (`./types`) supaya Vitest tidak
perlu konfigurasi alias `@/`.

- [ ] **Step 1: Tulis `lib/types.ts`**

```ts
export type Role = 'nasabah' | 'petugas'

export type LoanStatus = 'pending' | 'approved' | 'rejected'

export const DOC_TYPES = ['ktp', 'npwp', 'slip_gaji', 'rekening_koran', 'lainnya'] as const
export type DocType = (typeof DOC_TYPES)[number]

export type User = {
  id: string
  full_name: string
  email: string
  role: Role
  created_at: string
}

export type LoanApplication = {
  id: string
  user_id: string
  applicant_name?: string
  amount: number
  purpose: string
  status: LoanStatus
  reviewed_by?: string
  review_note?: string
  created_at: string
  updated_at: string
}

export type LoanDocument = {
  id: string
  loan_application_id: string
  doc_type: DocType
  file_path: string
  uploaded_at: string
}

// State yang dikembalikan Server Action ke form (lewat useFormState).
export type FormState = { error?: string; success?: string } | undefined
```

- [ ] **Step 2: Tulis test yang gagal**

`lib/format.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { formatDate, formatRupiah } from './format'

describe('formatRupiah', () => {
  it.each([
    [0, 'Rp 0'],
    [500, 'Rp 500'],
    [5000000, 'Rp 5.000.000'],
    [1234567890, 'Rp 1.234.567.890'],
  ])('formats %i as %s', (amount, expected) => {
    expect(formatRupiah(amount)).toBe(expected)
  })
})

describe('formatDate', () => {
  it('formats a date in Indonesian', () => {
    expect(formatDate('2026-09-24T06:36:51Z')).toBe('24 September 2026')
  })

  it('uses the Jakarta date, not the UTC date', () => {
    // 20:00 UTC = 03:00 WIB keesokan harinya
    expect(formatDate('2026-09-24T20:00:00Z')).toBe('25 September 2026')
  })
})
```

`lib/validation.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { isDocType, parseAmount } from './validation'

describe('parseAmount', () => {
  it.each([
    ['5000000', 5000000],
    [' 750000 ', 750000],
    ['1', 1],
  ])('accepts %j', (raw, expected) => {
    expect(parseAmount(raw)).toBe(expected)
  })

  it.each([[''], ['0'], ['-5'], ['5.5'], ['1e6'], ['abc'], ['5.000.000'], ['99999999999999999999']])(
    'rejects %j',
    (raw) => {
      expect(parseAmount(raw)).toBeNull()
    },
  )

  it('rejects a missing field', () => {
    expect(parseAmount(null)).toBeNull()
  })
})

describe('isDocType', () => {
  it.each(['ktp', 'npwp', 'slip_gaji', 'rekening_koran', 'lainnya'])('accepts %s', (value) => {
    expect(isDocType(value)).toBe(true)
  })

  it.each(['', 'KTP', 'paspor', null, 42])('rejects %j', (value) => {
    expect(isDocType(value)).toBe(false)
  })
})
```

`lib/cookies.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { extractToken } from './cookies'

describe('extractToken', () => {
  it('returns the token value from a Set-Cookie header', () => {
    expect(extractToken(['token=abc.def.ghi; Path=/; Max-Age=86400; HttpOnly; SameSite=Lax'])).toBe('abc.def.ghi')
  })

  it('ignores other cookies', () => {
    expect(extractToken(['theme=dark; Path=/', 'token=xyz; Path=/'])).toBe('xyz')
  })

  it('does not match cookies whose name only starts with "token"', () => {
    expect(extractToken(['tokenizer=1; Path=/'])).toBeNull()
  })

  it('returns null when there is no usable token cookie', () => {
    expect(extractToken([])).toBeNull()
    expect(extractToken(['token=; Path=/; Max-Age=0'])).toBeNull()
  })
})
```

`lib/routes.test.ts`:
```ts
import { describe, expect, it } from 'vitest'
import { homePathFor } from './routes'

describe('homePathFor', () => {
  it('sends nasabah to their dashboard', () => {
    expect(homePathFor('nasabah')).toBe('/dashboard')
  })

  it('sends petugas to the review list', () => {
    expect(homePathFor('petugas')).toBe('/review')
  })
})
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

```bash
cd ~/bri-loan-app/frontend && npm test
```
Expected: FAIL — `Failed to resolve import "./format"` (dan file lain).

- [ ] **Step 4: Implementasi**

`lib/format.ts`:
```ts
import type { DocType, LoanStatus } from './types'

const numberFormat = new Intl.NumberFormat('id-ID')

const dateFormat = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'Asia/Jakarta',
})

export function formatRupiah(amount: number): string {
  return `Rp ${numberFormat.format(amount)}`
}

export function formatDate(iso: string): string {
  return dateFormat.format(new Date(iso))
}

const STATUS_LABELS: Record<LoanStatus, string> = {
  pending: 'Menunggu Review',
  approved: 'Disetujui',
  rejected: 'Ditolak',
}

export function statusLabel(status: LoanStatus): string {
  return STATUS_LABELS[status]
}

const DOC_TYPE_LABELS: Record<DocType, string> = {
  ktp: 'KTP',
  npwp: 'NPWP',
  slip_gaji: 'Slip Gaji',
  rekening_koran: 'Rekening Koran',
  lainnya: 'Lainnya',
}

export function docTypeLabel(docType: DocType): string {
  return DOC_TYPE_LABELS[docType]
}
```

`lib/validation.ts`:
```ts
import { DOC_TYPES, type DocType } from './types'

export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024

// Jumlah pinjaman: bilangan bulat rupiah > 0, hanya digit (tanpa titik,
// desimal, tanda minus, atau notasi 1e6).
export function parseAmount(raw: FormDataEntryValue | null): number | null {
  if (typeof raw !== 'string') return null
  const trimmed = raw.trim()
  if (!/^\d+$/.test(trimmed)) return null
  const value = Number(trimmed)
  if (value <= 0 || !Number.isSafeInteger(value)) return null
  return value
}

export function isDocType(value: unknown): value is DocType {
  return typeof value === 'string' && (DOC_TYPES as readonly string[]).includes(value)
}
```

`lib/cookies.ts`:
```ts
// Mengambil nilai cookie "token" dari header Set-Cookie respons Go.
export function extractToken(setCookieHeaders: string[]): string | null {
  for (const header of setCookieHeaders) {
    const [pair] = header.split(';')
    const eq = pair.indexOf('=')
    if (eq === -1) continue
    const name = pair.slice(0, eq).trim()
    const value = pair.slice(eq + 1).trim()
    if (name === 'token' && value !== '') return value
  }
  return null
}
```

`lib/routes.ts`:
```ts
import type { Role } from './types'

export function homePathFor(role: Role): string {
  return role === 'petugas' ? '/review' : '/dashboard'
}
```

- [ ] **Step 5: Jalankan test, pastikan lolos**

```bash
npm test
```
Expected: semua test di 4 file PASS.

- [ ] **Step 6: Commit**

```bash
cd ~/bri-loan-app
git add frontend/lib
git commit -m "feat(frontend): add API types and tested formatting/validation helpers"
```

---

## Task 5: API client, sesi, dan halaman login

**Files:**
- Create: `frontend/lib/api.ts`, `frontend/lib/auth.ts`
- Create: `frontend/app/actions/auth.ts`
- Create: `frontend/components/SubmitButton.tsx`
- Create: `frontend/app/(auth)/layout.tsx`, `frontend/app/(auth)/login/page.tsx`, `frontend/app/(auth)/login/LoginForm.tsx`
- Modify: `frontend/app/page.tsx`

**Interfaces:**
- Consumes: `extractToken`, `homePathFor`, tipe `User`/`Role`/`FormState` (Task 4).
- Produces:
  - `apiFetch(path: string, init?: RequestInit): Promise<Response>` (server-only)
  - `getCurrentUser(): Promise<User | null>` (di-cache per request),
    `requireUser(role?: Role): Promise<User>` (redirect kalau tidak lolos)
  - Server Action `login(prev: FormState, formData: FormData): Promise<FormState>`
  - `<SubmitButton className? disabled?>children</SubmitButton>`

- [ ] **Step 1: `lib/api.ts`**

```ts
import 'server-only'
import { cookies } from 'next/headers'

function apiUrl(): string {
  const url = process.env.API_URL
  if (!url) throw new Error('API_URL is not set — copy frontend/.env.example to frontend/.env.local')
  return url
}

// Semua panggilan ke Go lewat sini: meneruskan cookie token milik user dan
// mematikan cache fetch Next.js (data tiap user berbeda).
export async function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers)
  const token = cookies().get('token')?.value
  if (token) headers.set('Cookie', `token=${token}`)

  return fetch(`${apiUrl()}${path}`, { ...init, headers, cache: 'no-store' })
}
```

`import 'server-only'` membuat build **gagal** kalau file ini tidak sengaja
di-import dari Client Component — mencegah logika server bocor ke bundle
browser.

- [ ] **Step 2: `lib/auth.ts`**

```ts
import 'server-only'
import { cache } from 'react'
import { redirect } from 'next/navigation'
import { apiFetch } from './api'
import { homePathFor } from './routes'
import type { Role, User } from './types'

// cache(): dalam SATU request, layout dan page yang sama-sama memanggil
// getCurrentUser() hanya memicu satu request ke Go.
export const getCurrentUser = cache(async (): Promise<User | null> => {
  const res = await apiFetch('/api/auth/me')
  if (res.status === 401) return null
  if (!res.ok) throw new Error(`GET /api/auth/me failed with status ${res.status}`)
  return res.json()
})

export async function requireUser(role?: Role): Promise<User> {
  const user = await getCurrentUser()
  if (!user) redirect('/login')
  if (role && user.role !== role) redirect(homePathFor(user.role))
  return user
}
```

- [ ] **Step 3: Server Action `app/actions/auth.ts`**

```ts
'use server'

import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { apiFetch } from '@/lib/api'
import { extractToken } from '@/lib/cookies'
import { homePathFor } from '@/lib/routes'
import type { FormState, User } from '@/lib/types'

export async function login(_prev: FormState, formData: FormData): Promise<FormState> {
  const email = String(formData.get('email') ?? '').trim()
  const password = String(formData.get('password') ?? '')
  if (!email || !password) return { error: 'Email dan password wajib diisi.' }

  const res = await apiFetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (res.status === 401) return { error: 'Email atau password salah.' }
  if (!res.ok) return { error: 'Login gagal, coba lagi.' }

  const token = extractToken(res.headers.getSetCookie())
  if (!token) return { error: 'Login gagal: server tidak mengirim token.' }

  cookies().set('token', token, {
    httpOnly: true,
    sameSite: 'lax',
    secure: process.env.NODE_ENV === 'production',
    path: '/',
    maxAge: 60 * 60 * 24,
  })

  const user: User = await res.json()
  // redirect() bekerja dengan melempar error khusus — jangan dibungkus try/catch.
  redirect(homePathFor(user.role))
}
```

- [ ] **Step 4: `components/SubmitButton.tsx`**

```tsx
'use client'

import { useFormStatus } from 'react-dom'

type Props = {
  children: React.ReactNode
  className?: string
  disabled?: boolean
}

export function SubmitButton({ children, className = 'btn-primary', disabled = false }: Props) {
  const { pending } = useFormStatus()
  return (
    <button type="submit" className={className} disabled={pending || disabled}>
      {pending ? 'Memproses…' : children}
    </button>
  )
}
```

- [ ] **Step 5: Layout auth `app/(auth)/layout.tsx`**

```tsx
import { redirect } from 'next/navigation'
import { getCurrentUser } from '@/lib/auth'
import { homePathFor } from '@/lib/routes'

export default async function AuthLayout({ children }: { children: React.ReactNode }) {
  const user = await getCurrentUser()
  if (user) redirect(homePathFor(user.role))

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="card w-full max-w-sm space-y-6">{children}</div>
    </main>
  )
}
```

- [ ] **Step 6: Form login (Client Component) `app/(auth)/login/LoginForm.tsx`**

```tsx
'use client'

import { useFormState } from 'react-dom'
import { login } from '@/app/actions/auth'
import { SubmitButton } from '@/components/SubmitButton'

export function LoginForm() {
  const [state, formAction] = useFormState(login, undefined)

  return (
    <form action={formAction} className="space-y-4">
      <div>
        <label htmlFor="email" className="label">Email</label>
        <input id="email" name="email" type="email" autoComplete="email" required className="input" />
      </div>
      <div>
        <label htmlFor="password" className="label">Password</label>
        <input id="password" name="password" type="password" autoComplete="current-password" required className="input" />
      </div>
      {state?.error && <p role="alert" className="alert-error">{state.error}</p>}
      <SubmitButton className="btn-primary w-full">Masuk</SubmitButton>
    </form>
  )
}
```

- [ ] **Step 7: Halaman login (Server Component) `app/(auth)/login/page.tsx`**

```tsx
import { LoginForm } from './LoginForm'

export default function LoginPage() {
  return (
    <>
      <div>
        <h1 className="text-2xl font-semibold">Masuk</h1>
        <p className="text-sm text-slate-600">Kredit APP Simulator</p>
      </div>
      <LoginForm />
    </>
  )
}
```

- [ ] **Step 8: Ganti `app/page.tsx` seluruhnya**

```tsx
import { redirect } from 'next/navigation'
import { getCurrentUser } from '@/lib/auth'
import { homePathFor } from '@/lib/routes'

export default async function Home() {
  const user = await getCurrentUser()
  redirect(user ? homePathFor(user.role) : '/login')
}
```

- [ ] **Step 9: Verifikasi manual** (Go API jalan dengan `.env`, lalu `npm run dev` di `frontend/`)

1. Buka http://localhost:3000 → otomatis ke `/login`.
2. Login `siti@example.com` + password salah → pesan "Email atau password salah.", tetap di `/login`.
3. Login `siti@example.com` / `rahasia123` → diarahkan ke `/dashboard` (masih **404** — dibuat di Task 7; yang diuji di sini redirect-nya).
4. DevTools → Application → Cookies → `http://localhost:3000`: ada cookie `token` dengan **HttpOnly ✓** dan SameSite `Lax`.
5. Buka `/login` lagi → langsung diarahkan ke `/dashboard` (sudah login).
6. Hapus cookie `token` di DevTools, buka `/` → kembali ke `/login`.

Lalu:
```bash
npm run lint && npm test && npm run build
```

- [ ] **Step 10: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add server-side API client, session helpers and login"
```

---

## Task 6: Halaman register

**Files:**
- Modify: `frontend/app/actions/auth.ts`
- Create: `frontend/app/(auth)/register/page.tsx`, `frontend/app/(auth)/register/RegisterForm.tsx`
- Modify: `frontend/app/(auth)/login/page.tsx`

**Interfaces:**
- Consumes: `apiFetch`, `SubmitButton`, `FormState`.
- Produces: Server Action `register(prev: FormState, formData: FormData): Promise<FormState>`;
  query `?registered=1` di `/login`.

- [ ] **Step 1: Tambahkan `register` di `app/actions/auth.ts`** (di bawah `login`)

```ts
export async function register(_prev: FormState, formData: FormData): Promise<FormState> {
  const fullName = String(formData.get('full_name') ?? '').trim()
  const email = String(formData.get('email') ?? '').trim()
  const password = String(formData.get('password') ?? '')
  if (!fullName || !email) return { error: 'Nama dan email wajib diisi.' }
  if (password.length < 8) return { error: 'Password minimal 8 karakter.' }

  const res = await apiFetch('/api/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ full_name: fullName, email, password }),
  })
  if (res.status === 409) return { error: 'Email sudah terdaftar.' }
  if (!res.ok) return { error: 'Registrasi gagal, periksa kembali data Anda.' }

  redirect('/login?registered=1')
}
```

- [ ] **Step 2: `app/(auth)/register/RegisterForm.tsx`**

```tsx
'use client'

import { useFormState } from 'react-dom'
import { register } from '@/app/actions/auth'
import { SubmitButton } from '@/components/SubmitButton'

export function RegisterForm() {
  const [state, formAction] = useFormState(register, undefined)

  return (
    <form action={formAction} className="space-y-4">
      <div>
        <label htmlFor="full_name" className="label">Nama Lengkap</label>
        <input id="full_name" name="full_name" autoComplete="name" required className="input" />
      </div>
      <div>
        <label htmlFor="email" className="label">Email</label>
        <input id="email" name="email" type="email" autoComplete="email" required className="input" />
      </div>
      <div>
        <label htmlFor="password" className="label">Password</label>
        <input id="password" name="password" type="password" autoComplete="new-password" minLength={8} required className="input" />
        <p className="mt-1 text-xs text-slate-500">Minimal 8 karakter.</p>
      </div>
      {state?.error && <p role="alert" className="alert-error">{state.error}</p>}
      <SubmitButton className="btn-primary w-full">Daftar</SubmitButton>
    </form>
  )
}
```

- [ ] **Step 3: `app/(auth)/register/page.tsx`**

```tsx
import Link from 'next/link'
import { RegisterForm } from './RegisterForm'

export default function RegisterPage() {
  return (
    <>
      <div>
        <h1 className="text-2xl font-semibold">Daftar Akun Nasabah</h1>
        <p className="text-sm text-slate-600">Kredit APP Simulator</p>
      </div>
      <RegisterForm />
      <p className="text-center text-sm text-slate-600">
        Sudah punya akun?{' '}
        <Link href="/login" className="font-medium text-blue-700 hover:underline">Masuk</Link>
      </p>
    </>
  )
}
```

- [ ] **Step 4: Ganti `app/(auth)/login/page.tsx` seluruhnya** (banner sukses + link daftar)

```tsx
import Link from 'next/link'
import { LoginForm } from './LoginForm'

export default function LoginPage({ searchParams }: { searchParams: { registered?: string } }) {
  return (
    <>
      <div>
        <h1 className="text-2xl font-semibold">Masuk</h1>
        <p className="text-sm text-slate-600">Kredit APP Simulator</p>
      </div>
      {searchParams.registered === '1' && (
        <p role="status" className="alert-success">Registrasi berhasil. Silakan masuk.</p>
      )}
      <LoginForm />
      <p className="text-center text-sm text-slate-600">
        Belum punya akun?{' '}
        <Link href="/register" className="font-medium text-blue-700 hover:underline">Daftar</Link>
      </p>
    </>
  )
}
```

- [ ] **Step 5: Verifikasi manual** (logout dulu dengan menghapus cookie `token` di DevTools)

1. `/login` → klik "Daftar" → `/register`.
2. Daftar dengan email `siti@example.com` → "Email sudah terdaftar."
3. Daftar `rina@example.com`, nama "Rina", password `rahasia123` → diarahkan ke `/login?registered=1` dengan banner hijau.
4. Login sebagai Rina → diarahkan ke `/dashboard` (masih 404 sampai Task 7).

```bash
npm run lint && npm run build
```

- [ ] **Step 6: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add nasabah registration page"
```

---

## Task 7: App shell + dashboard nasabah + logout

**Files:**
- Modify: `frontend/app/actions/auth.ts`
- Create: `frontend/lib/loans.ts`
- Create: `frontend/components/AppHeader.tsx`, `frontend/components/StatusBadge.tsx`, `frontend/components/LoanTable.tsx`
- Create: `frontend/app/(app)/layout.tsx`, `frontend/app/(app)/dashboard/page.tsx`

**Interfaces:**
- Consumes: `requireUser`, `apiFetch`, `homePathFor`, `formatRupiah`, `formatDate`, `statusLabel`.
- Produces:
  - Server Action `logout(): Promise<void>`
  - `getLoans(): Promise<LoanApplication[]>` (server-only)
  - `<StatusBadge status={LoanStatus} />`
  - `<LoanTable loans={LoanApplication[]} emptyMessage={string} showApplicant?={boolean} />`
  - `<AppHeader user={User} />`

- [ ] **Step 1: Tambahkan `logout` di `app/actions/auth.ts`** (di bawah `register`)

```ts
export async function logout(): Promise<void> {
  await apiFetch('/api/auth/logout', { method: 'POST' })
  cookies().delete('token')
  redirect('/login')
}
```

- [ ] **Step 2: `lib/loans.ts`**

```ts
import 'server-only'
import { redirect } from 'next/navigation'
import { apiFetch } from './api'
import type { LoanApplication } from './types'

export async function getLoans(): Promise<LoanApplication[]> {
  const res = await apiFetch('/api/loans')
  if (res.status === 401) redirect('/login')
  if (!res.ok) throw new Error(`GET /api/loans failed with status ${res.status}`)
  return res.json()
}
```

- [ ] **Step 3: `components/StatusBadge.tsx`**

```tsx
import { statusLabel } from '@/lib/format'
import type { LoanStatus } from '@/lib/types'

// Nama class ditulis utuh (bukan `bg-${warna}-100`) supaya Tailwind bisa
// menemukannya saat build.
const STYLES: Record<LoanStatus, string> = {
  pending: 'bg-amber-100 text-amber-800',
  approved: 'bg-green-100 text-green-800',
  rejected: 'bg-red-100 text-red-800',
}

export function StatusBadge({ status }: { status: LoanStatus }) {
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${STYLES[status]}`}>
      {statusLabel(status)}
    </span>
  )
}
```

- [ ] **Step 4: `components/LoanTable.tsx`**

```tsx
import Link from 'next/link'
import { StatusBadge } from '@/components/StatusBadge'
import { formatDate, formatRupiah } from '@/lib/format'
import type { LoanApplication } from '@/lib/types'

type Props = {
  loans: LoanApplication[]
  emptyMessage: string
  showApplicant?: boolean
}

export function LoanTable({ loans, emptyMessage, showApplicant = false }: Props) {
  if (loans.length === 0) {
    return <p className="card text-sm text-slate-600">{emptyMessage}</p>
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white shadow-sm">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50 text-left text-slate-600">
          <tr>
            <th className="px-4 py-3 font-medium">Tanggal</th>
            {showApplicant && <th className="px-4 py-3 font-medium">Pemohon</th>}
            <th className="px-4 py-3 font-medium">Jumlah</th>
            <th className="px-4 py-3 font-medium">Tujuan</th>
            <th className="px-4 py-3 font-medium">Status</th>
            <th className="px-4 py-3" />
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {loans.map((loan) => (
            <tr key={loan.id}>
              <td className="whitespace-nowrap px-4 py-3">{formatDate(loan.created_at)}</td>
              {showApplicant && <td className="px-4 py-3">{loan.applicant_name ?? '-'}</td>}
              <td className="whitespace-nowrap px-4 py-3">{formatRupiah(loan.amount)}</td>
              <td className="px-4 py-3">{loan.purpose}</td>
              <td className="px-4 py-3"><StatusBadge status={loan.status} /></td>
              <td className="px-4 py-3 text-right">
                <Link href={`/pinjaman/${loan.id}`} className="font-medium text-blue-700 hover:underline">Detail</Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

- [ ] **Step 5: `components/AppHeader.tsx`**

```tsx
import Link from 'next/link'
import { logout } from '@/app/actions/auth'
import { homePathFor } from '@/lib/routes'
import type { User } from '@/lib/types'

export function AppHeader({ user }: { user: User }) {
  const links =
    user.role === 'petugas'
      ? [{ href: '/review', label: 'Review Pengajuan' }]
      : [
          { href: '/dashboard', label: 'Pengajuan Saya' },
          { href: '/ajukan-pinjaman', label: 'Ajukan Pinjaman' },
        ]

  return (
    <header className="border-b border-slate-200 bg-white">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-4 px-4 py-3 sm:px-6">
        <div className="flex flex-wrap items-center gap-6">
          <Link href={homePathFor(user.role)} className="font-semibold text-blue-800">
            Kredit APP Simulator
          </Link>
          <nav className="flex gap-4 text-sm">
            {links.map((link) => (
              <Link key={link.href} href={link.href} className="text-slate-600 hover:text-slate-900">
                {link.label}
              </Link>
            ))}
          </nav>
        </div>
        <div className="flex items-center gap-3 text-sm">
          <span className="text-slate-700">{user.full_name}</span>
          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">{user.role}</span>
          {/* Server Action langsung dipasang di form: bekerja tanpa JavaScript di browser */}
          <form action={logout}>
            <button type="submit" className="btn-secondary">Keluar</button>
          </form>
        </div>
      </div>
    </header>
  )
}
```

- [ ] **Step 6: `app/(app)/layout.tsx`**

```tsx
import { AppHeader } from '@/components/AppHeader'
import { requireUser } from '@/lib/auth'

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const user = await requireUser()

  return (
    <div className="min-h-screen">
      <AppHeader user={user} />
      <main className="mx-auto max-w-5xl p-4 sm:p-6">{children}</main>
    </div>
  )
}
```

- [ ] **Step 7: `app/(app)/dashboard/page.tsx`**

```tsx
import Link from 'next/link'
import { LoanTable } from '@/components/LoanTable'
import { requireUser } from '@/lib/auth'
import { getLoans } from '@/lib/loans'

export default async function DashboardPage() {
  const user = await requireUser('nasabah')
  const loans = await getLoans()

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Pengajuan Saya</h1>
          <p className="text-sm text-slate-600">Halo, {user.full_name}. Berikut daftar pengajuan pinjaman Anda.</p>
        </div>
        <Link href="/ajukan-pinjaman" className="btn-primary">Ajukan Pinjaman</Link>
      </div>
      <LoanTable loans={loans} emptyMessage="Anda belum pernah mengajukan pinjaman." />
    </div>
  )
}
```

- [ ] **Step 8: Verifikasi manual**

1. Login Siti → `/dashboard` menampilkan header (nama "Siti", badge `nasabah`, tombol Keluar) dan tabel berisi pengajuan "Renovasi rumah" — jumlah `Rp 10.000.000`, badge merah "Ditolak".
2. Login Rina (dari Task 6) → pesan "Anda belum pernah mengajukan pinjaman."
3. Klik "Keluar" → kembali ke `/login`; cookie `token` hilang dari DevTools; buka `/dashboard` → diarahkan ke `/login`.
4. Login `budi2@example.com` (petugas) lalu buka `/dashboard` → diarahkan ke `/review` (masih 404 sampai Task 11).
5. View Page Source (`Ctrl+U`) di `/dashboard` sebagai Siti → teks "Renovasi rumah" sudah ada di HTML (Server Component, tanpa loading spinner client-side).

```bash
npm run lint && npm run build
```

- [ ] **Step 9: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add app shell, logout and nasabah dashboard"
```

---

## Task 8: Halaman detail pengajuan

**Files:**
- Modify: `frontend/lib/loans.ts`
- Create: `frontend/app/(app)/pinjaman/[id]/page.tsx`

**Interfaces:**
- Consumes: `requireUser`, `StatusBadge`, `formatRupiah`, `formatDate`, `docTypeLabel`, `homePathFor`.
- Produces: `getLoan(id: string): Promise<LoanApplication | null>`,
  `getLoanDocuments(id: string): Promise<LoanDocument[] | null>` (`null` = 404
  dari Go); halaman `/pinjaman/[id]` (Task 10 & 11 menambah form ke halaman ini).

- [ ] **Step 1: Tambahkan di `lib/loans.ts`**

Ubah import tipe menjadi:
```ts
import type { LoanApplication, LoanDocument } from './types'
```

Lalu tambahkan di bawah `getLoans`:
```ts
export async function getLoan(id: string): Promise<LoanApplication | null> {
  const res = await apiFetch(`/api/loans/${encodeURIComponent(id)}`)
  if (res.status === 401) redirect('/login')
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`GET /api/loans/{id} failed with status ${res.status}`)
  return res.json()
}

export async function getLoanDocuments(id: string): Promise<LoanDocument[] | null> {
  const res = await apiFetch(`/api/loans/${encodeURIComponent(id)}/documents`)
  if (res.status === 401) redirect('/login')
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`GET /api/loans/{id}/documents failed with status ${res.status}`)
  return res.json()
}
```

`encodeURIComponent`: `id` berasal dari URL browser. Tanpa encoding, id
seperti `../../auth/me` akan mengubah path yang dipanggil ke Go.

- [ ] **Step 2: `app/(app)/pinjaman/[id]/page.tsx`**

```tsx
import Link from 'next/link'
import { notFound } from 'next/navigation'
import { StatusBadge } from '@/components/StatusBadge'
import { requireUser } from '@/lib/auth'
import { docTypeLabel, formatDate, formatRupiah } from '@/lib/format'
import { getLoan, getLoanDocuments } from '@/lib/loans'
import { homePathFor } from '@/lib/routes'

export default async function LoanDetailPage({ params }: { params: { id: string } }) {
  const user = await requireUser()
  // Dua request ke Go berjalan paralel, bukan berurutan (menghindari waterfall).
  const [loan, documents] = await Promise.all([getLoan(params.id), getLoanDocuments(params.id)])
  if (!loan || !documents) notFound()

  return (
    <div className="space-y-6">
      <Link href={homePathFor(user.role)} className="text-sm text-blue-700 hover:underline">← Kembali</Link>

      <section className="card space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h1 className="text-2xl font-semibold">Detail Pengajuan</h1>
          <StatusBadge status={loan.status} />
        </div>
        <dl className="grid gap-4 sm:grid-cols-2">
          <Detail label="Pemohon" value={loan.applicant_name ?? '-'} />
          <Detail label="Jumlah Pinjaman" value={formatRupiah(loan.amount)} />
          <Detail label="Tujuan" value={loan.purpose} />
          <Detail label="Tanggal Pengajuan" value={formatDate(loan.created_at)} />
        </dl>
        {loan.review_note && (
          <div className="rounded-md bg-slate-50 p-3 text-sm">
            <p className="font-medium text-slate-700">Catatan Petugas</p>
            <p className="text-slate-600">{loan.review_note}</p>
          </div>
        )}
      </section>

      <section className="card space-y-4">
        <h2 className="text-lg font-semibold">Dokumen Pendukung</h2>
        {documents.length === 0 ? (
          <p className="text-sm text-slate-600">Belum ada dokumen.</p>
        ) : (
          <ul className="divide-y divide-slate-100 text-sm">
            {documents.map((doc) => (
              <li key={doc.id} className="flex justify-between gap-4 py-2">
                <span className="font-medium">{docTypeLabel(doc.doc_type)}</span>
                <span className="text-slate-500">{formatDate(doc.uploaded_at)}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-sm text-slate-500">{label}</dt>
      <dd className="font-medium">{value}</dd>
    </div>
  )
}
```

- [ ] **Step 3: Verifikasi manual**

1. Login Siti → `/dashboard` → klik "Detail" → `/pinjaman/2b2086f6-…`: Pemohon "Siti", `Rp 10.000.000`, badge "Ditolak", catatan "Slip gaji tidak valid", dan 2 dokumen (KTP, Lainnya).
2. Login Andi → buka URL detail milik Siti yang sama → halaman 404 bawaan Next.js (Go menjawab `404`, bukan `403` — keputusan backend Task 10).
3. Buka `/pinjaman/abc` → 404.

```bash
npm run lint && npm run build
```

- [ ] **Step 4: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add loan detail page with documents list"
```

---

## Task 9: Form pengajuan pinjaman

**Files:**
- Create: `frontend/app/actions/loans.ts`
- Create: `frontend/app/(app)/ajukan-pinjaman/page.tsx`, `frontend/app/(app)/ajukan-pinjaman/LoanForm.tsx`

**Interfaces:**
- Consumes: `apiFetch`, `parseAmount`, `formatRupiah`, `SubmitButton`, `requireUser`.
- Produces: Server Action `createLoan(prev: FormState, formData: FormData): Promise<FormState>`
  (redirect ke `/pinjaman/{id}` kalau sukses).

- [ ] **Step 1: `app/actions/loans.ts`**

```ts
'use server'

import { revalidatePath } from 'next/cache'
import { redirect } from 'next/navigation'
import { apiFetch } from '@/lib/api'
import type { FormState, LoanApplication } from '@/lib/types'
import { parseAmount } from '@/lib/validation'

// Server Action = endpoint HTTP publik. Validasi di sini untuk UX; aturan
// yang sebenarnya tetap ditegakkan Go.
export async function createLoan(_prev: FormState, formData: FormData): Promise<FormState> {
  const amount = parseAmount(formData.get('amount'))
  const purpose = String(formData.get('purpose') ?? '').trim()
  if (amount === null) return { error: 'Jumlah pinjaman harus berupa angka bulat lebih dari 0.' }
  if (!purpose) return { error: 'Tujuan pinjaman wajib diisi.' }

  const res = await apiFetch('/api/loans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ amount, purpose }),
  })
  if (res.status === 401) redirect('/login')
  if (!res.ok) return { error: 'Gagal mengajukan pinjaman, coba lagi.' }

  const loan: LoanApplication = await res.json()
  // Buang cache router di browser untuk dashboard supaya pengajuan baru langsung terlihat.
  revalidatePath('/dashboard')
  redirect(`/pinjaman/${loan.id}`)
}
```

- [ ] **Step 2: `app/(app)/ajukan-pinjaman/LoanForm.tsx`**

```tsx
'use client'

import { useState } from 'react'
import { useFormState } from 'react-dom'
import { createLoan } from '@/app/actions/loans'
import { SubmitButton } from '@/components/SubmitButton'
import { formatRupiah } from '@/lib/format'
import { parseAmount } from '@/lib/validation'

export function LoanForm() {
  const [state, formAction] = useFormState(createLoan, undefined)
  const [amountInput, setAmountInput] = useState('')
  const amount = parseAmount(amountInput)

  return (
    <form action={formAction} className="card space-y-4">
      <div>
        <label htmlFor="amount" className="label">Jumlah Pinjaman (Rp)</label>
        <input
          id="amount"
          name="amount"
          type="number"
          inputMode="numeric"
          min={1}
          step={1}
          required
          className="input"
          value={amountInput}
          onChange={(e) => setAmountInput(e.target.value)}
        />
        <p className="mt-1 text-xs text-slate-500">{amount !== null ? formatRupiah(amount) : 'Masukkan angka tanpa titik.'}</p>
      </div>
      <div>
        <label htmlFor="purpose" className="label">Tujuan Pinjaman</label>
        <textarea id="purpose" name="purpose" rows={3} required className="input" />
      </div>
      {state?.error && <p role="alert" className="alert-error">{state.error}</p>}
      <SubmitButton>Ajukan</SubmitButton>
    </form>
  )
}
```

- [ ] **Step 3: `app/(app)/ajukan-pinjaman/page.tsx`**

```tsx
import { requireUser } from '@/lib/auth'
import { LoanForm } from './LoanForm'

export default async function AjukanPinjamanPage() {
  await requireUser('nasabah')

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Ajukan Pinjaman</h1>
        <p className="text-sm text-slate-600">Setelah diajukan, Anda bisa mengupload dokumen pendukung di halaman detail.</p>
      </div>
      <LoanForm />
    </div>
  )
}
```

- [ ] **Step 4: Verifikasi manual**

1. Login Rina → "Ajukan Pinjaman". Ketik `15000000` → di bawah input muncul `Rp 15.000.000`.
2. Isi tujuan "Modal warung" → Ajukan → tombol berubah "Memproses…" → diarahkan ke halaman detail pengajuan baru (status "Menunggu Review", Pemohon "Rina").
3. Klik "Pengajuan Saya" di header → pengajuan baru langsung tampil tanpa refresh manual.
4. Tujuan diisi spasi saja (hapus atribut `required` sementara lewat DevTools, lalu submit) → "Tujuan pinjaman wajib diisi." — membuktikan validasi server tetap jalan walau validasi browser dilewati.
5. Login budi2 (petugas) → buka `/ajukan-pinjaman` → diarahkan ke `/review`.

```bash
npm run lint && npm run build
```

- [ ] **Step 5: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add loan application form"
```

---

## Task 10: Upload dokumen

**Files:**
- Modify: `frontend/next.config.mjs`
- Create: `frontend/app/actions/documents.ts`
- Create: `frontend/app/(app)/pinjaman/[id]/DocumentUploadForm.tsx`
- Modify: `frontend/app/(app)/pinjaman/[id]/page.tsx`

**Interfaces:**
- Consumes: `apiFetch`, `isDocType`, `MAX_UPLOAD_BYTES`, `DOC_TYPES`, `docTypeLabel`, `SubmitButton`.
- Produces: Server Action
  `uploadDocument(loanId: string, prev: FormState, formData: FormData): Promise<FormState>`
  (dipakai lewat `uploadDocument.bind(null, loanId)`).

- [ ] **Step 1: Naikkan batas body Server Action di `next.config.mjs`** (ganti seluruh file)

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  experimental: {
    serverActions: {
      // Default hanya 1 MB. Sedikit di atas batas Go (10 MB) supaya file yang
      // terlalu besar ditolak oleh pengecekan kita dengan pesan yang jelas.
      bodySizeLimit: '11mb',
    },
  },
}

export default nextConfig
```

- [ ] **Step 2: `app/actions/documents.ts`**

```ts
'use server'

import { revalidatePath } from 'next/cache'
import { redirect } from 'next/navigation'
import { apiFetch } from '@/lib/api'
import type { FormState } from '@/lib/types'
import { isDocType, MAX_UPLOAD_BYTES } from '@/lib/validation'

export async function uploadDocument(loanId: string, _prev: FormState, formData: FormData): Promise<FormState> {
  const docType = formData.get('doc_type')
  const file = formData.get('file')
  if (!isDocType(docType)) return { error: 'Pilih jenis dokumen.' }
  if (!(file instanceof File) || file.size === 0) return { error: 'Pilih file yang akan diupload.' }
  if (file.size > MAX_UPLOAD_BYTES) return { error: 'Ukuran file maksimal 10 MB.' }

  // Susun multipart baru untuk Go. Header Content-Type (beserta boundary)
  // diisi otomatis oleh fetch — jangan di-set manual.
  const body = new FormData()
  body.set('doc_type', docType)
  body.set('file', file, file.name)

  const res = await apiFetch(`/api/loans/${encodeURIComponent(loanId)}/documents`, { method: 'POST', body })
  if (res.status === 401) redirect('/login')
  if (res.status === 413) return { error: 'Ukuran file maksimal 10 MB.' }
  if (res.status === 403 || res.status === 404) return { error: 'Anda tidak dapat mengupload dokumen untuk pengajuan ini.' }
  if (!res.ok) return { error: 'Gagal mengupload dokumen, coba lagi.' }

  revalidatePath(`/pinjaman/${loanId}`)
  return { success: 'Dokumen berhasil diupload.' }
}
```

- [ ] **Step 3: `app/(app)/pinjaman/[id]/DocumentUploadForm.tsx`**

```tsx
'use client'

import { useEffect, useRef, useState } from 'react'
import { useFormState } from 'react-dom'
import { SubmitButton } from '@/components/SubmitButton'
import { docTypeLabel } from '@/lib/format'
import { DOC_TYPES, type FormState } from '@/lib/types'
import { MAX_UPLOAD_BYTES } from '@/lib/validation'

type Props = {
  action: (prev: FormState, formData: FormData) => Promise<FormState>
}

export function DocumentUploadForm({ action }: Props) {
  const [state, formAction] = useFormState(action, undefined)
  const [fileError, setFileError] = useState<string | null>(null)
  const formRef = useRef<HTMLFormElement>(null)

  // Kosongkan form setelah upload berhasil.
  useEffect(() => {
    if (state?.success) formRef.current?.reset()
  }, [state])

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    setFileError(file && file.size > MAX_UPLOAD_BYTES ? 'Ukuran file maksimal 10 MB.' : null)
  }

  return (
    <form ref={formRef} action={formAction} className="space-y-4 border-t border-slate-100 pt-4">
      <h3 className="font-medium">Upload Dokumen</h3>
      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor="doc_type" className="label">Jenis Dokumen</label>
          <select id="doc_type" name="doc_type" required defaultValue="" className="input">
            <option value="" disabled>Pilih jenis…</option>
            {DOC_TYPES.map((type) => (
              <option key={type} value={type}>{docTypeLabel(type)}</option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="file" className="label">File (maks. 10 MB)</label>
          <input id="file" name="file" type="file" required onChange={handleFileChange} className="block w-full text-sm" />
        </div>
      </div>
      {fileError && <p role="alert" className="alert-error">{fileError}</p>}
      {state?.error && <p role="alert" className="alert-error">{state.error}</p>}
      {state?.success && <p role="status" className="alert-success">{state.success}</p>}
      <SubmitButton disabled={fileError !== null}>Upload</SubmitButton>
    </form>
  )
}
```

- [ ] **Step 4: Pasang form di `app/(app)/pinjaman/[id]/page.tsx`**

Tambahkan import:
```tsx
import { uploadDocument } from '@/app/actions/documents'
import { DocumentUploadForm } from './DocumentUploadForm'
```

Di dalam `LoanDetailPage`, tepat setelah baris `if (!loan || !documents) notFound()`:
```tsx
  const isOwner = loan.user_id === user.id
```

Di section "Dokumen Pendukung", tepat sebelum `</section>` penutupnya:
```tsx
        {isOwner && <DocumentUploadForm action={uploadDocument.bind(null, loan.id)} />}
```

`uploadDocument.bind(null, loan.id)` dibuat di Server Component: `loan.id`
"dikunci" ke dalam action, sehingga Client Component tidak perlu (dan tidak
bisa) memilih loan lain.

- [ ] **Step 5: Verifikasi manual** (restart `npm run dev` — perubahan `next.config.mjs` butuh restart)

Siapkan file uji di Git Bash:
```bash
cd ~/bri-loan-app && echo "contoh slip gaji" > slip.txt && head -c 11000000 /dev/zero > besar.bin
```

1. Login Rina → detail pengajuan "Modal warung" → form "Upload Dokumen" tampil.
2. Pilih "Slip Gaji" + `slip.txt` → Upload → pesan hijau, form kosong lagi, dokumen "Slip Gaji" muncul di daftar.
3. Pilih `besar.bin` → pesan "Ukuran file maksimal 10 MB." langsung muncul dan tombol Upload nonaktif.
4. Login budi2 (petugas) → buka detail pengajuan Rina → form upload **tidak** tampil.
5. `ls backend/uploads/<id-pengajuan-rina>/` → ada file dengan nama acak `.txt`.

```bash
rm slip.txt besar.bin
cd frontend && npm run lint && npm run build
```

- [ ] **Step 6: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add document upload on loan detail page"
```

---

## Task 11: Halaman review petugas + form approve/reject

**Files:**
- Modify: `frontend/app/actions/loans.ts`
- Create: `frontend/app/(app)/review/page.tsx`
- Create: `frontend/app/(app)/pinjaman/[id]/ReviewForm.tsx`
- Modify: `frontend/app/(app)/pinjaman/[id]/page.tsx`

**Interfaces:**
- Consumes: `getLoans`, `LoanTable` (`showApplicant`), `SubmitButton`, `requireUser`.
- Produces: Server Action
  `reviewLoan(loanId: string, prev: FormState, formData: FormData): Promise<FormState>`;
  `<ReviewForm action currentStatus currentNote? />`.

- [ ] **Step 1: Tambahkan `reviewLoan` di `app/actions/loans.ts`** (di bawah `createLoan`)

```ts
export async function reviewLoan(loanId: string, _prev: FormState, formData: FormData): Promise<FormState> {
  const status = formData.get('status')
  const note = String(formData.get('review_note') ?? '').trim()
  if (status !== 'approved' && status !== 'rejected') return { error: 'Pilih keputusan: setujui atau tolak.' }
  if (status === 'rejected' && !note) return { error: 'Catatan wajib diisi saat menolak pengajuan.' }

  const res = await apiFetch(`/api/loans/${encodeURIComponent(loanId)}/review`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status, review_note: note || null }),
  })
  if (res.status === 401) redirect('/login')
  if (res.status === 403) return { error: 'Anda tidak berwenang mereview pengajuan ini.' }
  if (res.status === 404) return { error: 'Pengajuan tidak ditemukan.' }
  if (!res.ok) return { error: 'Gagal menyimpan keputusan, coba lagi.' }

  revalidatePath(`/pinjaman/${loanId}`)
  revalidatePath('/review')
  return { success: 'Keputusan berhasil disimpan.' }
}
```

- [ ] **Step 2: `app/(app)/review/page.tsx`**

```tsx
import { LoanTable } from '@/components/LoanTable'
import { requireUser } from '@/lib/auth'
import { getLoans } from '@/lib/loans'

export default async function ReviewPage() {
  await requireUser('petugas')
  const loans = await getLoans()
  const pending = loans.filter((loan) => loan.status === 'pending')
  const reviewed = loans.filter((loan) => loan.status !== 'pending')

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-semibold">Review Pengajuan</h1>
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Menunggu Review ({pending.length})</h2>
        <LoanTable loans={pending} showApplicant emptyMessage="Tidak ada pengajuan yang menunggu review." />
      </section>
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Sudah Direview ({reviewed.length})</h2>
        <LoanTable loans={reviewed} showApplicant emptyMessage="Belum ada pengajuan yang direview." />
      </section>
    </div>
  )
}
```

- [ ] **Step 3: `app/(app)/pinjaman/[id]/ReviewForm.tsx`**

```tsx
'use client'

import { useFormState } from 'react-dom'
import { SubmitButton } from '@/components/SubmitButton'
import type { FormState, LoanStatus } from '@/lib/types'

type Props = {
  action: (prev: FormState, formData: FormData) => Promise<FormState>
  currentStatus: LoanStatus
  currentNote?: string
}

export function ReviewForm({ action, currentStatus, currentNote }: Props) {
  const [state, formAction] = useFormState(action, undefined)

  return (
    <form action={formAction} className="space-y-4">
      <fieldset className="space-y-2">
        <legend className="label">Keputusan</legend>
        <label className="flex items-center gap-2 text-sm">
          <input type="radio" name="status" value="approved" required defaultChecked={currentStatus === 'approved'} />
          Setujui
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="radio" name="status" value="rejected" defaultChecked={currentStatus === 'rejected'} />
          Tolak
        </label>
      </fieldset>
      <div>
        <label htmlFor="review_note" className="label">
          Catatan <span className="font-normal text-slate-500">(wajib jika menolak)</span>
        </label>
        <textarea id="review_note" name="review_note" rows={3} defaultValue={currentNote ?? ''} className="input" />
      </div>
      {state?.error && <p role="alert" className="alert-error">{state.error}</p>}
      {state?.success && <p role="status" className="alert-success">{state.success}</p>}
      <SubmitButton>Simpan Keputusan</SubmitButton>
    </form>
  )
}
```

- [ ] **Step 4: Ganti `app/(app)/pinjaman/[id]/page.tsx` seluruhnya** (versi final: upload + review)

```tsx
import Link from 'next/link'
import { notFound } from 'next/navigation'
import { uploadDocument } from '@/app/actions/documents'
import { reviewLoan } from '@/app/actions/loans'
import { StatusBadge } from '@/components/StatusBadge'
import { requireUser } from '@/lib/auth'
import { docTypeLabel, formatDate, formatRupiah } from '@/lib/format'
import { getLoan, getLoanDocuments } from '@/lib/loans'
import { homePathFor } from '@/lib/routes'
import { DocumentUploadForm } from './DocumentUploadForm'
import { ReviewForm } from './ReviewForm'

export default async function LoanDetailPage({ params }: { params: { id: string } }) {
  const user = await requireUser()
  // Dua request ke Go berjalan paralel, bukan berurutan (menghindari waterfall).
  const [loan, documents] = await Promise.all([getLoan(params.id), getLoanDocuments(params.id)])
  if (!loan || !documents) notFound()

  const isOwner = loan.user_id === user.id
  const isPetugas = user.role === 'petugas'

  return (
    <div className="space-y-6">
      <Link href={homePathFor(user.role)} className="text-sm text-blue-700 hover:underline">← Kembali</Link>

      <section className="card space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h1 className="text-2xl font-semibold">Detail Pengajuan</h1>
          <StatusBadge status={loan.status} />
        </div>
        <dl className="grid gap-4 sm:grid-cols-2">
          <Detail label="Pemohon" value={loan.applicant_name ?? '-'} />
          <Detail label="Jumlah Pinjaman" value={formatRupiah(loan.amount)} />
          <Detail label="Tujuan" value={loan.purpose} />
          <Detail label="Tanggal Pengajuan" value={formatDate(loan.created_at)} />
        </dl>
        {loan.review_note && (
          <div className="rounded-md bg-slate-50 p-3 text-sm">
            <p className="font-medium text-slate-700">Catatan Petugas</p>
            <p className="text-slate-600">{loan.review_note}</p>
          </div>
        )}
      </section>

      <section className="card space-y-4">
        <h2 className="text-lg font-semibold">Dokumen Pendukung</h2>
        {documents.length === 0 ? (
          <p className="text-sm text-slate-600">Belum ada dokumen.</p>
        ) : (
          <ul className="divide-y divide-slate-100 text-sm">
            {documents.map((doc) => (
              <li key={doc.id} className="flex justify-between gap-4 py-2">
                <span className="font-medium">{docTypeLabel(doc.doc_type)}</span>
                <span className="text-slate-500">{formatDate(doc.uploaded_at)}</span>
              </li>
            ))}
          </ul>
        )}
        {isOwner && <DocumentUploadForm action={uploadDocument.bind(null, loan.id)} />}
      </section>

      {isPetugas && (
        <section className="card space-y-4">
          <h2 className="text-lg font-semibold">{loan.status === 'pending' ? 'Review Pengajuan' : 'Ubah Keputusan'}</h2>
          {isOwner ? (
            <p className="alert-error">Anda tidak dapat mereview pengajuan milik sendiri (prinsip maker-checker).</p>
          ) : (
            <ReviewForm
              action={reviewLoan.bind(null, loan.id)}
              currentStatus={loan.status}
              currentNote={loan.review_note}
            />
          )}
        </section>
      )}
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-sm text-slate-500">{label}</dt>
      <dd className="font-medium">{value}</dd>
    </div>
  )
}
```

- [ ] **Step 5: Verifikasi manual**

1. Login budi2 → `/review`: pengajuan "Modal warung" (Rina) di bagian "Menunggu Review", kolom Pemohon terisi.
2. Buka detail pengajuan Rina → pilih "Tolak" tanpa catatan → "Catatan wajib diisi saat menolak pengajuan."
3. Pilih "Setujui", catatan "Dokumen lengkap" → "Keputusan berhasil disimpan.", badge berubah "Disetujui", catatan tampil.
4. Klik "Review Pengajuan" di header → pengajuan Rina pindah ke "Sudah Direview".
5. Buka detail pengajuan milik budi2 sendiri (`4c7278d1-…`) → pesan maker-checker, **tidak** ada form.
6. Login Rina → dashboard: badge "Disetujui"; detail menampilkan "Catatan Petugas: Dokumen lengkap" dan **tidak** ada form review.
7. Login Siti → buka `/review` → diarahkan ke `/dashboard`.

```bash
npm run lint && npm run build
```

- [ ] **Step 6: Commit**

```bash
cd ~/bri-loan-app
git add frontend
git commit -m "feat(frontend): add petugas review list and approve/reject form"
```

---

## Task 12: Error/loading states, uji end-to-end, README

**Files:**
- Create: `frontend/app/not-found.tsx`, `frontend/app/error.tsx`, `frontend/app/(app)/loading.tsx`
- Modify: `README.md`

**Interfaces:**
- Consumes: semua halaman sebelumnya.
- Produces: frontend Fase 1 selesai + dokumentasi cara menjalankannya.

- [ ] **Step 1: `app/not-found.tsx`**

```tsx
import Link from 'next/link'

export default function NotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="card max-w-md space-y-3 text-center">
        <h1 className="text-xl font-semibold">Halaman tidak ditemukan</h1>
        <p className="text-sm text-slate-600">
          Halaman atau pengajuan yang Anda cari tidak ada, atau Anda tidak memiliki akses.
        </p>
        <Link href="/" className="btn-primary">Kembali ke Beranda</Link>
      </div>
    </main>
  )
}
```

Pesannya sengaja tidak membedakan "tidak ada" dan "tidak punya akses" —
konsisten dengan keputusan 404 di backend.

- [ ] **Step 2: `app/error.tsx`**

```tsx
'use client'

// Error boundary wajib Client Component: ia menangkap error saat render di
// browser dan menyediakan reset() untuk mencoba render ulang.
export default function Error({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="card max-w-md space-y-3 text-center">
        <h1 className="text-xl font-semibold">Terjadi kesalahan</h1>
        <p className="text-sm text-slate-600">Layanan sedang tidak dapat dihubungi. Coba lagi beberapa saat.</p>
        <button type="button" onClick={() => reset()} className="btn-primary">Coba Lagi</button>
      </div>
    </main>
  )
}
```

- [ ] **Step 3: `app/(app)/loading.tsx`**

```tsx
export default function Loading() {
  return (
    <div className="space-y-4" aria-busy="true" aria-live="polite">
      <div className="h-8 w-48 animate-pulse rounded bg-slate-200" />
      <div className="h-40 animate-pulse rounded-lg bg-slate-200" />
      <span className="sr-only">Memuat…</span>
    </div>
  )
}
```

- [ ] **Step 4: Uji error state**

1. Dengan frontend jalan, hentikan Go API (`Ctrl+C`), lalu buka `/dashboard` → halaman "Terjadi kesalahan".
2. Nyalakan Go lagi (dengan `.env`), klik "Coba Lagi" → dashboard tampil normal.
3. Buka `/tidak-ada` → halaman "Halaman tidak ditemukan".

- [ ] **Step 5: Uji end-to-end alur spec** (akun baru, dari nol)

1. Register `dewi@example.com` → login.
2. Ajukan pinjaman `Rp 25.000.000`, tujuan "Biaya pendidikan".
3. Upload KTP dan Slip Gaji.
4. Logout → login budi2 → `/review` → pengajuan Dewi di "Menunggu Review" → detail → dokumen KTP & Slip Gaji terlihat → Tolak dengan catatan "Slip gaji kurang dari 3 bulan".
5. Logout → login Dewi → badge "Ditolak" + catatan petugas terlihat.

- [ ] **Step 6: Pemeriksaan akhir**

```bash
cd ~/bri-loan-app/frontend
npm run lint && npm test && npm run build
cd ../backend && go test ./...
```
Expected: semua lolos.

- [ ] **Step 7: Update `README.md`**

Ganti kalimat pembuka (baris pertama setelah judul) menjadi:
```markdown
A fullstack app (Go API + Next.js) that simulates a bank loan application workflow: a **nasabah** (customer) submits a loan application and uploads supporting documents, and a **petugas** (loan officer) reviews it and approves or rejects it.
```

Di **Status**, ganti baris frontend menjadi:
```markdown
- [x] Phase 1 — Frontend (Next.js)
```

Di **Tech Stack**, tambahkan di akhir daftar:
```markdown
- **Next.js 14** (App Router, Server Components, Server Actions), **TypeScript**, **Tailwind CSS 3**
- **Vitest** — unit tests for pure frontend helpers
```

Di **Getting Started**, tambahkan sub-bagian ini tepat sebelum `### Creating a petugas account`:
````markdown
### Run the frontend
In a second terminal (the API must be running):
```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```
Open http://localhost:3000.
````

Di **Design & Security Decisions**, tambahkan kelompok baru sebelum **Testing**:
```markdown
**Frontend**
- **The browser never calls the Go API directly** — Server Components and Server Actions call it server-to-server, so the API URL is never exposed and no CORS configuration is needed.
- **Role checks in the frontend are for UX only** — hiding a button is not security; every request is authorized again by the Go API.
- **URL parameters are encoded before being forwarded to the API** (`encodeURIComponent`) — a crafted ID cannot change which API path is called.
```

Di **Running Tests**, ganti blok kode menjadi:
````markdown
```bash
cd backend && go test ./...
cd ../frontend && npm test
```
````

- [ ] **Step 8: Commit & push**

```bash
cd ~/bri-loan-app
git add frontend README.md
git commit -m "feat(frontend): add error/loading states and document frontend setup"
git push
```

---

Setelah Task 12, Fase 1 (backend + frontend) selesai. Fase 2 (approval
bertingkat + audit log review) mendapat spec terpisah.
