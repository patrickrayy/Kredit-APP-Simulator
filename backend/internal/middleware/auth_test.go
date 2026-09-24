package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"loanapp/internal/auth"
)

func TestAuthRequired_RejectMissingCookies(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	AuthRequired(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if handlerCalled {
		t.Error("next handler should not be called without a token cookie")
	}
}

func TestAuthRequired_AllowsValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long")
	token, err := auth.GenerateToken("user-1", "nasabah")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	var gotUser AuthUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	rec := httptest.NewRecorder()

	AuthRequired(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUser.UserID != "user-1" || gotUser.Role != "nasabah" {
		t.Errorf("context user = %+v, want UserID=user-1, Role=nasabah", gotUser)
	}
}

func TestRequireRole_RejectsWrongRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(contextWithUser(req.Context(), AuthUser{UserID: "user-1", Role: "nasabah"}))
	rec := httptest.NewRecorder()

	RequireRole("petugas")(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
