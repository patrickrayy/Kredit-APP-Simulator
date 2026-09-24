package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-that-is-at-least-32-chars-long"

func TestGenerateAndVerifyToken_RoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	tokenString, err := GenerateToken("user-123", "nasabah")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := VerifyToken(tokenString)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Role != "nasabah" {
		t.Errorf("Role = %q, want %q", claims.Role, "nasabah")
	}
}

func TestVerifyToken_RejectsGarbageToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	_, err := VerifyToken("not-a-real-token")
	if err == nil {
		t.Fatal("expected error for garbage token, got nil")
	}
}

func TestVerifyToken_RejectsUnexpectedAlgorithm(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	// Token ditandatangani dengan secret yang BENAR, tapi algoritma HS512.
	// Server kita hanya membuat token HS256, jadi token ini harus ditolak.
	claims := Claims{
		UserID: "user-1",
		Role:   "petugas",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	if _, err := VerifyToken(tokenString); err == nil {
		t.Fatal("expected HS512 token to be rejected, got nil error")
	}
}

func TestCheckSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"short", "dev-secret-change-me", true},
		{"31 chars", strings.Repeat("a", 31), true},
		{"32 chars", strings.Repeat("a", 32), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.secret)
			if err := CheckSecret(); (err != nil) != tt.wantErr {
				t.Errorf("CheckSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
