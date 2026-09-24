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
