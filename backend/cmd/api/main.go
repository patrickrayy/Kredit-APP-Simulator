package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"loanapp/internal/auth"
	"loanapp/internal/handler"
	mw "loanapp/internal/middleware"
	"loanapp/internal/repository"
)

func main() {
	if err := auth.CheckSecret(); err != nil {
		log.Fatal(err)
	}
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

	userRepo := repository.NewUserRepository(db)
	authHandler := handler.NewAuthHandler(userRepo)

	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)
	r.With(mw.AuthRequired).Get("/api/auth/me", authHandler.Me)

	loanRepo := repository.NewLoanRepository(db)
	loanHandler := handler.NewLoanHandler(loanRepo)

	docRepo := repository.NewDocumentRepository(db)
	documentHandler := handler.NewDocumentHandler(loanRepo, docRepo, "uploads")
	if err := os.MkdirAll("uploads", 0o755); err != nil {
		log.Fatalf("create upload dir failed: %v", err)
	}

	r.Route("/api/loans", func(r chi.Router) {
		r.Use(mw.AuthRequired)
		r.Post("/", loanHandler.Create)
		r.Get("/", loanHandler.List)
		r.Get("/{id}", loanHandler.Get)
		r.Post("/{id}/documents", documentHandler.Upload)
		r.Get("/{id}/documents", documentHandler.List)

		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("petugas"))
			r.Patch("/{id}/review", loanHandler.Review)
		})
	})

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
