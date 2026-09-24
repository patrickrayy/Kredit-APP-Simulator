package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	mw "loanapp/internal/middleware"
	"loanapp/internal/model"
	"loanapp/internal/repository"
)

type LoanHandler struct {
	loans *repository.LoanRepository
}

type reviewRequest struct {
	Status     string  `json:"status"`
	ReviewNote *string `json:"review_note"`
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

func isLoanNotFound(err error) bool {
	var pgErr *pgconn.PgError
	return errors.Is(err, sql.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02")
}

func canAccessLoan(user mw.AuthUser, loan *model.LoanApplication) bool {
	return user.Role == "petugas" || loan.UserID == user.UserID
}

func (h *LoanHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	loan, err := h.loans.GetByID(r.Context(), id)
	if err != nil {
		if isLoanNotFound(err) {
			http.Error(w, "loan application not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get loan application", http.StatusInternalServerError)
		return
	}

	// Sengaja 404 (bukan 403) supaya tidak membocorkan bahwa loan ini ada.
	if !canAccessLoan(user, loan) {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loan)
}
func validateReviewRequest(req reviewRequest) error {
	if req.Status != "approved" && req.Status != "rejected" {
		return errors.New(`status must be "approved" or "rejected"`)
	}
	if req.Status == "rejected" && (req.ReviewNote == nil || strings.TrimSpace(*req.ReviewNote) == "") {
		return errors.New("review_note is required when rejecting")
	}
	return nil
}
func (h *LoanHandler) Review(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req reviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := validateReviewRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simpan note yang sudah di-trim; note kosong disimpan sebagai NULL.
	if req.ReviewNote != nil {
		trimmed := strings.TrimSpace(*req.ReviewNote)
		if trimmed == "" {
			req.ReviewNote = nil
		} else {
			req.ReviewNote = &trimmed
		}
	}

	id := chi.URLParam(r, "id")
	existing, err := h.loans.GetByID(r.Context(), id)
	if err != nil {
		if isLoanNotFound(err) {
			http.Error(w, "loan application not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get loan application", http.StatusInternalServerError)
		return
	}

	// Maker-checker: petugas tidak boleh me-review pengajuannya sendiri.
	if existing.UserID == user.UserID {
		http.Error(w, "cannot review your own loan application", http.StatusForbidden)
		return
	}

	loan, err := h.loans.Review(r.Context(), id, user.UserID, req.Status, req.ReviewNote)
	if err != nil {
		if isLoanNotFound(err) {
			http.Error(w, "loan application not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to review loan application", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loan)
}
