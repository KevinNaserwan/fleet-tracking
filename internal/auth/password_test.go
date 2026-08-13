package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	params := PasswordParams{
		Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	}
	encoded, err := HashPassword("correct horse battery staple", params)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	valid, err := VerifyPassword("correct horse battery staple", encoded)
	if err != nil || !valid {
		t.Fatalf("VerifyPassword(correct) = %v, %v", valid, err)
	}
	valid, err = VerifyPassword("wrong password", encoded)
	if err != nil || valid {
		t.Fatalf("VerifyPassword(wrong) = %v, %v", valid, err)
	}
}

func TestVerifyPasswordRejectsUnboundedHash(t *testing.T) {
	_, err := VerifyPassword(
		"password",
		"$argon2id$v=19$m=999999999,t=1,p=1$c2FsdHNhbHQ$AAAAAAAAAAAAAAAAAAAAAA",
	)
	if err == nil {
		t.Fatal("VerifyPassword() accepted unsafe memory cost")
	}
}
