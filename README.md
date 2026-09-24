# Kredit APP Simulator

A backend API that simulates a bank loan application workflow: a **nasabah** (customer) submits a loan application and uploads supporting documents, and a **petugas** (loan officer) reviews it and approves or rejects it.

> **Disclaimer:** This is a personal learning project. It is not affiliated with any bank or financial institution, and all data is fictional.

## Status

- [x] Phase 1 — Backend API (auth, loan applications, review, document upload)
- [ ] Phase 1 — Frontend (Next.js) — in progress
- [ ] Phase 2+ — see [Roadmap](#known-limitations--roadmap)

## Tech Stack

- **Go** — `net/http` + [chi](https://github.com/go-chi/chi) router
- **PostgreSQL 16** (Docker Compose) — accessed with `sqlx` + raw SQL
- **golang-migrate** — schema migrations
- **JWT** (HS256) in an HttpOnly cookie, **bcrypt** password hashing

## Getting Started

### Prerequisites
- Go 1.27+
- Docker
- `golang-migrate` CLI:
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

### Run
```bash
cd backend

# 1. Start PostgreSQL
docker compose up -d

# 2. Apply migrations
migrate -path migrations -database "postgres://loanapp:devpassword@localhost:5432/loanapp?sslmode=disable" up

# 3. Configure environment
cp .env.example .env
# then set JWT_SECRET in .env, e.g. generate one with:
openssl rand -hex 32

# 4. Start the API (loads .env into the environment)
set -a; source .env; set +a
go run ./cmd/api
```

Check it is running: `curl http://localhost:8080/health` → `ok`

> The server refuses to start if `JWT_SECRET` is missing or shorter than 32 characters.

### Creating a petugas account
`/api/auth/register` always creates a `nasabah`: the role is never taken from client input, so nobody can sign themselves up as a loan officer. In a real bank, officer accounts are provisioned internally; here you promote a user directly in the database:

```bash
docker exec loanapp-postgres psql -U loanapp -d loanapp \
  -c "UPDATE users SET role = 'petugas' WHERE email = 'someone@example.com';"
```
Log in again afterwards — the role is stored inside the JWT.

## API

| Method | Path | Access | Notes |
|---|---|---|---|
| `GET` | `/health` | public | DB connectivity check |
| `POST` | `/api/auth/register` | public | always creates a `nasabah` |
| `POST` | `/api/auth/login` | public | sets the `token` HttpOnly cookie |
| `POST` | `/api/auth/logout` | public | clears the cookie |
| `POST` | `/api/loans` | authenticated | new application starts as `pending` |
| `GET` | `/api/loans` | authenticated | `nasabah`: own loans · `petugas`: all loans |
| `GET` | `/api/loans/{id}` | owner or `petugas` | others get `404` |
| `PATCH` | `/api/loans/{id}/review` | `petugas` | `approved` / `rejected`; cannot review own loan |
| `POST` | `/api/loans/{id}/documents` | applicant only | multipart, max 10 MB |
| `GET` | `/api/loans/{id}/documents` | owner or `petugas` | |

## Design & Security Decisions

**Authentication & authorization**
- **Role is never accepted from the client on register** — every self-registered user is a `nasabah`, so privilege cannot be self-assigned.
- **Generic `invalid email or password` on login** — the same message for unknown email and wrong password prevents user enumeration.
- **JWT in an HttpOnly cookie** — JavaScript in the browser cannot read the token, which limits the damage of an XSS bug.
- **`JWT_SECRET` is required at startup (min. 32 chars), with no default** — a missing or weak secret stops the server instead of silently signing tokens with a guessable key.
- **Only `HS256` tokens are accepted** — pinning the algorithm blocks tokens signed with an unexpected algorithm.
- **`404` instead of `403` for other users' loans** — responding `403` would confirm the loan exists; `404` prevents probing for valid IDs (IDOR).
- **Maker-checker** — a `petugas` cannot review their own loan application, so no one can approve their own credit.

**Business rules**
- **Rejecting requires a `review_note`** — a customer should always be told why their application was rejected.
- **Amounts are stored as `BIGINT` rupiah, not floats** — floating-point types cause rounding errors with money.

**File uploads**
- **Server-generated random file names** (`crypto/rand`) — the client's file name is never used in a disk path, which prevents path traversal and overwriting other files.
- **10 MB limit enforced while reading the body** (`http.MaxBytesReader` → `413`) — oversized uploads are cut off instead of being read in full.
- **`doc_type` is restricted to a fixed list** (`ktp`, `npwp`, `slip_gaji`, `rekening_koran`, `lainnya`) — keeps document data consistent for reviewers.
- **No orphan files** — if saving the database record fails, the file just written to disk is removed.

**Testing**
- **Authorization and validation rules are pure functions with unit tests** (`canAccessLoan`, `validateReviewRequest`, `safeExt`, `CheckSecret`) — security rules are tested without a database, so a regression fails fast.

## Running Tests

```bash
cd backend
go test ./...
```

## Known Limitations & Roadmap

**Known limitations (Phase 1)**
- A `petugas` can re-review a loan, which overwrites the previous decision; there is no audit history of review decisions yet.
- All file types are accepted and there is no download endpoint yet. Before adding one, files should be served with `Content-Disposition: attachment` and the accepted types reconsidered.
- Uploaded documents are stored on local disk; production would use object storage (e.g. S3 / MinIO).
- JWTs are stateless: logout clears the cookie but cannot revoke a token that was copied elsewhere, and a role change only applies after the next login.
- The auth cookie is not marked `Secure`, because local development runs over plain HTTP; production must use HTTPS with `Secure` cookies.

**Roadmap**
- **Phase 2** — multi-level approval (with an audit log of review decisions)
- **Phase 3** — credit scoring
- **Phase 4** — disbursement & installment schedules
- **Phase 5** — notifications
