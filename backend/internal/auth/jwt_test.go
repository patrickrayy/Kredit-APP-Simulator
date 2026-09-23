package auth

import "testing"

func TestGenerateAndVerifyToken_RoundTrip(t *testing.T) {
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
	_, err := VerifyToken("not-a-real-token")
	if err == nil {
		t.Fatal("expected error for garbage token, got nil")
	}
}
