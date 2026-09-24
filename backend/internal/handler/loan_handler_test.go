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

func TestValidateReviewRequest(t *testing.T) {
	note := "Dokumen lengkap"
	blank := "   "

	tests := []struct {
		name    string
		req     reviewRequest
		wantErr bool
	}{
		{"approve without note", reviewRequest{Status: "approved"}, false},
		{"approve with note", reviewRequest{Status: "approved", ReviewNote: &note}, false},
		{"reject with note", reviewRequest{Status: "rejected", ReviewNote: &note}, false},
		{"reject without note", reviewRequest{Status: "rejected"}, true},
		{"reject with blank note", reviewRequest{Status: "rejected", ReviewNote: &blank}, true},
		{"invalid status", reviewRequest{Status: "ok"}, true},
		{"pending is not a review decision", reviewRequest{Status: "pending"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReviewRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateReviewRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
