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
