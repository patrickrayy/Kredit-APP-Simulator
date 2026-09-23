package auth

import "testing"

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash, err := HashPassword("s3cretPassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "s3cretPassword" {
		t.Fatal("hash must not equal plaintext")
	}
	if !CheckPassword("s3cretPassword", hash) {
		t.Fatal("CheckPassword should succeed with correct password")
	}
}

func TestCheckPassword_RejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("s3cretPassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if CheckPassword("wrongPassword", hash) {
		t.Fatal("CheckPassword should fail with wrong password")
	}
}
