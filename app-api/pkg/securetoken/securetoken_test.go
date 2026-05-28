package securetoken_test

import (
	"encoding/base64"
	"testing"

	"prasankit-api/pkg/securetoken"
)

func TestNew_RawDecodesToThirtyTwoBytes(t *testing.T) {
	raw, _, err := securetoken.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("DecodeString(%q) error = %v", raw, err)
	}
	if len(b) != 32 {
		t.Errorf("decoded length = %d, want 32", len(b))
	}
}

func TestNew_HashMatchesHashFunction(t *testing.T) {
	raw, hash, err := securetoken.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := securetoken.Hash(raw); got != hash {
		t.Errorf("Hash(raw) = %q, want %q", got, hash)
	}
}

func TestHash_IsDeterministic(t *testing.T) {
	const input = "test-token-value"
	a := securetoken.Hash(input)
	b := securetoken.Hash(input)
	if a != b {
		t.Errorf("Hash not deterministic: %q != %q", a, b)
	}
}

func TestNew_TwoCallsProduceDifferentRawAndHash(t *testing.T) {
	raw1, hash1, err := securetoken.New()
	if err != nil {
		t.Fatalf("first New() error = %v", err)
	}
	raw2, hash2, err := securetoken.New()
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}
	if raw1 == raw2 {
		t.Errorf("two New() calls returned equal raw tokens")
	}
	if hash1 == hash2 {
		t.Errorf("two New() calls returned equal hashes")
	}
}
