package passwordhash_test

import (
	"testing"

	"prasankit-api/pkg/passwordhash"
)

// bcrypt is intentionally slow; keep test cases minimal.
func TestHash_DiffersFromPlaintext(t *testing.T) {
	const pw = "hunter2"
	h, err := passwordhash.Hash(pw)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if h == pw {
		t.Errorf("hash equals plaintext — that is wrong")
	}
}

func TestVerify_CorrectPassword(t *testing.T) {
	const pw = "correct-horse-battery-staple"
	h, err := passwordhash.Hash(pw)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if err := passwordhash.Verify(h, pw); err != nil {
		t.Errorf("Verify(correct) = %v, want nil", err)
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	const pw = "correct-horse-battery-staple"
	h, err := passwordhash.Hash(pw)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if err := passwordhash.Verify(h, "wrong-password"); err == nil {
		t.Errorf("Verify(wrong) = nil, want error")
	}
}
