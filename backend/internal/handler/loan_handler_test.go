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
