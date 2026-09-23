# Loan Application Backend (Fase 1) Implementation Plan

> **Mode eksekusi:** Plan ini TIDAK dieksekusi oleh subagent otonom.
> Mengikuti "Cara Kerja (Mentoring Workflow)" di spec: setiap task dikerjakan
> interaktif satu per satu — konsep dijelaskan, user (yang belajar) yang
> mengetik kodenya sendiri, lalu direview seperti code review sungguhan
> sebelum lanjut ke task berikutnya. Checkbox (`- [ ]`) dipakai untuk
> melacak progres antar sesi, bukan untuk instruksi ke subagent.

**Goal:** REST API Go yang berjalan penuh untuk Fase 1 — auth (register/
login/logout), pengajuan pinjaman (create/list/detail/review), dan upload
dokumen pendukung — bisa diuji end-to-end lewat curl/Postman sebelum
frontend disentuh.

**Architecture:** `net/http` + chi router untuk routing & middleware,
`sqlx` + driver `pgx` (stdlib mode) untuk akses PostgreSQL dengan raw SQL,
JWT (HS256) disimpan di HTTP-only cookie untuk auth, bcrypt untuk password
hashing. Struktur folder: `handler` (HTTP layer) → `repository` (query SQL)
→ `model` (struct data), plus `auth` (util hashing/JWT) dan `middleware`
(auth guard + role guard) sebagai lapisan lintas-fitur.

**Tech Stack:** Go 1.27.1 (module `go 1.23`), `github.com/go-chi/chi/v5`,
`github.com/jmoiron/sqlx`, `github.com/jackc/pgx/v5` (stdlib driver),
`golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5`,
PostgreSQL 16 via Docker Compose, `golang-migrate` untuk migrasi schema.

**Spec:** `docs/superpowers/specs/2026-09-23-loan-application-mvp-design.md`

## Global Constraints

- Password disimpan sebagai bcrypt hash, tidak pernah plaintext.
- File upload disimpan di disk lokal backend (`uploads/`); hanya path
  relatif yang disimpan di DB.
- Kepemilikan data (nasabah hanya akses pengajuan/dokumen miliknya sendiri)
  divalidasi di level handler/repository, bukan hanya di frontend nanti.
- Middleware `AuthRequired` (parse JWT dari cookie) dan `RequireRole`
  (khusus endpoint petugas) dipasang secara eksplisit per route group.
- `psql` tidak terpasang di host — migrasi dijalankan via `golang-migrate`
  CLI atau lewat `docker exec` ke dalam container, bukan psql manual.
- Backend (Go): unit test untuk business logic penting — password
  hashing/verify, JWT generate/verify, authorization check (nasabah tidak
  bisa akses data nasabah lain). Endpoint lain diverifikasi manual via
  curl (sesuai spec, automated HTTP test bukan prioritas Fase 1).
- Docker Compose menjalankan PostgreSQL untuk `backend/`.

---

## Task 1: Project Scaffold + PostgreSQL via Docker Compose

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/api/main.go`
- Create: `backend/docker-compose.yml`
- Create: `backend/.gitignore`

**Interfaces:**
- Produces: server listening on `:8080`, route `GET /health` → `200 "ok"`.
  Postgres container reachable at `localhost:5432`, db `loanapp`, user
  `loanapp`, password `devpassword`.

- [ ] **Step 1: Init Go module**

Run di folder `backend/` (buat foldernya dulu kalau belum ada):

```
go mod init loanapp
```

- [ ] **Step 2: Tulis `docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:16
    container_name: loanapp-postgres
    environment:
      POSTGRES_USER: loanapp
      POSTGRES_PASSWORD: devpassword
      POSTGRES_DB: loanapp
    ports:
      - "5432:5432"
    volumes:
      - loanapp-pgdata:/var/lib/postgresql/data

volumes:
  loanapp-pgdata:
```

- [ ] **Step 3: Tulis `.gitignore`**

```
uploads/
```

- [ ] **Step 4: Install chi router**

```
go get github.com/go-chi/chi/v5
```

- [ ] **Step 5: Tulis `cmd/api/main.go`**

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 6: Nyalakan Postgres, jalankan server, verifikasi**

```
docker compose up -d
go run ./cmd/api
```

Di terminal lain: `curl http://localhost:8080/health` → harus balas `ok`
dengan status 200, dan baris log request muncul di terminal server (dari
`chimw.Logger`).

- [ ] **Step 7: Commit**

```
git add backend/go.mod backend/go.sum backend/cmd backend/docker-compose.yml backend/.gitignore
git commit -m "feat(backend): scaffold Go API with chi router and Postgres compose"
```

---

## Task 2: Database Schema & Migrations

**Files:**
- Create: `backend/migrations/000001_init_schema.up.sql`
- Create: `backend/migrations/000001_init_schema.down.sql`

**Interfaces:**
- Produces: tabel `users`, `loan_applications`, `loan_documents` di
  database `loanapp` sesuai schema di spec.

- [ ] **Step 1: Install `golang-migrate` CLI**

```
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Pastikan `$(go env GOPATH)/bin` ada di PATH supaya command `migrate` bisa
dipanggil langsung.

- [ ] **Step 2: Tulis migration file naik (`up`)**

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
  doc_type TEXT NOT NULL,
  file_path TEXT NOT NULL,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] **Step 3: Tulis migration file turun (`down`)**

```sql
DROP TABLE loan_documents;
DROP TABLE loan_applications;
DROP TABLE users;
```

- [ ] **Step 4: Jalankan migrasi**

```
migrate -path migrations -database "postgres://loanapp:devpassword@localhost:5432/loanapp?sslmode=disable" up
```

- [ ] **Step 5: Verifikasi via container (bukan psql host)**

```
docker exec -it loanapp-postgres psql -U loanapp -d loanapp -c "\dt"
```

Harus muncul 3 tabel: `users`, `loan_applications`, `loan_documents`.

- [ ] **Step 6: Commit**

```
git add backend/migrations
git commit -m "feat(backend): add initial schema migration"
```

---

## Task 3: Database Connection Wiring

**Files:**
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Produces: `*sqlx.DB` yang bisa dipakai task-task berikutnya lewat
  parameter constructor (`NewXRepository(db)`).

- [ ] **Step 1: Install dependency**

```
go get github.com/jmoiron/sqlx github.com/jackc/pgx/v5
```

- [ ] **Step 2: Update `main.go`**

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://loanapp:devpassword@localhost:5432/loanapp?sslmode=disable"
	}

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 3: Verifikasi**

```
go run ./cmd/api
```

`curl http://localhost:8080/health` → `200 ok`. Lalu `docker compose stop
postgres`, curl lagi → harus `503`. Nyalakan lagi: `docker compose start
postgres`.

- [ ] **Step 4: Commit**

```
git add backend/cmd backend/go.mod backend/go.sum
git commit -m "feat(backend): wire sqlx/pgx database connection"
```

---

## Task 4: Password Hashing Util

**Files:**
- Create: `backend/internal/auth/password.go`
- Test: `backend/internal/auth/password_test.go`

**Interfaces:**
- Produces: `auth.HashPassword(plain string) (string, error)`,
  `auth.CheckPassword(plain, hash string) bool`.

- [ ] **Step 1: Tulis failing test**

```go
package auth

import "testing"

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash, err := HashPassword("s3cretPassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "s3cretPassword" {
		t.Fatal("hash must not equal plaintext")
	}
	if !CheckPassword("s3cretPassword", hash) {
		t.Fatal("CheckPassword should succeed with correct password")
	}
}

func TestCheckPassword_RejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("s3cretPassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if CheckPassword("wrongPassword", hash) {
		t.Fatal("CheckPassword should fail with wrong password")
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/auth/...`
Expected: FAIL — `undefined: HashPassword` (fungsinya belum ada).

- [ ] **Step 3: Install bcrypt & implementasi**

```
go get golang.org/x/crypto/bcrypt
```

```go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}
```

- [ ] **Step 4: Jalankan test, pastikan lolos**

Run: `go test ./internal/auth/... -v`
Expected: PASS untuk kedua test.

- [ ] **Step 5: Commit**

```
git add backend/internal/auth backend/go.mod backend/go.sum
git commit -m "feat(backend): add bcrypt password hashing util"
```

---

## Task 5: JWT Generate/Verify Util

**Files:**
- Create: `backend/internal/auth/jwt.go`
- Test: `backend/internal/auth/jwt_test.go`

**Interfaces:**
- Consumes: none.
- Produces: `auth.Claims{ UserID, Role string }`,
  `auth.GenerateToken(userID, role string) (string, error)`,
  `auth.VerifyToken(tokenString string) (*Claims, error)`,
  `auth.ErrInvalidToken`.

- [ ] **Step 1: Tulis failing test**

```go
package auth

import "testing"

func TestGenerateAndVerifyToken_RoundTrip(t *testing.T) {
	tokenString, err := GenerateToken("user-123", "nasabah")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := VerifyToken(tokenString)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Role != "nasabah" {
		t.Errorf("Role = %q, want %q", claims.Role, "nasabah")
	}
}

func TestVerifyToken_RejectsGarbageToken(t *testing.T) {
	_, err := VerifyToken("not-a-real-token")
	if err == nil {
		t.Fatal("expected error for garbage token, got nil")
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/auth/...`
Expected: FAIL — `undefined: GenerateToken`.

- [ ] **Step 3: Install dependency & implementasi**

```
go get github.com/golang-jwt/jwt/v5
```

```go
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	return []byte(secret)
}

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
```

- [ ] **Step 4: Jalankan test, pastikan lolos**

Run: `go test ./internal/auth/... -v`
Expected: PASS untuk semua 4 test di package `auth`.

- [ ] **Step 5: Commit**

```
git add backend/internal/auth backend/go.mod backend/go.sum
git commit -m "feat(backend): add JWT generate/verify util"
```

---

## Task 6: User Model, Repository & POST /api/auth/register

**Files:**
- Create: `backend/internal/model/user.go`
- Create: `backend/internal/repository/user_repository.go`
- Create: `backend/internal/handler/auth_handler.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `auth.HashPassword` (Task 4).
- Produces: `model.User`, `repository.NewUserRepository(db *sqlx.DB)
  *UserRepository` dengan method `Create` dan `FindByEmail`,
  `handler.NewAuthHandler(users *UserRepository) *AuthHandler` dengan
  method `Register`. Route `POST /api/auth/register`.

- [ ] **Step 1: Tulis `model/user.go`**

```go
package model

import "time"

type User struct {
	ID           string    `db:"id" json:"id"`
	FullName     string    `db:"full_name" json:"full_name"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
```

- [ ] **Step 2: Tulis `repository/user_repository.go`**

```go
package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"loanapp/internal/model"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, fullName, email, passwordHash, role string) (*model.User, error) {
	var u model.User
	query := `
		INSERT INTO users (full_name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, full_name, email, password_hash, role, created_at`
	err := r.db.QueryRowxContext(ctx, query, fullName, email, passwordHash, role).StructScan(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	query := `SELECT id, full_name, email, password_hash, role, created_at FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &u, query, email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
```

- [ ] **Step 3: Tulis `handler/auth_handler.go` (bagian Register)**

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"loanapp/internal/auth"
	"loanapp/internal/repository"
)

type AuthHandler struct {
	users *repository.UserRepository
}

func NewAuthHandler(users *repository.UserRepository) *AuthHandler {
	return &AuthHandler{users: users}
}

type registerRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.FullName == "" || req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "full_name, email are required and password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to process password", http.StatusInternalServerError)
		return
	}

	// Role selalu "nasabah" di endpoint publik ini — role tidak pernah
	// dipercaya dari input client, supaya orang tidak bisa daftar sebagai
	// "petugas" sendiri.
	user, err := h.users.Create(r.Context(), req.FullName, req.Email, hash, "nasabah")
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
```

- [ ] **Step 4: Wire ke `main.go`**

Tambahkan setelah `db, err := sqlx.Connect(...)`:

```go
	userRepo := repository.NewUserRepository(db)
	authHandler := handler.NewAuthHandler(userRepo)

	r.Post("/api/auth/register", authHandler.Register)
```

(tambahkan import `"loanapp/internal/handler"` dan `"loanapp/internal/repository"`)

- [ ] **Step 5: Verifikasi manual**

```
go run ./cmd/api
```

```
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Budi Santoso","email":"budi@example.com","password":"password123"}'
```

Harus balas `201` dengan JSON user (tanpa field `password_hash`). Ulangi
request yang sama persis → harus `409 email already registered`. Coba
password pendek (`"pw"`) → `400`.

- [ ] **Step 6: Commit**

```
git add backend/internal backend/cmd backend/go.mod backend/go.sum
git commit -m "feat(backend): add user registration endpoint"
```

---

## Task 7: POST /api/auth/login & POST /api/auth/logout

**Files:**
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `auth.CheckPassword`, `auth.GenerateToken` (Task 4 & 5),
  `UserRepository.FindByEmail` (Task 6).
- Produces: `AuthHandler.Login`, `AuthHandler.Logout`. Route
  `POST /api/auth/login` (set cookie `token`), `POST /api/auth/logout`
  (hapus cookie).

- [ ] **Step 1: Tambahkan `Login` dan `Logout` di `auth_handler.go`**

```go
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.users.FindByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}
```

Catatan: `FindByEmail` yang gagal (user tidak ada) dan password yang salah
sengaja balas pesan yang **sama persis** (`invalid email or password`) —
supaya penyerang tidak bisa menebak email mana yang terdaftar.

- [ ] **Step 2: Wire route di `main.go`**

```go
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)
```

- [ ] **Step 3: Verifikasi manual**

```
curl -i -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"budi@example.com","password":"password123"}'
```

Harus `200`, ada header `Set-Cookie: token=...; HttpOnly`. Coba password
salah → `401`. Lalu:

```
curl -i -X POST http://localhost:8080/api/auth/logout
```

Harus `204` dengan `Set-Cookie: token=; Max-Age=0` (menghapus cookie).

- [ ] **Step 4: Commit**

```
git add backend/internal/handler backend/cmd
git commit -m "feat(backend): add login and logout endpoints"
```

---

## Task 8: AuthRequired & RequireRole Middleware

**Files:**
- Create: `backend/internal/middleware/auth.go`
- Test: `backend/internal/middleware/auth_test.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `auth.VerifyToken` (Task 5).
- Produces: `middleware.AuthUser{ UserID, Role string }`,
  `middleware.AuthRequired(next http.Handler) http.Handler`,
  `middleware.UserFromContext(ctx) (AuthUser, bool)`,
  `middleware.RequireRole(role string) func(http.Handler) http.Handler`.
  Task 9+ pakai `mw.UserFromContext` untuk tahu siapa yang request.

- [ ] **Step 1: Tulis failing test**

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"loanapp/internal/auth"
)

func TestAuthRequired_RejectsMissingCookie(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	AuthRequired(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Error("next handler should not be called without a token cookie")
	}
}

func TestAuthRequired_AllowsValidToken(t *testing.T) {
	token, err := auth.GenerateToken("user-1", "nasabah")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	var gotUser AuthUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	rec := httptest.NewRecorder()

	AuthRequired(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUser.UserID != "user-1" || gotUser.Role != "nasabah" {
		t.Errorf("context user = %+v, want UserID=user-1 Role=nasabah", gotUser)
	}
}

func TestRequireRole_RejectsWrongRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(contextWithUser(req.Context(), AuthUser{UserID: "user-1", Role: "nasabah"}))
	rec := httptest.NewRecorder()

	RequireRole("petugas")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/middleware/...`
Expected: FAIL — package/fungsi belum ada.

- [ ] **Step 3: Implementasi `middleware/auth.go`**

```go
package middleware

import (
	"context"
	"net/http"

	"loanapp/internal/auth"
)

type contextKey string

const userContextKey contextKey = "authUser"

type AuthUser struct {
	UserID string
	Role   string
}

func contextWithUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(userContextKey).(AuthUser)
	return u, ok
}

func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := auth.VerifyToken(cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := contextWithUser(r.Context(), AuthUser{UserID: claims.UserID, Role: claims.Role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || user.Role != role {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

Catatan penting: `RequireRole` **harus** dipasang setelah `AuthRequired`
di chain middleware (butuh `AuthUser` sudah ada di context). Urutan salah
→ selalu 403 karena `UserFromContext` tidak ketemu apa-apa.

- [ ] **Step 4: Jalankan test, pastikan lolos**

Run: `go test ./internal/middleware/... -v`
Expected: PASS untuk ketiga test.

- [ ] **Step 5: Commit**

```
git add backend/internal/middleware backend/cmd
git commit -m "feat(backend): add AuthRequired and RequireRole middleware"
```

(Wiring middleware ke route group `/api/loans` dilakukan di Task 9, saat
route itu pertama kali dibuat.)

---

## Task 9: Loan Repository + POST/GET /api/loans

**Files:**
- Create: `backend/internal/model/loan.go`
- Create: `backend/internal/repository/loan_repository.go`
- Create: `backend/internal/handler/loan_handler.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `mw.AuthRequired`, `mw.UserFromContext` (Task 8).
- Produces: `model.LoanApplication`, `repository.NewLoanRepository(db)
  *LoanRepository` dengan `Create`, `ListByUser`, `ListAll`.
  `handler.NewLoanHandler(loans *LoanRepository) *LoanHandler` dengan
  `Create`, `List`. Route group `/api/loans` (dipakai lagi di Task 10-11).

- [ ] **Step 1: Tulis `model/loan.go`**

```go
package model

import "time"

type LoanApplication struct {
	ID         string    `db:"id" json:"id"`
	UserID     string    `db:"user_id" json:"user_id"`
	Amount     float64   `db:"amount" json:"amount"`
	Purpose    string    `db:"purpose" json:"purpose"`
	Status     string    `db:"status" json:"status"`
	ReviewedBy *string   `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewNote *string   `db:"review_note" json:"review_note,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}
```

- [ ] **Step 2: Tulis `repository/loan_repository.go`**

```go
package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"loanapp/internal/model"
)

type LoanRepository struct {
	db *sqlx.DB
}

func NewLoanRepository(db *sqlx.DB) *LoanRepository {
	return &LoanRepository{db: db}
}

func (r *LoanRepository) Create(ctx context.Context, userID string, amount float64, purpose string) (*model.LoanApplication, error) {
	var loan model.LoanApplication
	query := `
		INSERT INTO loan_applications (user_id, amount, purpose)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, amount, purpose, status, reviewed_by, review_note, created_at, updated_at`
	err := r.db.QueryRowxContext(ctx, query, userID, amount, purpose).StructScan(&loan)
	if err != nil {
		return nil, err
	}
	return &loan, nil
}

func (r *LoanRepository) ListByUser(ctx context.Context, userID string) ([]model.LoanApplication, error) {
	loans := []model.LoanApplication{}
	query := `
		SELECT id, user_id, amount, purpose, status, reviewed_by, review_note, created_at, updated_at
		FROM loan_applications WHERE user_id = $1 ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &loans, query, userID); err != nil {
		return nil, err
	}
	return loans, nil
}

func (r *LoanRepository) ListAll(ctx context.Context) ([]model.LoanApplication, error) {
	loans := []model.LoanApplication{}
	query := `
		SELECT id, user_id, amount, purpose, status, reviewed_by, review_note, created_at, updated_at
		FROM loan_applications ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &loans, query); err != nil {
		return nil, err
	}
	return loans, nil
}
```

- [ ] **Step 3: Tulis `handler/loan_handler.go` (bagian Create & List)**

```go
package handler

import (
	"encoding/json"
	"net/http"

	mw "loanapp/internal/middleware"
	"loanapp/internal/repository"
)

type LoanHandler struct {
	loans *repository.LoanRepository
}

func NewLoanHandler(loans *repository.LoanRepository) *LoanHandler {
	return &LoanHandler{loans: loans}
}

type createLoanRequest struct {
	Amount  float64 `json:"amount"`
	Purpose string  `json:"purpose"`
}

func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 || req.Purpose == "" {
		http.Error(w, "amount must be positive and purpose is required", http.StatusBadRequest)
		return
	}

	loan, err := h.loans.Create(r.Context(), user.UserID, req.Amount, req.Purpose)
	if err != nil {
		http.Error(w, "failed to create loan application", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loan)
}

func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var (
		result any
		err    error
	)
	if user.Role == "petugas" {
		result, err = h.loans.ListAll(r.Context())
	} else {
		result, err = h.loans.ListByUser(r.Context(), user.UserID)
	}
	if err != nil {
		http.Error(w, "failed to list loan applications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 4: Wire route group di `main.go`**

```go
	loanRepo := repository.NewLoanRepository(db)
	loanHandler := handler.NewLoanHandler(loanRepo)

	r.Route("/api/loans", func(r chi.Router) {
		r.Use(mw.AuthRequired)
		r.Post("/", loanHandler.Create)
		r.Get("/", loanHandler.List)
	})
```

(tambahkan import `mw "loanapp/internal/middleware"`)

- [ ] **Step 5: Verifikasi manual**

Login sebagai `budi@example.com` (Task 7), ambil cookie dari respons
(`-c cookies.txt` di curl), lalu:

```
curl -b cookies.txt -X POST http://localhost:8080/api/loans \
  -H "Content-Type: application/json" \
  -d '{"amount":5000000,"purpose":"Modal usaha"}'
```

Harus `201`. Lalu:

```
curl -b cookies.txt http://localhost:8080/api/loans
```

Harus `200` dengan array berisi 1 pengajuan. Register+login user kedua,
buat pengajuan lain, pastikan list milik user pertama **tidak** memuat
punya user kedua.

- [ ] **Step 6: Commit**

```
git add backend/internal backend/cmd
git commit -m "feat(backend): add loan application create and list endpoints"
```

---

## Task 10: GET /api/loans/{id} + Authorization Check (Unit Test)

**Files:**
- Modify: `backend/internal/repository/loan_repository.go`
- Modify: `backend/internal/handler/loan_handler.go`
- Test: `backend/internal/handler/loan_handler_test.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Produces: `LoanRepository.GetByID(ctx, id) (*model.LoanApplication,
  error)`, `canAccessLoan(user mw.AuthUser, loan *model.LoanApplication)
  bool` (dipakai lagi di Task 12 untuk list dokumen), `LoanHandler.Get`.
  Route `GET /api/loans/{id}`.

- [ ] **Step 1: Tambahkan `GetByID` di repository**

```go
func (r *LoanRepository) GetByID(ctx context.Context, id string) (*model.LoanApplication, error) {
	var loan model.LoanApplication
	query := `
		SELECT id, user_id, amount, purpose, status, reviewed_by, review_note, created_at, updated_at
		FROM loan_applications WHERE id = $1`
	if err := r.db.GetContext(ctx, &loan, query, id); err != nil {
		return nil, err
	}
	return &loan, nil
}
```

- [ ] **Step 2: Tulis failing test untuk logic otorisasi**

```go
package handler

import (
	"testing"

	mw "loanapp/internal/middleware"
	"loanapp/internal/model"
)

func TestCanAccessLoan_OwnerCanAccessOwnLoan(t *testing.T) {
	user := mw.AuthUser{UserID: "user-1", Role: "nasabah"}
	loan := &model.LoanApplication{UserID: "user-1"}

	if !canAccessLoan(user, loan) {
		t.Fatal("owner should be able to access their own loan")
	}
}

func TestCanAccessLoan_NasabahCannotAccessOthersLoan(t *testing.T) {
	user := mw.AuthUser{UserID: "user-1", Role: "nasabah"}
	loan := &model.LoanApplication{UserID: "user-2"}

	if canAccessLoan(user, loan) {
		t.Fatal("nasabah should not be able to access another nasabah's loan")
	}
}

func TestCanAccessLoan_PetugasCanAccessAnyLoan(t *testing.T) {
	user := mw.AuthUser{UserID: "petugas-1", Role: "petugas"}
	loan := &model.LoanApplication{UserID: "user-2"}

	if !canAccessLoan(user, loan) {
		t.Fatal("petugas should be able to access any loan")
	}
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

Run: `go test ./internal/handler/...`
Expected: FAIL — `undefined: canAccessLoan`.

- [ ] **Step 4: Implementasi `canAccessLoan` + handler `Get`**

Tambahkan di `loan_handler.go`:

```go
func canAccessLoan(user mw.AuthUser, loan *model.LoanApplication) bool {
	return user.Role == "petugas" || loan.UserID == user.UserID
}

func (h *LoanHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	loan, err := h.loans.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}

	if !canAccessLoan(user, loan) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loan)
}
```

Tambahkan import `"github.com/go-chi/chi/v5"` dan `"loanapp/internal/model"`
di `loan_handler.go`.

- [ ] **Step 5: Jalankan test, pastikan lolos**

Run: `go test ./internal/handler/... -v`
Expected: PASS untuk ketiga test `TestCanAccessLoan_*`.

- [ ] **Step 6: Wire route di `main.go`**

```go
		r.Get("/{id}", loanHandler.Get)
```

(baris ini masuk ke dalam `r.Route("/api/loans", ...)` yang sudah ada dari
Task 9)

- [ ] **Step 7: Verifikasi manual**

```
curl -b cookies.txt http://localhost:8080/api/loans/<id-milik-sendiri>
```

Harus `200`. Coba pakai cookie user lain untuk id yang sama → `403`. Coba
id acak yang tidak ada → `404`.

- [ ] **Step 8: Commit**

```
git add backend/internal backend/cmd
git commit -m "feat(backend): add loan detail endpoint with ownership check"
```

---

## Task 11: PATCH /api/loans/{id}/review (Petugas Only)

**Files:**
- Modify: `backend/internal/repository/loan_repository.go`
- Modify: `backend/internal/handler/loan_handler.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `mw.RequireRole("petugas")` (Task 8).
- Produces: `LoanRepository.Review`, `LoanHandler.Review`. Route
  `PATCH /api/loans/{id}/review`.

- [ ] **Step 1: Tambahkan `Review` di repository**

```go
func (r *LoanRepository) Review(ctx context.Context, id, reviewerID, status string, note *string) (*model.LoanApplication, error) {
	var loan model.LoanApplication
	query := `
		UPDATE loan_applications
		SET status = $1, reviewed_by = $2, review_note = $3, updated_at = now()
		WHERE id = $4
		RETURNING id, user_id, amount, purpose, status, reviewed_by, review_note, created_at, updated_at`
	err := r.db.QueryRowxContext(ctx, query, status, reviewerID, note, id).StructScan(&loan)
	if err != nil {
		return nil, err
	}
	return &loan, nil
}
```

- [ ] **Step 2: Tambahkan `Review` di handler**

```go
type reviewRequest struct {
	Status     string  `json:"status"`
	ReviewNote *string `json:"review_note"`
}

func (h *LoanHandler) Review(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req reviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status != "approved" && req.Status != "rejected" {
		http.Error(w, `status must be "approved" or "rejected"`, http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	loan, err := h.loans.Review(r.Context(), id, user.UserID, req.Status, req.ReviewNote)
	if err != nil {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loan)
}
```

- [ ] **Step 3: Wire route di `main.go`**

Ganti isi `r.Route("/api/loans", ...)` jadi:

```go
	r.Route("/api/loans", func(r chi.Router) {
		r.Use(mw.AuthRequired)
		r.Post("/", loanHandler.Create)
		r.Get("/", loanHandler.List)
		r.Get("/{id}", loanHandler.Get)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("petugas"))
			r.Patch("/{id}/review", loanHandler.Review)
		})
	})
```

- [ ] **Step 4: Verifikasi manual**

Karena belum ada endpoint untuk membuat akun petugas (Fase 1 memang tidak
punya self-register untuk role itu — dibuat manual), promosikan satu user
lewat container:

```
docker exec -it loanapp-postgres psql -U loanapp -d loanapp \
  -c "UPDATE users SET role = 'petugas' WHERE email = 'budi@example.com';"
```

Login ulang user itu (JWT lama masih bawa role lama), lalu:

```
curl -b cookies.txt -X PATCH http://localhost:8080/api/loans/<id>/review \
  -H "Content-Type: application/json" \
  -d '{"status":"approved","review_note":"Dokumen lengkap"}'
```

Harus `200` dengan status berubah. Coba dari user nasabah biasa → `403`.
Coba `status` tidak valid (misal `"ok"`) → `400`.

- [ ] **Step 5: Commit**

```
git add backend/internal backend/cmd
git commit -m "feat(backend): add petugas loan review endpoint"
```

---

## Task 12: Document Upload & List

**Files:**
- Create: `backend/internal/model/document.go`
- Create: `backend/internal/repository/document_repository.go`
- Create: `backend/internal/handler/document_handler.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `LoanRepository.GetByID`, `canAccessLoan` (Task 10).
- Produces: `model.LoanDocument`, `repository.NewDocumentRepository(db)
  *DocumentRepository` dengan `Create`, `ListByLoan`.
  `handler.NewDocumentHandler(loans, documents, uploadDir) *DocumentHandler`
  dengan `Upload`, `List`. Route `POST /api/loans/{id}/documents`,
  `GET /api/loans/{id}/documents`.

- [ ] **Step 1: Tulis `model/document.go`**

```go
package model

import "time"

type LoanDocument struct {
	ID                string    `db:"id" json:"id"`
	LoanApplicationID string    `db:"loan_application_id" json:"loan_application_id"`
	DocType           string    `db:"doc_type" json:"doc_type"`
	FilePath          string    `db:"file_path" json:"file_path"`
	UploadedAt        time.Time `db:"uploaded_at" json:"uploaded_at"`
}
```

- [ ] **Step 2: Tulis `repository/document_repository.go`**

```go
package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"loanapp/internal/model"
)

type DocumentRepository struct {
	db *sqlx.DB
}

func NewDocumentRepository(db *sqlx.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) Create(ctx context.Context, loanID, docType, filePath string) (*model.LoanDocument, error) {
	var doc model.LoanDocument
	query := `
		INSERT INTO loan_documents (loan_application_id, doc_type, file_path)
		VALUES ($1, $2, $3)
		RETURNING id, loan_application_id, doc_type, file_path, uploaded_at`
	err := r.db.QueryRowxContext(ctx, query, loanID, docType, filePath).StructScan(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *DocumentRepository) ListByLoan(ctx context.Context, loanID string) ([]model.LoanDocument, error) {
	docs := []model.LoanDocument{}
	query := `
		SELECT id, loan_application_id, doc_type, file_path, uploaded_at
		FROM loan_documents WHERE loan_application_id = $1 ORDER BY uploaded_at`
	if err := r.db.SelectContext(ctx, &docs, query, loanID); err != nil {
		return nil, err
	}
	return docs, nil
}
```

- [ ] **Step 3: Tulis `handler/document_handler.go`**

```go
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	mw "loanapp/internal/middleware"
	"loanapp/internal/repository"
)

type DocumentHandler struct {
	loans     *repository.LoanRepository
	documents *repository.DocumentRepository
	uploadDir string
}

func NewDocumentHandler(loans *repository.LoanRepository, documents *repository.DocumentRepository, uploadDir string) *DocumentHandler {
	return &DocumentHandler{loans: loans, documents: documents, uploadDir: uploadDir}
}

const maxUploadSize = 10 << 20 // 10 MB

func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	loanID := chi.URLParam(r, "id")
	loan, err := h.loans.GetByID(r.Context(), loanID)
	if err != nil {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}
	if loan.UserID != user.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large or invalid form", http.StatusBadRequest)
		return
	}

	docType := strings.TrimSpace(r.FormValue("doc_type"))
	if docType == "" {
		http.Error(w, "doc_type is required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	loanDir := filepath.Join(h.uploadDir, loanID)
	if err := os.MkdirAll(loanDir, 0o755); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	// filepath.Base membuang path apa pun yang menempel di nama file asli
	// (mis. "../../etc/passwd") sebelum dipakai bikin path di disk.
	safeName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(header.Filename))
	relativePath := filepath.Join(loanID, safeName)
	fullPath := filepath.Join(h.uploadDir, relativePath)

	dst, err := os.Create(fullPath)
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	doc, err := h.documents.Create(r.Context(), loanID, docType, relativePath)
	if err != nil {
		http.Error(w, "failed to record document", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	loanID := chi.URLParam(r, "id")
	loan, err := h.loans.GetByID(r.Context(), loanID)
	if err != nil {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}
	if !canAccessLoan(user, loan) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	docs, err := h.documents.ListByLoan(r.Context(), loanID)
	if err != nil {
		http.Error(w, "failed to list documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}
```

- [ ] **Step 4: Wire route di `main.go`**

```go
	docRepo := repository.NewDocumentRepository(db)
	documentHandler := handler.NewDocumentHandler(loanRepo, docRepo, "uploads")
	os.MkdirAll("uploads", 0o755)
```

Tambahkan dua baris di dalam `r.Route("/api/loans", ...)`, sebelum grup
`RequireRole("petugas")`:

```go
		r.Post("/{id}/documents", documentHandler.Upload)
		r.Get("/{id}/documents", documentHandler.List)
```

- [ ] **Step 5: Verifikasi manual**

```
curl -b cookies.txt -X POST http://localhost:8080/api/loans/<id>/documents \
  -F "doc_type=ktp" \
  -F "file=@/path/ke/file/contoh.jpg"
```

Harus `201`. Cek file benar-benar tersimpan di
`backend/uploads/<id>/<timestamp>-contoh.jpg`. Lalu:

```
curl -b cookies.txt http://localhost:8080/api/loans/<id>/documents
```

Harus `200` dengan array berisi dokumen itu. Coba upload pakai cookie
nasabah lain ke id pinjaman ini → `403`.

- [ ] **Step 6: Commit**

```
git add backend/internal backend/cmd
git commit -m "feat(backend): add loan document upload and list endpoints"
```

---

Setelah Task 12 selesai dan semua verifikasi manual + `go test ./...`
lolos, backend Fase 1 selesai dan siap jadi basis plan frontend
(Next.js) yang akan ditulis terpisah.
