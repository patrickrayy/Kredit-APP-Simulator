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

func (r *LoanRepository) Create(ctx context.Context, userID string, amount int64, purpose string) (*model.LoanApplication, error) {
	var loan model.LoanApplication
	query := `
	INSERT INTO loan_applications(user_id, amount, purpose)
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
	FROM loan_applications ORDER BY created_at DESC `
	if err := r.db.SelectContext(ctx, &loans, query); err != nil {
		return nil, err
	}
	return loans, nil
}

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
