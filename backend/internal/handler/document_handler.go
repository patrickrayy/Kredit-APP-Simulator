package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	mw "loanapp/internal/middleware"
	"loanapp/internal/repository"
)

const maxUploadSize = 10 << 20 // 10 MB

var validDocTypes = map[string]bool{
	"ktp":            true,
	"npwp":           true,
	"slip_gaji":      true,
	"rekening_koran": true,
	"lainnya":        true,
}

func isValidDocType(docType string) bool {
	return validDocTypes[docType]
}

// safeExt mengambil ekstensi dari nama file asli, hanya kalau isinya huruf
// kecil/angka dan panjangnya wajar. Selain itu dikembalikan "".
func safeExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) < 2 || len(ext) > 10 {
		return ""
	}
	for _, c := range ext[1:] {
		if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') {
			return ""
		}
	}
	return ext
}

// storedFileName membuat nama acak untuk disimpan di disk. Nama asli dari
// client tidak pernah dipakai (cegah path traversal), hanya ekstensinya.
func storedFileName(original string) string {
	return rand.Text() + safeExt(original)
}

type DocumentHandler struct {
	loans     *repository.LoanRepository
	documents *repository.DocumentRepository
	uploadDir string
}

func NewDocumentHandler(loans *repository.LoanRepository, documents *repository.DocumentRepository, uploadDir string) *DocumentHandler {
	return &DocumentHandler{loans: loans, documents: documents, uploadDir: uploadDir}
}

func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	loan, err := h.loans.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if isLoanNotFound(err) {
			http.Error(w, "loan application not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get loan application", http.StatusInternalServerError)
		return
	}
	// Nasabah lain: 404 supaya tidak membocorkan bahwa loan ini ada.
	if !canAccessLoan(user, loan) {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}
	// Petugas boleh melihat loan, tapi yang upload dokumen hanya pemohon.
	if loan.UserID != user.UserID {
		http.Error(w, "only the applicant can upload documents", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "file too large (max 10 MB)", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	docType := strings.ToLower(strings.TrimSpace(r.FormValue("doc_type")))
	if !isValidDocType(docType) {
		http.Error(w, "doc_type must be one of: ktp, npwp, slip_gaji, rekening_koran, lainnya", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Pakai loan.ID dari DB (bukan dari URL) untuk membangun path di disk.
	loanDir := filepath.Join(h.uploadDir, loan.ID)
	if err := os.MkdirAll(loanDir, 0o755); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	storedName := storedFileName(header.Filename)
	fullPath := filepath.Join(loanDir, storedName)
	relativePath := filepath.ToSlash(filepath.Join(loan.ID, storedName))

	dst, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	_, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(fullPath)
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	doc, err := h.documents.Create(r.Context(), loan.ID, docType, relativePath)
	if err != nil {
		// Jangan tinggalkan file yatim di disk kalau pencatatan di DB gagal.
		os.Remove(fullPath)
		http.Error(w, "failed to record document", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	loan, err := h.loans.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if isLoanNotFound(err) {
			http.Error(w, "loan application not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get loan application", http.StatusInternalServerError)
		return
	}
	if !canAccessLoan(user, loan) {
		http.Error(w, "loan application not found", http.StatusNotFound)
		return
	}

	docs, err := h.documents.ListByLoan(r.Context(), loan.ID)
	if err != nil {
		http.Error(w, "failed to list documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}
