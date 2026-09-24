package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	mw "loanapp/internal/middleware"
	"loanapp/internal/model"
	"loanapp/internal/repository"
)

type LoanHandler struct {
	loans *repository.LoanRepository
}

func NewLoanHandler(loans *repository.LoanRepository) *LoanHandler {
	return &LoanHandler{loans: loans}
}

type createLoanRequest struct {
	Amount  int64  `json:"amount"`
	Purpose string `json:"purpose"`
}

func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createLoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Purpose = strings.TrimSpace(req.Purpose)

	if req.Amount <= 0 || req.Purpose == "" {
		http.Error(w, "amount must be positive and purpose is required", http.StatusBadRequest)
		return
	}

	loan, err := h.loans.Create(r.Context(), user.UserID, req.Amount, req.Purpose)
	if err != nil {
		http.Error(w, "failed to create loan application", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loan)
}

func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var (
		loans []model.LoanApplication
		err   error
	)
	if user.Role == "petugas" {
		loans, err = h.loans.ListAll(r.Context())
	} else {
		loans, err = h.loans.ListByUser(r.Context(), user.UserID)
	}
	if err != nil {
		http.Error(w, "failed to list loan applications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loans)
}
