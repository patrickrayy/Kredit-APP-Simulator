package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"loanapp/internal/model"
)

// loanSelect dipakai bersama oleh semua query baca supaya daftar kolom dan
// JOIN ke users hanya ditulis sekali.
const loanSelect = `
	SELECT la.id, la.user_id, u.full_name AS applicant_name, la.amount, la.purpose,
	       la.status, la.reviewed_by, la.review_note, la.created_at, la.updated_at
	FROM loan_applications la
	JOIN users u ON u.id = la.user_id`

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
	query := loanSelect + ` WHERE la.user_id = $1 ORDER BY la.created_at DESC`

	if err := r.db.SelectContext(ctx, &loans, query, userID); err != nil {
		return nil, err
	}
	return loans, nil
}

func (r *LoanRepository) ListAll(ctx context.Context) ([]model.LoanApplication, error) {
	loans := []model.LoanApplication{}
	query := loanSelect + ` ORDER BY la.created_at DESC`

	if err := r.db.SelectContext(ctx, &loans, query); err != nil {
		return nil, err
	}
	return loans, nil
}

func (r *LoanRepository) GetByID(ctx context.Context, id string) (*model.LoanApplication, error) {
	var loan model.LoanApplication
	query := loanSelect + ` WHERE la.id = $1`

	if err := r.db.GetContext(ctx, &loan, query, id); err != nil {
		return nil, err
	}
	return &loan, nil
}

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
