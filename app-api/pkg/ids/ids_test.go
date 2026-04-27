package ids

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUUIDReturnsVersion7(t *testing.T) {
	id, err := NewUUID()
	if err != nil {
		t.Fatalf("NewUUID: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("NewUUID returned nil UUID")
	}
	if id.Version() != 7 {
		t.Fatalf("UUID version = %d, want 7", id.Version())
	}
}
