package passwordhash

import "testing"

func TestBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher(4)

	hash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash == "" {
		t.Fatal("hash was empty")
	}
	if !IsBcryptHash(hash) {
		t.Fatal("hash is not a bcrypt hash")
	}
	if err := hasher.Compare(hash, "correct-password"); err != nil {
		t.Fatalf("Compare correct password: %v", err)
	}
	if err := hasher.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("Compare wrong password returned nil")
	}
}
