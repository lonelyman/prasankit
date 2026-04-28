package securetoken

import "testing"

func TestNew(t *testing.T) {
	token, hash, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if token == hash {
		t.Fatal("hash must not equal token")
	}
	if Hash(token) != hash {
		t.Fatal("hash does not match token")
	}
}
