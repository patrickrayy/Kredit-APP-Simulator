package model

import "time"

type LoanApplication struct {
	ID            string    `db:"id" json:"id"`
	UserID        string    `db:"user_id" json:"user_id"`
	ApplicantName string    `db:"applicant_name" json:"applicant_name,omitempty"`
	Amount        int64     `db:"amount" json:"amount"`
	Purpose       string    `db:"purpose" json:"purpose"`
	Status        string    `db:"status" json:"status"`
	ReviewedBy    *string   `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewNote    *string   `db:"review_note" json:"review_note,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}
