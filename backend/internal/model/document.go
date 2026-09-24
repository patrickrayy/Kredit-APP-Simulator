package model

import "time"

type LoanDocument struct {
	ID                string    `db:"id" json:"id"`
	LoanApplicationID string    `db:"loan_application_id" json:"loan_application_id"`
	DocType           string    `db:"doc_type" json:"doc_type"`
	FilePath          string    `db:"file_path" json:"file_path"`
	UploadedAt        time.Time `db:"uploaded_at" json:"uploaded_at"`
}
